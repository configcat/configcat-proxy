package ofrep

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/julienschmidt/httprouter"
	"hash/maphash"
	"io"
	"net/http"
)

type errorCode string
type reason string

const (
	GeneralErrorCode        errorCode = "GENERAL"
	FlagNotFoundErrorCode   errorCode = "FLAG_NOT_FOUND"
	InvalidContextErrorCode errorCode = "INVALID_CONTEXT"

	DefaultReason        reason = "DEFAULT"
	TargetingMatchReason reason = "TARGETING_MATCH"

	SdkIdHeader = "X-ConfigCat-SdkId"
	ServerName  = "configcat-proxy"
)

type ofrepConfiguration struct {
	Name         string       `json:"name"`
	Capabilities capabilities `json:"capabilities"`
}

type capabilities struct {
	CacheInvalidation featureCacheInvalidation `json:"cacheInvalidation"`
	FlagEvaluation    flagEvaluation           `json:"flagEvaluation"`
}

type flagEvaluation struct {
	SupportedTypes []string `json:"supportedTypes"`
}

type featureCacheInvalidation struct {
	Polling featureCacheInvalidationPolling `json:"polling"`
}

type featureCacheInvalidationPolling struct {
	Enabled bool `json:"enabled"`
}

type evaluationRequest struct {
	Context model.UserAttrs `json:"context"`
}

type evaluationResponse struct {
	Key     string      `json:"key"`
	Reason  reason      `json:"reason"`
	Variant string      `json:"variant"`
	Value   interface{} `json:"value"`
}

type errorResponse struct {
	Key          string    `json:"key"`
	ErrorCode    errorCode `json:"errorCode"`
	ErrorDetails string    `json:"errorDetails"`
}

type bulkEvaluationResponse struct {
	Flags []interface{} `json:"flags"`
}

type bulkErrorResponse struct {
	ErrorCode    errorCode `json:"errorCode"`
	ErrorDetails string    `json:"errorDetails"`
}

type generalErrorResponse struct {
	ErrorDetails string `json:"errorDetails"`
}

type Server struct {
	sdkRegistrar  sdk.Registrar
	config        *config.OFREPConfig
	logger        log.Logger
	ofrepConf     []byte
	ofrepConfETag string
	seed          maphash.Seed
}

func NewServer(sdkRegistrar sdk.Registrar, config *config.OFREPConfig, log log.Logger) *Server {
	cdnLogger := log.WithPrefix("api")
	ofrepConf, _ := json.Marshal(ofrepConfiguration{
		Name: ServerName,
		Capabilities: capabilities{
			CacheInvalidation: featureCacheInvalidation{
				Polling: featureCacheInvalidationPolling{
					Enabled: true,
				},
			},
			FlagEvaluation: flagEvaluation{
				SupportedTypes: []string{"string", "boolean", "int", "float"},
			},
		},
	})
	return &Server{
		sdkRegistrar:  sdkRegistrar,
		config:        config,
		logger:        cdnLogger,
		ofrepConf:     ofrepConf,
		ofrepConfETag: utils.GenerateEtag(ofrepConf),
		seed:          maphash.MakeSeed(),
	}
}

