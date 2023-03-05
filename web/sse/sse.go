package sse

import (
	"encoding/json"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/metrics"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/stream"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

const streamDataName = "data"

type Server struct {
	streamServer stream.Server
	config       config.SseConfig
	logger       log.Logger
	sdkClient    sdk.Client
	stop         chan struct{}
}

func NewServer(sdkClient sdk.Client, metrics metrics.Handler, conf config.SseConfig, logger log.Logger) *Server {
	sseLog := logger.WithLevel(conf.Log.GetLevel()).WithPrefix("sse")
	return &Server{
		streamServer: stream.NewServer(sdkClient, metrics, sseLog, "sse"),
		logger:       sseLog,
		config:       conf,
		sdkClient:    sdkClient,
		stop:         make(chan struct{}),
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
	streamData := vars.ByName("data")
	if streamData == "" {
		http.Error(w, fmt.Sprintf("'%s' must be set", streamDataName), http.StatusBadRequest)
		return
	}
	streamContext, err := utils.Base64URLDecode(streamData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode incoming '%s'", streamDataName), http.StatusBadRequest)
		return
	}
	var evalReq model.EvalRequest
	err = json.Unmarshal(streamContext, &evalReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to deserialize incoming '%s'", streamDataName), http.StatusBadRequest)
		return
	}
	if evalReq.Key == "" {
		http.Error(w, "'key' must be set", http.StatusBadRequest)
		return
	}

	var userAttrs *sdk.UserAttrs
	if evalReq.User != nil {
		userAttrs = &sdk.UserAttrs{Attrs: evalReq.User}
	}

	conn := s.streamServer.CreateConnection(evalReq.Key, userAttrs)

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
			s.streamServer.CloseConnection(conn, evalReq.Key)
			return
		case <-s.stop:
			return
		}
	}
}

func (s *Server) Close() {
	close(s.stop)
	s.streamServer.Close()
}
