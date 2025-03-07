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
	sdkRegistrar sdk.Registrar
	config       *config.ApiConfig
	logger       log.Logger
}

func NewServer(sdkRegistrar sdk.Registrar, config *config.ApiConfig, log log.Logger) *Server {
	cdnLogger := log.WithPrefix("api")
	return &Server{
		sdkRegistrar: sdkRegistrar,
		config:       config,
		logger:       cdnLogger,
	}
}

func (s *Server) Eval(w http.ResponseWriter, r *http.Request) {
	var evalReq model.EvalRequest
	sdkClient, err, code := s.parseRequest(r, &evalReq)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}
	eval := sdkClient.Eval(evalReq.Key, evalReq.User)
	if eval.Error != nil {
		var errKeyNotFound configcat.ErrKeyNotFound
		if errors.As(eval.Error, &errKeyNotFound) {
			http.Error(w, "feature flag or setting with key '"+evalReq.Key+"' not found", http.StatusBadRequest)
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
	var evalReq model.EvalRequest
	sdkClient, err, code := s.parseRequest(r, &evalReq)
	if err != nil {
		http.Error(w, err.Error(), code)
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
	sdkClient, err, code := s.getSDKClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), code)
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
	sdkClient, err, code := s.getSDKClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), code)
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

func (s *Server) parseRequest(r *http.Request, evalReq *model.EvalRequest) (sdk.Client, error, int) {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body"), http.StatusBadRequest
	}
	err = json.Unmarshal(reqBody, &evalReq)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON body: %s", err), http.StatusBadRequest
	}
	sdkClient, err, code := s.getSDKClient(r.Context())
	if err != nil {
		return nil, err, code
	}
	return sdkClient, nil, http.StatusOK
}

func (s *Server) getSDKClient(ctx context.Context) (sdk.Client, error, int) {
	vars := httprouter.ParamsFromContext(ctx)
	sdkId := vars.ByName("sdkId")
	if sdkId == "" {
		return nil, fmt.Errorf("'sdkId' path parameter must be set"), http.StatusNotFound
	}
	sdkClient := s.sdkRegistrar.GetSdkOrNil(sdkId)
	if sdkClient == nil {
		return nil, fmt.Errorf("invalid SDK identifier: '%s'", sdkId), http.StatusNotFound
	}
	if !sdkClient.IsInValidState() {
		return nil, fmt.Errorf("SDK with identifier '%s' is in an invalid state; please check the logs for more details", sdkId), http.StatusInternalServerError
	}
	return sdkClient, nil, http.StatusOK
}