func (s *Server) Eval(w http.ResponseWriter, r *http.Request) {
	vars := httprouter.ParamsFromContext(r.Context())
	key := vars.ByName("key")
	if key == "" {
		s.writeError(w, errorResponse{ErrorCode: GeneralErrorCode, ErrorDetails: "'key' path parameter must be set", Key: ""}, http.StatusBadRequest)
		return
	}

	var evalReq evaluationRequest
	sdkClient, err, errCode, code := s.parseRequest(r, &evalReq)
	if err != nil {
		if code == http.StatusInternalServerError {
			s.writeError(w, generalErrorResponse{ErrorDetails: err.Error()}, code)
			return
		}
		s.writeError(w, errorResponse{ErrorDetails: err.Error(), ErrorCode: errCode, Key: key}, code)
		return
	}
	mapTargetingKeyToIdentifier(evalReq.Context)
	eval := sdkClient.Eval(key, evalReq.Context)
	if eval.Error != nil {
		var errKeyNotFound configcat.ErrKeyNotFound
		if errors.As(eval.Error, &errKeyNotFound) {
			s.writeError(w, errorResponse{ErrorDetails: "feature flag or setting with key '" + key + "' not found", ErrorCode: FlagNotFoundErrorCode, Key: key}, http.StatusNotFound)
		} else {
			s.writeError(w, generalErrorResponse{ErrorDetails: "the request failed; please check the server logs for more details"}, http.StatusInternalServerError)
		}
		return
	}
	payload := toEvalResponse(&eval, key)
	data, err := json.Marshal(payload)
	if err != nil {
		s.writeError(w, generalErrorResponse{ErrorDetails: err.Error()}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (s *Server) EvalAll(w http.ResponseWriter, r *http.Request) {
	var evalReq evaluationRequest
	sdkClient, err, errCode, code := s.parseRequest(r, &evalReq)
	if err != nil {
		if code == http.StatusInternalServerError {
			s.writeError(w, generalErrorResponse{ErrorDetails: err.Error()}, code)
			return
		}
		s.writeError(w, bulkErrorResponse{ErrorDetails: err.Error(), ErrorCode: errCode}, code)
		return
	}
	etag := r.Header.Get("If-None-Match")
	c := sdkClient.GetCachedJson()
	genEtag := s.calcEtag(evalReq.Context, c.ETag)
	if etag != "" && etag == genEtag {
		w.Header().Set("ETag", genEtag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	mapTargetingKeyToIdentifier(evalReq.Context)
	details := sdkClient.EvalAll(evalReq.Context)
	flags := make([]interface{}, 0, len(details))
	for key, detail := range details {
		if detail.Error != nil {
			flags = append(flags, errorResponse{ErrorDetails: detail.Error.Error(), ErrorCode: GeneralErrorCode, Key: key})
		} else {
			flags = append(flags, toEvalResponse(&detail, key))
		}
	}
	payload := bulkEvaluationResponse{Flags: flags}
	data, err := json.Marshal(payload)
	if err != nil {
		s.writeError(w, generalErrorResponse{ErrorDetails: err.Error()}, http.StatusInternalServerError)
		return
	}
	w.Header().Set("ETag", genEtag)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (s *Server) GetConfiguration(w http.ResponseWriter, r *http.Request) {
	etag := r.Header.Get("If-None-Match")
	if etag != "" && etag == s.ofrepConfETag {
		w.Header().Set("ETag", s.ofrepConfETag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", s.ofrepConfETag)
	_, _ = w.Write(s.ofrepConf)
}

func (s *Server) writeError(w http.ResponseWriter, body interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	data, _ := json.Marshal(body)
	_, _ = w.Write(data)
}

func (s *Server) parseRequest(r *http.Request, evalReq *evaluationRequest) (sdk.Client, error, errorCode, int) {
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body"), GeneralErrorCode, http.StatusBadRequest
	}
	if len(reqBody) > 0 {
		err = json.Unmarshal(reqBody, &evalReq)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON body: " + err.Error()), InvalidContextErrorCode, http.StatusBadRequest
		}
	}
	sdkClient, err, errCode, code := s.getSDKClient(r)
	if err != nil {
		return nil, err, errCode, code
	}
	return sdkClient, nil, "", http.StatusOK
}

func (s *Server) getSDKClient(r *http.Request) (sdk.Client, error, errorCode, int) {
	sdkId := r.Header.Get(SdkIdHeader)
	if sdkId == "" {
		return nil, fmt.Errorf("'%s' header must be set", SdkIdHeader), GeneralErrorCode, http.StatusBadRequest
	}
	sdkClient := s.sdkRegistrar.GetSdkOrNil(sdkId)
	if sdkClient == nil {
		return nil, fmt.Errorf("invalid SDK identifier: '%s'", sdkId), GeneralErrorCode, http.StatusBadRequest
	}
	if !sdkClient.IsInValidState() {
		return nil, fmt.Errorf("SDK with identifier '%s' is in an invalid state; please check the logs for more details", sdkId), GeneralErrorCode, http.StatusInternalServerError
	}
	return sdkClient, nil, "", http.StatusOK
}

func (s *Server) calcEtag(attr model.UserAttrs, configJsonEtag string) string {
	attrHash := attr.Discriminator(s.seed)
	payload := append([]byte(configJsonEtag), utils.Uint64ToBytes(attrHash)...)
	return utils.GenerateEtag(payload)
}

func mapTargetingKeyToIdentifier(attr model.UserAttrs) {
	if val, ok := attr["targetingKey"]; ok {
		attr["Identifier"] = val
	}
}

func toEvalResponse(data *model.EvalData, key string) evaluationResponse {
	response := evaluationResponse{
		Key:     key,
		Value:   data.Value,
		Variant: data.VariationId,
		Reason:  DefaultReason,
	}
	if data.IsTargeting {
		response.Reason = TargetingMatchReason
	}
	return response
}
