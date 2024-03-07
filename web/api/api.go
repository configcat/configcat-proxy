package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	configcat "github.com/configcat/go-sdk/v9"
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
		http.Error(w, "Failed to parse JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}
	sdkClient, err := s.getSDKClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	eval, err := sdkClient.Eval(evalReq.Key, evalReq.User)
	if err != nil {
		var errKeyNotFound configcat.ErrKeyNotFound
		if errors.As(err, &errKeyNotFound) {
			http.Error(w, "evaluation failed; the setting with the key '"+evalReq.Key+"' not found", http.StatusBadRequest)
		} else {
			http.Error(w, "the request failed; please check the logs for more details", http.StatusInternalServerError)
		}
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
		http.Error(w, "Failed to parse JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}
	sdkClient, err := s.getSDKClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
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
		http.Error(w, err.Error(), http.StatusNotFound)
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
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	err = sdkClient.Refresh()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) ICanHasCoffee(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusTeapot)
}

func (s *Server) getSDKClient(ctx context.Context) (sdk.Client, error) {
	vars := httprouter.ParamsFromContext(ctx)
	sdkId := vars.ByName("sdkId")
	if sdkId == "" {
		return nil, fmt.Errorf("'sdkId' path parameter must be set")
	}
	sdkClient, ok := s.sdkClients[sdkId]
	if !ok {
		return nil, fmt.Errorf("invalid SDK identifier: '%s'", sdkId)
	}
	return sdkClient, nil
}
