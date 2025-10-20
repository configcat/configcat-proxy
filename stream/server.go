package stream

import (
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/pubsub"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/puzpuzpuz/xsync/v3"
)

type Server interface {
	GetStreamOrNil(sdkId string) Stream
	GetStreamBySdkKeyOrNil(sdkKey string) Stream
	Close()
}

type server struct {
	streams           *xsync.MapOf[string, Stream]
	streamsBySdkKey   *xsync.MapOf[string, Stream]
	log               log.Logger
	sdkRegistrar      sdk.Registrar
	telemetryReporter telemetry.Reporter
	serverType        string
	sdkChanged        chan string
	stop              chan struct{}
}

func NewServer(sdkRegistrar sdk.Registrar, telemetryReporter telemetry.Reporter, log log.Logger, serverType string) Server {
	strLog := log.WithPrefix("stream-server")
	streams := xsync.NewMapOf[string, Stream]()
	streamsBySdkKey := xsync.NewMapOf[string, Stream]()
	for id, sdkClient := range sdkRegistrar.GetAll() {
		str := NewStream(id, sdkClient, telemetryReporter, strLog, serverType)
		streams.Store(id, str)
		key1, key2 := str.SdkKeys()
		if key1 != "" {
			streamsBySdkKey.Store(key1, str)
		}
		if key2 != nil && len(*key2) > 0 {
			streamsBySdkKey.Store(*key2, str)
		}
	}
	srv := &server{
		log:               strLog,
		streams:           streams,
		streamsBySdkKey:   streamsBySdkKey,
		sdkRegistrar:      sdkRegistrar,
		telemetryReporter: telemetryReporter,
		serverType:        serverType,
		stop:              make(chan struct{}),
	}
	if autoRegistrar, ok := sdkRegistrar.(pubsub.SubscriptionHandler[string]); ok {
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
		str, loaded := s.streams.LoadOrCompute(sdkId, func() Stream {
			return NewStream(sdkId, sdkClient, s.telemetryReporter, s.log, s.serverType)
		})
		if loaded {
			key1, key2 := str.SdkKeys()
			if key1 != "" {
				s.streamsBySdkKey.Delete(key1)
			}
			if key2 != nil && *key2 != "" {
				s.streamsBySdkKey.Delete(*key2)
			}
			str.ResetSdk(sdkClient)
		}
		key1, key2 := sdkClient.SdkKeys()
		if key1 != "" {
			s.streamsBySdkKey.Store(key1, str)
		}
		if key2 != nil && *key2 != "" {
			s.streamsBySdkKey.Store(*key2, str)
		}
	} else {
		if existing, loaded := s.streams.LoadAndDelete(sdkId); loaded {
			key1, key2 := existing.SdkKeys()
			if key1 != "" {
				s.streamsBySdkKey.Delete(key1)
			}
			if key2 != nil && len(*key2) > 0 {
				s.streamsBySdkKey.Delete(*key2)
			}
			existing.Close()
		}
	}
}

func (s *server) GetStreamOrNil(sdkId string) Stream {
	str, _ := s.streams.Load(sdkId)
	return str
}

func (s *server) GetStreamBySdkKeyOrNil(sdkKey string) Stream {
	str, _ := s.streamsBySdkKey.Load(sdkKey)
	return str
}

func (s *server) Close() {
	close(s.stop)
	if autoRegistrar, ok := s.sdkRegistrar.(pubsub.SubscriptionHandler[string]); ok {
		autoRegistrar.Unsubscribe(s.sdkChanged)
	}
	s.streams.Range(func(key string, value Stream) bool {
		value.Close()
		s.streams.Delete(key)
		return true
	})
	s.streamsBySdkKey.Clear()
	s.log.Reportf("shutdown complete")
}
