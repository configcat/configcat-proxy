package stream

import (
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	"sync/atomic"
)

type Server interface {
	CreateConnection(key string, user *sdk.UserAttrs) Connection
	CloseConnection(conn Connection, key string)
	Close()
}

type channel struct {
	connections []Connection
	lastPayload *model.ResponsePayload
	user        *sdk.UserAttrs
	key         string
}

type connEstablished struct {
	conn Connection
	user *sdk.UserAttrs
	key  string
}

type connClosed struct {
	conn Connection
	key  string
}

type server struct {
	sdkClient        sdk.Client
	sdkConfigChanged <-chan struct{}
	stop             chan struct{}
	log              log.Logger
	serverType       string
	metrics          metrics.Handler
	channels         map[string]*channel
	connEstablished  chan *connEstablished
	connClosed       chan *connClosed
	connCount        int64
}

func NewServer(sdkClient sdk.Client, metrics metrics.Handler, log log.Logger, serverType string) Server {
	s := &server{
		channels:         make(map[string]*channel),
		connEstablished:  make(chan *connEstablished),
		connClosed:       make(chan *connClosed),
		stop:             make(chan struct{}),
		sdkConfigChanged: sdkClient.SubConfigChanged(serverType),
		sdkClient:        sdkClient,
		log:              log.WithPrefix("stream-server"),
		serverType:       serverType,
		metrics:          metrics,
	}
	s.run()
	return s
}

func (s *server) run() {
	go func() {
		for {
			select {
			case established := <-s.connEstablished:
				s.addConnection(established)
				s.log.Debugf("connection established, all connections: %d", atomic.AddInt64(&s.connCount, 1))
				if s.metrics != nil {
					s.metrics.IncrementConnection(s.serverType, established.key)
				}

			case closed := <-s.connClosed:
				s.removeConnection(closed)
				s.log.Debugf("connection closed, all connections: %d", atomic.AddInt64(&s.connCount, -1))
				if s.metrics != nil {
					s.metrics.DecrementConnection(s.serverType, closed.key)
				}

			case <-s.sdkConfigChanged:
				s.log.Debugf("sending payload to %d connection(s)", atomic.LoadInt64(&s.connCount))
				s.notifyConnections()

			case <-s.stop:
				return
			}
		}
	}()
}

func (s *server) CreateConnection(key string, user *sdk.UserAttrs) Connection {
	var extra = ""
	if user != nil {
		extra = user.Discriminator()
	}
	conn := newConnection(extra)
	s.connEstablished <- &connEstablished{conn: conn, user: user, key: key}
	return conn
}

func (s *server) CloseConnection(conn Connection, key string) {
	s.connClosed <- &connClosed{conn: conn, key: key}
}

func (s *server) Close() {
	s.sdkClient.UnsubConfigChanged(s.serverType)
	close(s.stop)
	s.log.Reportf("shutdown complete")
}

func (s *server) addConnection(established *connEstablished) {
	id := established.key + established.conn.GetExtraAttrs()
	ch, ok := s.channels[id]
	if !ok {
		val, _ := s.sdkClient.Eval(established.key, established.user)
		ch = &channel{user: established.user, key: established.key}
		payload := model.PayloadFromEvalData(&val)
		ch.lastPayload = &payload
		s.channels[id] = ch
	}
	ch.connections = append(ch.connections, established.conn)
	established.conn.Publish(ch.lastPayload)
}

func (s *server) removeConnection(closed *connClosed) {
	id := closed.key + closed.conn.GetExtraAttrs()
	ch, ok := s.channels[id]
	if !ok {
		return
	}
	index := -1
	for i := range ch.connections {
		if ch.connections[i] == closed.conn {
			index = i
			break
		}
	}
	if index != -1 {
		ch.connections = append(ch.connections[:index], ch.connections[index+1:]...)
	}
	if len(ch.connections) == 0 {
		delete(s.channels, id)
	}
}

func (s *server) notifyConnections() {
	for _, ch := range s.channels {
		val, err := s.sdkClient.Eval(ch.key, ch.user)
		if err != nil {
			continue
		}
		if val.Value != ch.lastPayload.Value {
			payload := model.PayloadFromEvalData(&val)
			ch.lastPayload = &payload
			for _, conn := range ch.connections {
				conn.Publish(&payload)
			}
		}
	}
}
