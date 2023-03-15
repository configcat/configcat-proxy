package sse

import (
	"encoding/json"
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
	config       *config.SseConfig
	logger       log.Logger
	stop         chan struct{}
}

func NewServer(sdkClients map[string]sdk.Client, metrics metrics.Handler, conf *config.SseConfig, logger log.Logger) *Server {
	sseLog := logger.WithLevel(conf.Log.GetLevel()).WithPrefix("sse")
	return &Server{
		streamServer: stream.NewServer(sdkClients, metrics, sseLog, "sse"),
		logger:       sseLog,
		config:       conf,
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
		http.Error(w, "'"+streamDataName+"' path parameter must be set", http.StatusBadRequest)
		return
	}
	env := vars.ByName("env")
	if env == "" {
		http.Error(w, "'env' path parameter must be set", http.StatusBadRequest)
		return
	}
	streamContext, err := utils.Base64URLDecode(streamData)
	if err != nil {
		http.Error(w, "Failed to decode incoming '"+streamDataName+"'", http.StatusBadRequest)
		return
	}
	var evalReq model.EvalRequest
	err = json.Unmarshal(streamContext, &evalReq)
	if err != nil {
		http.Error(w, "Failed to deserialize incoming '"+streamDataName+"'", http.StatusBadRequest)
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

	str := s.streamServer.GetStreamOrNil(env)
	if str == nil {
		http.Error(w, "Invalid environment identifier: '"+env+"'", http.StatusBadRequest)
	}

	conn := str.CreateConnection(evalReq.Key, userAttrs)
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for {
		select {
		case payload := <-conn.Receive():
			data, e := json.Marshal(payload)
			if e == nil {
				_, e = w.Write(formatSseMsg(data))
				if e == nil {
					flusher.Flush()
				} else {
					s.logger.Errorf("%s", e)
				}
			} else {
				s.logger.Errorf("%s", e)
			}
		case <-r.Context().Done():
			str.CloseConnection(conn, evalReq.Key)
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

func formatSseMsg(b []byte) []byte {
	r := make([]byte, 0, len(b)+8)
	r = append(r, "data: "...)
	r = append(r, b...)
	r = append(r, '\n', '\n')
	return r
}
