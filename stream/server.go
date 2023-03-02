package stream

import (
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/sdk"
	"sync"
	"time"
)

type Server interface {
	GetOrCreateStream(key string) Stream
	Close()
}

type server struct {
	streams    map[string]*stream
	sdkClient  sdk.Client
	cleanup    *time.Ticker
	closed     chan struct{}
	closedOnce sync.Once
	log        log.Logger
	serverType string
	mu         sync.Mutex
	metrics    metrics.Handler
}

func NewServer(sdkClient sdk.Client, metrics metrics.Handler, log log.Logger, serverType string) Server {
	s := &server{
		streams:    make(map[string]*stream),
		cleanup:    time.NewTicker(30 * time.Second),
		closed:     make(chan struct{}),
		sdkClient:  sdkClient,
		log:        log.WithPrefix("stream-server"),
		serverType: serverType,
		metrics:    metrics,
	}
	s.run()
	return s
}

func (s *server) run() {
	go func() {
		for {
			select {
			case <-s.cleanup.C:
				s.teardownStaleStreams()
			case <-s.closed:
				s.cleanup.Stop()
				s.teardownAllStreams()
				s.log.Reportf("shutdown complete")
				return
			}
		}
	}()
}

func (s *server) GetOrCreateStream(key string) Stream {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := s.streams[key]
	if r != nil {
		r.markedForClose.Store(false)
		return r
	}
	st := newStream(key, s.serverType, s.sdkClient, s.metrics, s.log)
	s.streams[key] = st
	return st
}

func (s *server) Close() {
	s.closedOnce.Do(func() {
		close(s.closed)
	})
}

func (s *server) teardownAllStreams() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, str := range s.streams {
		str.close()
		delete(s.streams, id)
	}
}

func (s *server) teardownStaleStreams() {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for id, str := range s.streams {
		if str.markedForClose.Load() {
			str.close()
			delete(s.streams, id)
			count++
		}
	}
	s.log.Debugf("scheduled cleanup closed %d stale stream(s)", count)
}
