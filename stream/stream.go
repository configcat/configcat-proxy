package stream

import (
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/sdk"
	"hash/maphash"
)

type Stream interface {
	CreateSingleFlagConnection(key string, user sdk.UserAttrs) *Connection
	CreateAllFlagsConnection(user sdk.UserAttrs) *Connection
	CloseSingleFlagConnection(conn *Connection, key string)
	CloseAllFlagsConnection(conn *Connection)
	Close()
}

type connEstablished struct {
	conn *Connection
	user sdk.UserAttrs
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
	sdkId            string
	metrics          metrics.Handler
	channels         map[string]map[uint64]channel
	connEstablished  chan *connEstablished
	connClosed       chan *connClosed
	connCount        int64
	seed             maphash.Seed
}

func NewStream(sdkId string, sdkClient sdk.Client, metrics metrics.Handler, log log.Logger, serverType string) Stream {
	s := &stream{
		channels:         make(map[string]map[uint64]channel),
		connEstablished:  make(chan *connEstablished),
		connClosed:       make(chan *connClosed),
		stop:             make(chan struct{}),
		sdkConfigChanged: sdkClient.SubConfigChanged(serverType + sdkId),
		sdkClient:        sdkClient,
		log:              log.WithPrefix("stream-" + sdkId),
		serverType:       serverType,
		sdkId:            sdkId,
		metrics:          metrics,
		seed:             maphash.MakeSeed(),
	}
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

func (s *stream) CreateSingleFlagConnection(key string, user sdk.UserAttrs) *Connection {
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

func (s *stream) CreateAllFlagsConnection(user sdk.UserAttrs) *Connection {
	var discriminator uint64
	if user != nil {
		discriminator = user.Discriminator(s.seed)
	}
	conn := newConnection(discriminator)
	select {
	case <-s.stop:
		return conn
	default:
		s.connEstablished <- &connEstablished{conn: conn, user: user, key: allFlagsDiscriminator}
		return conn
	}
}

func (s *stream) CloseSingleFlagConnection(conn *Connection, key string) {
	select {
	case <-s.stop:
		return
	default:
		s.connClosed <- &connClosed{conn: conn, key: key}
	}
}

func (s *stream) CloseAllFlagsConnection(conn *Connection) {
	select {
	case <-s.stop:
		return
	default:
		s.connClosed <- &connClosed{conn: conn, key: allFlagsDiscriminator}
	}
}

func (s *stream) Close() {
	close(s.stop)
	s.sdkClient.UnsubConfigChanged(s.serverType + s.sdkId)
	s.log.Reportf("shutdown complete")
}

func (s *stream) addConnection(established *connEstablished) {
	bucket, ok := s.channels[established.key]
	if !ok {
		ch := createChannel(established, s.sdkClient)
		bucket = map[uint64]channel{established.conn.discriminator: ch}
		s.channels[established.key] = bucket
	}
	ch, ok := bucket[established.conn.discriminator]
	if !ok {
		ch = createChannel(established, s.sdkClient)
		bucket[established.conn.discriminator] = ch
	}
	ch.AddConnection(established.conn)
	established.conn.receive <- ch.LastPayload()
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
			sent += ch.Notify(s.sdkClient, key)
		}
	}
	if sent > 0 {
		s.log.Debugf("payload sent to %d connection(s)", sent)
	}
}
