package stream

import (
	"github.com/configcat/configcat-proxy/diag/metrics"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/puzpuzpuz/xsync/v3"
)

type Server interface {
	GetStreamOrNil(sdkId string) Stream
	Close()
}

type server struct {
	streams      *xsync.MapOf[string, Stream]
	log          log.Logger
	sdkRegistrar sdk.Registrar
	metrics      metrics.Reporter
	serverType   string
	sdkChanged   chan string
	stop         chan struct{}
}

func NewServer(sdkRegistrar sdk.Registrar, metrics metrics.Reporter, log log.Logger, serverType string) Server {
	strLog := log.WithPrefix("stream-server")
	streams := xsync.NewMapOf[string, Stream]()
	for id, sdkClient := range sdkRegistrar.GetAll() {
		streams.Store(id, NewStream(id, sdkClient, metrics, strLog, serverType))
	}
	srv := &server{
		log:          strLog,
		streams:      streams,
		sdkRegistrar: sdkRegistrar,
		metrics:      metrics,
		serverType:   serverType,
		stop:         make(chan struct{}),
	}
	if autoRegistrar, ok := sdkRegistrar.(sdk.AutoRegistrar); ok {
		srv.sdkChanged = make(chan string, 1)
		autoRegistrar.Subscribe(srv.sdkChanged)
		go srv.run()
	}
	return srv
}

func (s *server) run() {
	for {
		select {
		case sdkId := <-s.sdkChanged:
			s.handleSdkId(sdkId)
		case <-s.stop:
			return
		}
	}
}

func (s *server) handleSdkId(sdkId string) {
	sdkClient := s.sdkRegistrar.GetSdkOrNil(sdkId)
	if sdkClient != nil {
		s.streams.LoadOrCompute(sdkId, func() Stream {
			return NewStream(sdkId, sdkClient, s.metrics, s.log, s.serverType)
		})
	} else {
		if str, ok := s.streams.LoadAndDelete(sdkId); ok {
			str.Close()
		}
	}
}

func (s *server) GetStreamOrNil(sdkId string) Stream {
	str, _ := s.streams.Load(sdkId)
	return str
}

func (s *server) Close() {
	close(s.stop)
	if autoRegistrar, ok := s.sdkRegistrar.(sdk.AutoRegistrar); ok {
		autoRegistrar.Unsubscribe(s.sdkChanged)
	}
	s.streams.Range(func(key string, value Stream) bool {
		value.Close()
		s.streams.Delete(key)
		return true
	})
	s.log.Reportf("shutdown complete")
}
