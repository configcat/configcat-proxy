package stream

import (
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	"sync/atomic"
)

type Stream interface {
	CreateConnection(key string, user *sdk.UserAttrs) *Connection
	CloseConnection(conn *Connection, key string)
	Close()
}

type channel struct {
	connections []*Connection
	lastPayload *model.ResponsePayload
	user        *sdk.UserAttrs
	key         string
}

type connEstablished struct {
	conn *Connection
	user *sdk.UserAttrs
	key  string
}

type connClosed struct {
	conn *Connection
	key  string
}

type stream struct {
	sdkClient        sdk.Client
	sdkConfigChanged <-chan struct{}
	stop             chan struct{}
	log              log.Logger
	serverType       string
	envId            string
	metrics          metrics.Handler
	channels         map[string]*channel
	connEstablished  chan *connEstablished
	connClosed       chan *connClosed
	connCount        int64
}

func NewStream(envId string, sdkClient sdk.Client, metrics metrics.Handler, log log.Logger, serverType string) Stream {
	s := &stream{
		channels:         make(map[string]*channel),
		connEstablished:  make(chan *connEstablished),
		connClosed:       make(chan *connClosed),
		stop:             make(chan struct{}),
		sdkConfigChanged: sdkClient.SubConfigChanged(serverType + envId),
		sdkClient:        sdkClient,
		log:              log.WithPrefix("stream-%s" + envId),
		serverType:       serverType,
		envId:            envId,
		metrics:          metrics,
	}
	go s.run()
	return s
}

func (s *stream) run() {
	for {
		select {
		case established := <-s.connEstablished:
			s.addConnection(established)
			s.log.Debugf("connection established, all connections: %d", atomic.AddInt64(&s.connCount, 1))
			if s.metrics != nil {
				s.metrics.IncrementConnection(s.envId, s.serverType, established.key)
			}

		case closed := <-s.connClosed:
			s.removeConnection(closed)
			s.log.Debugf("connection closed, all connections: %d", atomic.AddInt64(&s.connCount, -1))
			if s.metrics != nil {
				s.metrics.DecrementConnection(s.envId, s.serverType, closed.key)
			}

		case <-s.sdkConfigChanged:
			s.notifyConnections()

		case <-s.stop:
			return
		}
	}
}

func (s *stream) CreateConnection(key string, user *sdk.UserAttrs) *Connection {
	var discriminator string
	if user != nil {
		discriminator = user.Discriminator()
	}
	conn := newConnection(discriminator)
	select {
	case <-s.stop:
		return conn
	default:
		s.connEstablished <- &connEstablished{conn: conn, user: user, key: key}
		return conn
	}
}

func (s *stream) CloseConnection(conn *Connection, key string) {
	select {
	case <-s.stop:
		return
	default:
		s.connClosed <- &connClosed{conn: conn, key: key}
	}
}

func (s *stream) Close() {
	close(s.stop)
	s.sdkClient.UnsubConfigChanged(s.serverType + s.envId)
	s.log.Reportf("shutdown complete")
}

func (s *stream) addConnection(established *connEstablished) {
	id := established.key + established.conn.discriminator
	ch, ok := s.channels[id]
	if !ok {
		val, _ := s.sdkClient.Eval(established.key, established.user)
		ch = &channel{user: established.user, key: established.key}
		payload := model.PayloadFromEvalData(&val)
		ch.lastPayload = &payload
		s.channels[id] = ch
	}
	ch.connections = append(ch.connections, established.conn)
	established.conn.receive <- ch.lastPayload
}

func (s *stream) removeConnection(closed *connClosed) {
	close(closed.conn.receive)
	id := closed.key + closed.conn.discriminator
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
		ch.connections[index] = nil
		ch.connections = append(ch.connections[:index], ch.connections[index+1:]...)
	}
	if len(ch.connections) == 0 {
		delete(s.channels, id)
	}
}

func (s *stream) notifyConnections() {
	sent := 0
	for _, ch := range s.channels {
		val, err := s.sdkClient.Eval(ch.key, ch.user)
		if err != nil {
			continue
		}
		if val.Value != ch.lastPayload.Value {
			payload := model.PayloadFromEvalData(&val)
			ch.lastPayload = &payload
			for _, conn := range ch.connections {
				sent++
				conn.receive <- &payload
			}
		}
	}
	if sent > 0 {
		s.log.Debugf("payload sent to %d connection(s)", sent)
	}
}
