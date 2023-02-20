package api

import (
	"encoding/json"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	"io"
	"net/http"
)

type evalRequest struct {
	Key  string            `json:"key"`
	User map[string]string `json:"user"`
}

type keysResponse struct {
	Keys []string `json:"keys"`
}

type Server struct {
	sdkClient sdk.Client
	config    config.ApiConfig
	logger    log.Logger
}

func NewServer(client sdk.Client, config config.ApiConfig, log log.Logger) *Server {
	cdnLogger := log.WithPrefix("api")
	return &Server{
		sdkClient: client,
		config:    config,
		logger:    cdnLogger,
	}
}

func (s *Server) Eval(w http.ResponseWriter, r *http.Request) {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	var evalReq evalRequest
	err = json.Unmarshal(reqBody, &evalReq)
	if err != nil {
		http.Error(w, "failed to parse JSON body", http.StatusBadRequest)
		return
	}
	var userAttrs sdk.UserAttrs
	if evalReq.User != nil {
		userAttrs = sdk.UserAttrs{Attrs: evalReq.User}
	}
	eval, err := s.sdkClient.Eval(evalReq.Key, &userAttrs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	payload := model.PayloadFromEvalData(&eval)
	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (s *Server) EvalAll(w http.ResponseWriter, r *http.Request) {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	var evalReq evalRequest
	err = json.Unmarshal(reqBody, &evalReq)
	if err != nil {
		http.Error(w, "failed to parse JSON body", http.StatusBadRequest)
		return
	}
	var userAttrs sdk.UserAttrs
	if evalReq.User != nil {
		userAttrs = sdk.UserAttrs{Attrs: evalReq.User}
	}
	details := s.sdkClient.EvalAll(&userAttrs)
	res := make(map[string]model.ResponsePayload, len(details))
	for key, detail := range details {
		res[key] = model.PayloadFromEvalData(&detail)
	}
	data, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (s *Server) Keys(w http.ResponseWriter, _ *http.Request) {
	keys := s.sdkClient.Keys()
	data, err := json.Marshal(keysResponse{Keys: keys})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (s *Server) Refresh(w http.ResponseWriter, r *http.Request) {
	err := s.sdkClient.Refresh()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
