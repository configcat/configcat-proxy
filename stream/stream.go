package stream

import (
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	"sync"
	"sync/atomic"
)

type Stream interface {
	CreateConnection(user *sdk.UserAttrs) Connection
}

var defaultConnectionDiscriminator = "no-user"

type channel struct {
	connections []*connection
	lastPayload atomic.Pointer[model.ResponsePayload]
	user        *sdk.UserAttrs
}

type connEstablished struct {
	connection *connection
	user       *sdk.UserAttrs
}

type stream struct {
	key                   string
	channels              map[string]*channel
	connectionEstablished chan *connEstablished
	connectionClosed      chan *connection
	connectionCount       int64
	closed                chan struct{}
	closedOnce            sync.Once
	markedForClose        atomic.Bool
	log                   log.Logger
	sdkClient             sdk.Client
	sdkConfigChanged      <-chan struct{}
	metrics               metrics.Handler
	serverType            string
}

func newStream(key string, serverType string, sdkClient sdk.Client, metrics metrics.Handler, log log.Logger) *stream {
	streamLog := log.WithPrefix("stream")
	s := &stream{
		key:                   key,
		channels:              make(map[string]*channel),
		connectionEstablished: make(chan *connEstablished),
		connectionClosed:      make(chan *connection),
		closed:                make(chan struct{}),
		log:                   streamLog,
		sdkClient:             sdkClient,
		sdkConfigChanged:      sdkClient.SubConfigChanged(serverType + key),
		metrics:               metrics,
		serverType:            serverType,
	}
	s.run()
	streamLog.Debugf("stream created (#%s)", key)
	return s
}

func (s *stream) run() {
	go func(str *stream) {
		for {
			select {
			case established := <-str.connectionEstablished:
				str.addConnection(established)
				str.log.Debugf("#%s: connection established, all connections: %d", str.key, atomic.AddInt64(&str.connectionCount, 1))
				if str.metrics != nil {
					str.metrics.IncrementConnection(str.serverType, str.key)
				}

			case connection := <-str.connectionClosed:
				str.removeConnection(connection)
				str.log.Debugf("#%s: connection closed, all connections: %d", str.key, atomic.AddInt64(&str.connectionCount, -1))
				if str.metrics != nil {
					str.metrics.DecrementConnection(str.serverType, str.key)
				}

			case <-str.sdkConfigChanged:
				str.log.Debugf("#%s: sending payload to %d connection(s)", str.key, atomic.LoadInt64(&str.connectionCount))
				str.notifyConnections()

			case <-str.closed:
				str.tearDown()
				str.log.Infof("#%s: stream closed", str.key)
				return
			}
		}
	}(s)
}

func (s *stream) CreateConnection(user *sdk.UserAttrs) Connection {
	var discriminator = ""
	if user != nil {
		discriminator = user.Discriminator()
	}
	connection := newConnection(s.connectionClosed, discriminator)
	s.connectionEstablished <- &connEstablished{connection: connection, user: user}
	return connection
}

func (s *stream) close() {
	s.closedOnce.Do(func() {
		close(s.closed)
	})
}

func (s *stream) addConnection(established *connEstablished) {
	var discriminator = defaultConnectionDiscriminator
	if established.connection.discriminator != "" {
		discriminator = established.connection.discriminator
	}
	ch, ok := s.channels[discriminator]
	if !ok {
		val, _ := s.sdkClient.Eval(s.key, established.user)
		ch = &channel{user: established.user}
		payload := model.PayloadFromEvalData(&val)
		ch.lastPayload.Store(&payload)
		s.channels[discriminator] = ch
	}
	ch.connections = append(ch.connections, established.connection)
	payload := ch.lastPayload.Load()
	established.connection.receive <- payload
}

func (s *stream) removeConnection(connection *connection) {
	var discriminator = defaultConnectionDiscriminator
	if connection.discriminator != "" {
		discriminator = connection.discriminator
	}
	ch, ok := s.channels[discriminator]
	if !ok {
		return
	}
	index := -1
	for i := range ch.connections {
		if ch.connections[i] == connection {
			index = i
			break
		}
	}
	if index != -1 {
		ch.connections = append(ch.connections[:index], ch.connections[index+1:]...)
	}
	if len(ch.connections) == 0 {
		delete(s.channels, discriminator)
	}
	if len(s.channels) == 0 {
		s.markedForClose.Store(true)
	}
}

func (s *stream) notifyConnections() {
	for _, b := range s.channels {
		val, err := s.sdkClient.Eval(s.key, b.user)
		if err != nil {
			continue
		}
		payload := model.PayloadFromEvalData(&val)
		b.lastPayload.Store(&payload)
		for _, conn := range b.connections {
			conn.receive <- &payload
		}
	}
}

func (s *stream) tearDown() {
	s.sdkClient.UnsubConfigChanged(s.key)
	for id, b := range s.channels {
		for i := 0; i < len(b.connections); i++ {
			close(b.connections[i].receive)
		}
		delete(s.channels, id)
	}
}
