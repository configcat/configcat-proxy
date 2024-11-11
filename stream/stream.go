package stream

import (
	"github.com/configcat/configcat-proxy/diag/metrics"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	"hash/maphash"
	"sync/atomic"
)

type Stream interface {
	CanEval(key string) bool
	IsInValidState() bool
	CreateConnection(key string, user model.UserAttrs) *Connection
	CloseConnection(conn *Connection, key string)
	ResetSdk(client sdk.Client)
	Close()
	Closed() <-chan struct{}
}

type connEstablished struct {
	conn *Connection
	user model.UserAttrs
	key  string
}

type connClosed struct {
	conn *Connection
	key  string
}

type stream struct {
	sdkClient        atomic.Value
	sdkConfigChanged chan struct{}
	stop             chan struct{}
	log              log.Logger
	serverType       string
	sdkId            string
	metrics          metrics.Reporter
	channels         map[string]map[uint64]channel
	connEstablished  chan *connEstablished
	connClosed       chan *connClosed
	connCount        int64
	seed             maphash.Seed
}

func NewStream(sdkId string, sdkClient sdk.Client, metrics metrics.Reporter, log log.Logger, serverType string) Stream {
	s := &stream{
		channels:         make(map[string]map[uint64]channel),
		connEstablished:  make(chan *connEstablished),
		connClosed:       make(chan *connClosed),
		stop:             make(chan struct{}),
		sdkConfigChanged: make(chan struct{}, 1),
		sdkClient:        atomic.Value{},
		log:              log.WithPrefix("stream-" + sdkId),
		serverType:       serverType,
		sdkId:            sdkId,
		metrics:          metrics,
		seed:             maphash.MakeSeed(),
	}
	s.sdkClient.Store(sdkClient)
	sdkClient.Subscribe(s.sdkConfigChanged)
	go s.run()
	return s
}

func (s *stream) run() {
	for {
		select {
		case established := <-s.connEstablished:
			s.addConnection(established)
			s.connCount++
			s.log.Debugf("#%s: connection established, all connections: %d", established.key, s.connCount)
			if s.metrics != nil {
				s.metrics.IncrementConnection(s.sdkId, s.serverType, established.key)
			}

		case closed := <-s.connClosed:
			s.removeConnection(closed)
			s.connCount--
			s.log.Debugf("#%s: connection closed, all connections: %d", closed.key, s.connCount)
			if s.metrics != nil {
				s.metrics.DecrementConnection(s.sdkId, s.serverType, closed.key)
			}

		case <-s.sdkConfigChanged:
			s.notifyConnections()

		case <-s.stop:
			return
		}
	}
}

func (s *stream) CanEval(key string) bool {
	keys := s.sdkClient.Load().(sdk.Client).Keys()
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

func (s *stream) IsInValidState() bool {
	return s.sdkClient.Load().(sdk.Client).IsInValidState()
}

func (s *stream) CreateConnection(key string, user model.UserAttrs) *Connection {
	var discriminator uint64
	if user != nil {
		discriminator = user.Discriminator(s.seed)
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

func (s *stream) ResetSdk(client sdk.Client) {
	select {
	case <-s.stop:
		return
	default:
		old := s.sdkClient.Swap(client).(sdk.Client)
		old.Unsubscribe(s.sdkConfigChanged)
		client.Subscribe(s.sdkConfigChanged)
	}
}

func (s *stream) Close() {
	close(s.stop)
	s.sdkClient.Load().(sdk.Client).Unsubscribe(s.sdkConfigChanged)
	s.log.Reportf("shutdown complete")
}

func (s *stream) Closed() <-chan struct{} {
	return s.stop
}

func (s *stream) addConnection(established *connEstablished) {
	bucket, ok := s.channels[established.key]
	if !ok {
		ch := createChannel(established, s.sdkClient.Load().(sdk.Client))
		bucket = map[uint64]channel{established.conn.discriminator: ch}
		s.channels[established.key] = bucket
	}
	ch, ok := bucket[established.conn.discriminator]
	if !ok {
		ch = createChannel(established, s.sdkClient.Load().(sdk.Client))
		bucket[established.conn.discriminator] = ch
	}
	ch.AddConnection(established.conn)
	established.conn.receive <- ch.LastPayload()
	if s.metrics != nil {
		s.metrics.AddSentMessageCount(1, s.sdkId, s.serverType, established.key)
	}
}

func (s *stream) removeConnection(closed *connClosed) {
	bucket, ok := s.channels[closed.key]
	if !ok {
		return
	}
	ch, ok := bucket[closed.conn.discriminator]
	if !ok {
		return
	}
	ch.RemoveConnection(closed.conn)
	if ch.IsEmpty() {
		delete(bucket, closed.conn.discriminator)
	}
	if len(bucket) == 0 {
		delete(s.channels, closed.key)
	}
}

func (s *stream) notifyConnections() {
	sent := 0
	for key, bucket := range s.channels {
		for _, ch := range bucket {
			count := ch.Notify(s.sdkClient.Load().(sdk.Client), key)
			sent += count
			if s.metrics != nil {
				s.metrics.AddSentMessageCount(count, s.sdkId, s.serverType, key)
			}
		}
	}
	if sent > 0 {
		s.log.Debugf("payload sent to %d connection(s)", sent)
	}
}
