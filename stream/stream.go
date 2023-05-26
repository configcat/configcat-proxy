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
	sdkClient                 sdk.Client
	sdkConfigChanged          <-chan struct{}
	stop                      chan struct{}
	log                       log.Logger
	serverType                string
	sdkId                     string
	metrics                   metrics.Handler
	singleFlagChannels        map[string]map[uint64]*singleFlagChannel
	allFlagChannels           map[uint64]*allFlagsChannel
	singleFlagConnEstablished chan *connEstablished
	singleFlagConnClosed      chan *connClosed
	allFlagsConnEstablished   chan *connEstablished
	allFlagsConnClosed        chan *connClosed
	connCount                 int64
	seed                      maphash.Seed
}

func NewStream(sdkId string, sdkClient sdk.Client, metrics metrics.Handler, log log.Logger, serverType string) Stream {
	s := &stream{
		singleFlagChannels:        make(map[string]map[uint64]*singleFlagChannel),
		allFlagChannels:           make(map[uint64]*allFlagsChannel),
		singleFlagConnEstablished: make(chan *connEstablished),
		singleFlagConnClosed:      make(chan *connClosed),
		allFlagsConnEstablished:   make(chan *connEstablished),
		allFlagsConnClosed:        make(chan *connClosed),
		stop:                      make(chan struct{}),
		sdkConfigChanged:          sdkClient.SubConfigChanged(serverType + sdkId),
		sdkClient:                 sdkClient,
		log:                       log.WithPrefix("stream-" + sdkId),
		serverType:                serverType,
		sdkId:                     sdkId,
		metrics:                   metrics,
		seed:                      maphash.MakeSeed(),
	}
	go s.run()
	return s
}

func (s *stream) run() {
	for {
		select {
		case established := <-s.singleFlagConnEstablished:
			s.addSingleFlagConnection(established)
			s.connCount++
			s.log.Debugf("#%s: connection established, all connections: %d", established.key, s.connCount)
			if s.metrics != nil {
				s.metrics.IncrementConnection(s.sdkId, s.serverType, established.key)
			}

		case established := <-s.allFlagsConnEstablished:
			s.addAllFlagsConnection(established)
			s.connCount++
			s.log.Debugf("#%s: connection established, all connections: %d", established.key, s.connCount)
			if s.metrics != nil {
				s.metrics.IncrementConnection(s.sdkId, s.serverType, established.key)
			}

		case closed := <-s.singleFlagConnClosed:
			s.removeSingleFlagConnection(closed)
			s.connCount--
			s.log.Debugf("#%s: connection closed, all connections: %d", closed.key, s.connCount)
			if s.metrics != nil {
				s.metrics.DecrementConnection(s.sdkId, s.serverType, closed.key)
			}

		case closed := <-s.allFlagsConnClosed:
			s.removeAllFlagsConnection(closed)
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
		s.singleFlagConnEstablished <- &connEstablished{conn: conn, user: user, key: key}
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
		s.allFlagsConnEstablished <- &connEstablished{conn: conn, user: user, key: allFlagsDiscriminator}
		return conn
	}
}

func (s *stream) CloseSingleFlagConnection(conn *Connection, key string) {
	select {
	case <-s.stop:
		return
	default:
		s.singleFlagConnClosed <- &connClosed{conn: conn, key: key}
	}
}

func (s *stream) CloseAllFlagsConnection(conn *Connection) {
	select {
	case <-s.stop:
		return
	default:
		s.allFlagsConnClosed <- &connClosed{conn: conn, key: allFlagsDiscriminator}
	}
}

func (s *stream) Close() {
	close(s.stop)
	s.sdkClient.UnsubConfigChanged(s.serverType + s.sdkId)
	s.log.Reportf("shutdown complete")
}

func (s *stream) addSingleFlagConnection(established *connEstablished) {
	bucket, ok := s.singleFlagChannels[established.key]
	if !ok {
		ch := createSingleFlagChannel(established, s.sdkClient)
		bucket = map[uint64]*singleFlagChannel{established.conn.discriminator: ch}
		s.singleFlagChannels[established.key] = bucket
	}
	ch, ok := bucket[established.conn.discriminator]
	if !ok {
		ch = createSingleFlagChannel(established, s.sdkClient)
		bucket[established.conn.discriminator] = ch
	}
	ch.connections = append(ch.connections, established.conn)
	established.conn.receive <- ch.lastPayload
}

func (s *stream) addAllFlagsConnection(established *connEstablished) {
	ch, ok := s.allFlagChannels[established.conn.discriminator]
	if !ok {
		ch = createAllFlagsChannel(established, s.sdkClient)
		s.allFlagChannels[established.conn.discriminator] = ch
	}
	ch.connections = append(ch.connections, established.conn)
	established.conn.receive <- ch.lastPayload
}

func (s *stream) removeSingleFlagConnection(closed *connClosed) {
	bucket, ok := s.singleFlagChannels[closed.key]
	if !ok {
		return
	}
	ch, ok := bucket[closed.conn.discriminator]
	if !ok {
		return
	}
	ch.RemoveConnection(closed.conn)
	if len(ch.connections) == 0 {
		delete(bucket, closed.conn.discriminator)
	}
	if len(bucket) == 0 {
		delete(s.singleFlagChannels, closed.key)
	}
}

func (s *stream) removeAllFlagsConnection(closed *connClosed) {
	ch, ok := s.allFlagChannels[closed.conn.discriminator]
	if !ok {
		return
	}
	ch.RemoveConnection(closed.conn)
	if len(ch.connections) == 0 {
		delete(s.allFlagChannels, closed.conn.discriminator)
	}
}

func (s *stream) notifyConnections() {
	sent := 0
	for key, bucket := range s.singleFlagChannels {
		for _, ch := range bucket {
			sent += ch.Notify(s.sdkClient, key)
		}
	}
	for _, ch := range s.allFlagChannels {
		sent += ch.Notify(s.sdkClient)
	}
	if sent > 0 {
		s.log.Debugf("payload sent to %d connection(s)", sent)
	}
}
