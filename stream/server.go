package stream

import (
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/sdk"
)

type Server interface {
	GetStreamOrNil(envId string) Stream
	Close()
}

type server struct {
	streams map[string]Stream
	log     log.Logger
}

func NewServer(sdkClients map[string]sdk.Client, metrics metrics.Handler, log log.Logger, serverType string) Server {
	strLog := log.WithPrefix("stream-server")
	streams := make(map[string]Stream)
	for id, sdkClient := range sdkClients {
		streams[id] = NewStream(id, sdkClient, metrics, strLog, serverType)
	}
	return &server{
		log:     strLog,
		streams: streams,
	}
}

func (s *server) GetStreamOrNil(envId string) Stream {
	if stream, ok := s.streams[envId]; ok {
		return stream
	}
	return nil
}

func (s *server) Close() {
	for id, str := range s.streams {
		str.Close()
		delete(s.streams, id)
	}
	s.log.Reportf("shutdown complete")
}
