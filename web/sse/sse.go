package sse

import (
	"encoding/json"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/stream"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/url"
	"sync"
)

const streamKeyName = "key"

type Server struct {
	streamServer stream.Server
	config       config.SseConfig
	logger       log.Logger
	sdkClient    sdk.Client
	closed       chan struct{}
	closedOnce   sync.Once
}

func NewServer(sdkClient sdk.Client, metrics metrics.Handler, conf config.SseConfig, logger log.Logger) *Server {
	sseLog := logger.WithLevel(conf.Log.GetLevel()).WithPrefix("sse")
	return &Server{
		streamServer: stream.NewServer(sdkClient, metrics, sseLog, "sse"),
		logger:       sseLog,
		config:       conf,
		sdkClient:    sdkClient,
		closed:       make(chan struct{}),
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusNotImplemented)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Add("X-Accel-Buffering", "no")

	vars := httprouter.ParamsFromContext(r.Context())
	streamKey := vars.ByName("key")
	if streamKey == "" {
		http.Error(w, fmt.Sprintf("'%s' must be set", streamKeyName), http.StatusBadRequest)
		return
	}

	user := parseUserFromQuery(r.URL.Query())

	str := s.streamServer.GetOrCreateStream(streamKey)
	conn := str.CreateConnection(user)

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for {
		select {
		case payload := <-conn.Receive():
			data, e := json.Marshal(payload)
			if e == nil {
				_, e = fmt.Fprintf(w, "data: %s\n\n", string(data))
				if e == nil {
					flusher.Flush()
				} else {
					s.logger.Errorf("%s", e)
				}
			} else {
				s.logger.Errorf("%s", e)
			}
		case <-r.Context().Done():
			conn.Close()
			return
		case <-s.closed:
			return
		}
	}
}

func (s *Server) Close() {
	s.closedOnce.Do(func() {
		close(s.closed)
	})
	s.streamServer.Close()
}

func parseUserFromQuery(query url.Values) *sdk.UserAttrs {
	query.Del(streamKeyName)
	if len(query) == 0 {
		return nil
	}
	attrMap := make(map[string]string, len(query))
	for k, v := range query {
		if len(v) != 0 {
			attrMap[k] = v[0]
		}
	}
	if len(attrMap) == 0 {
		return nil
	}
	return &sdk.UserAttrs{Attrs: attrMap}
}
