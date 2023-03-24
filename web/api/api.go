package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/julienschmidt/httprouter"
	"io"
	"net/http"
)

type keysResponse struct {
	Keys []string `json:"keys"`
}

type Server struct {
	sdkClients map[string]sdk.Client
	config     *config.ApiConfig
	logger     log.Logger
}

func NewServer(sdkClients map[string]sdk.Client, config *config.ApiConfig, log log.Logger) *Server {
	cdnLogger := log.WithPrefix("api")
	return &Server{
		sdkClients: sdkClients,
		config:     config,
		logger:     cdnLogger,
	}
}

func (s *Server) Eval(w http.ResponseWriter, r *http.Request) {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	var evalReq model.EvalRequest
	err = json.Unmarshal(reqBody, &evalReq)
	if err != nil {
		http.Error(w, "Failed to parse JSON body", http.StatusBadRequest)
		return
	}
	sdkClient, err := s.getSDKClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	eval, err := sdkClient.Eval(evalReq.Key, evalReq.User)
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
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	var evalReq model.EvalRequest
	err = json.Unmarshal(reqBody, &evalReq)
	if err != nil {
		http.Error(w, "Failed to parse JSON body", http.StatusBadRequest)
		return
	}
	sdkClient, err := s.getSDKClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	details := sdkClient.EvalAll(evalReq.User)
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

func (s *Server) Keys(w http.ResponseWriter, r *http.Request) {
	sdkClient, err := s.getSDKClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	keys := sdkClient.Keys()
	data, err := json.Marshal(keysResponse{Keys: keys})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (s *Server) Refresh(w http.ResponseWriter, r *http.Request) {
	sdkClient, err := s.getSDKClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = sdkClient.Refresh()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) getSDKClient(ctx context.Context) (sdk.Client, error) {
	vars := httprouter.ParamsFromContext(ctx)
	env := vars.ByName("env")
	if env == "" {
		return nil, fmt.Errorf("'env' path parameter must be set")
	}
	sdkClient, ok := s.sdkClients[env]
	if !ok {
		return nil, fmt.Errorf("invalid environment identifier: '%s'", env)
	}
	return sdkClient, nil
}
