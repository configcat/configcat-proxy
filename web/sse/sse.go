package sse

import (
	"encoding/json"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/metrics"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
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

func NewServer(sdkRegistrar sdk.Registrar, metrics metrics.Reporter, conf *config.SseConfig, logger log.Logger) *Server {
	sseLog := logger.WithLevel(conf.Log.GetLevel()).WithPrefix("sse")
	return &Server{
		streamServer: stream.NewServer(sdkRegistrar, metrics, sseLog, "sse"),
		logger:       sseLog,
		config:       conf,
		stop:         make(chan struct{}),
	}
}

func (s *Server) SingleFlag(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusNotImplemented)
		return
	}

	evalReq, sdkId := prepareResponse(w, r, true)
	if evalReq == nil {
		return
	}

	if evalReq.Key == "" {
		http.Error(w, "'key' must be set", http.StatusBadRequest)
		return
	}

	str := s.streamServer.GetStreamOrNil(sdkId)
	if str == nil {
		http.Error(w, "SDK not found for identifier: '"+sdkId+"'", http.StatusNotFound)
		return
	}
	if !str.IsInValidState() {
		http.Error(w, "SDK with identifier '"+sdkId+"' is in an invalid state; please check the logs for more details", http.StatusInternalServerError)
		return
	}
	if !str.CanEval(evalReq.Key) {
		http.Error(w, "feature flag or setting with key '"+evalReq.Key+"' not found", http.StatusBadRequest)
		return
	}

	s.listenAndRespond(str, evalReq.User, evalReq.Key, w, r, flusher)
}

func (s *Server) AllFlags(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusNotImplemented)
		return
	}

	evalReq, sdkId := prepareResponse(w, r, false)
	str := s.streamServer.GetStreamOrNil(sdkId)
	if str == nil {
		http.Error(w, "SDK not found for identifier: '"+sdkId+"'", http.StatusNotFound)
		return
	}
	if !str.IsInValidState() {
		http.Error(w, "SDK with identifier '"+sdkId+"' is in an invalid state; please check the logs for more details", http.StatusInternalServerError)
		return
	}

	s.listenAndRespond(str, evalReq.User, stream.AllFlagsDiscriminator, w, r, flusher)
}

func (s *Server) Close() {
	close(s.stop)
	s.streamServer.Close()
}

func (s *Server) listenAndRespond(str stream.Stream, attrs model.UserAttrs, key string, w http.ResponseWriter, r *http.Request, flusher http.Flusher) {
	conn := str.CreateConnection(key, attrs)

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
			str.CloseConnection(conn, key)
			return
		case <-str.Closed():
			return
		case <-s.stop:
			return
		}
	}
}

func prepareResponse(w http.ResponseWriter, r *http.Request, dataMustSet bool) (*model.EvalRequest, string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Add("X-Accel-Buffering", "no")

	vars := httprouter.ParamsFromContext(r.Context())
	sdkId := vars.ByName("sdkId")
	if sdkId == "" {
		http.Error(w, "'sdkId' path parameter must be set", http.StatusBadRequest)
		return nil, ""
	}
	streamData := vars.ByName("data")
	if streamData == "" {
		if dataMustSet {
			http.Error(w, "'"+streamDataName+"' path parameter must be set", http.StatusBadRequest)
			return nil, ""
		} else {
			return &model.EvalRequest{}, sdkId
		}
	} else {
		streamContext, err := utils.Base64URLDecode(streamData)
		if err != nil {
			http.Error(w, "failed to decode incoming '"+streamDataName+"'", http.StatusBadRequest)
			return nil, ""
		}
		var evalReq model.EvalRequest
		err = json.Unmarshal(streamContext, &evalReq)
		if err != nil {
			http.Error(w, "failed to deserialize incoming '"+streamDataName+"': "+err.Error(), http.StatusBadRequest)
			return nil, ""
		}
		return &evalReq, sdkId
	}
}

func formatSseMsg(b []byte) []byte {
	r := make([]byte, 0, len(b)+8)
	r = append(r, "data: "...)
	r = append(r, b...)
	r = append(r, '\n', '\n')
	return r
}
