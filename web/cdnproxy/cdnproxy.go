package cdnproxy

import (
	"context"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type Server struct {
	sdkClients map[string]sdk.Client
	config     *config.CdnProxyConfig
	logger     log.Logger
}

func NewServer(sdkClients map[string]sdk.Client, config *config.CdnProxyConfig, log log.Logger) *Server {
	cdnLogger := log.WithPrefix("cdn-proxy")
	return &Server{
		sdkClients: sdkClients,
		config:     config,
		logger:     cdnLogger,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sdkClient, err, code := s.getSDKClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}
	w.Header().Set("Cache-Control", "max-age=0, must-revalidate")
	etag := r.Header.Get("If-None-Match")
	if etag == "" && r.URL != nil {
		query := r.URL.Query()
		etag = query.Get("ccetag")
	}
	c := sdkClient.GetCachedJson()
	if etag == "" || c.ETag != etag {
		w.Header().Set("ETag", c.ETag)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(c.ConfigJson)
	} else {
		w.Header().Set("ETag", c.ETag)
		w.WriteHeader(http.StatusNotModified)
	}
}

func (s *Server) getSDKClient(ctx context.Context) (sdk.Client, error, int) {
	vars := httprouter.ParamsFromContext(ctx)
	sdkId := vars.ByName("sdkId")
	sdkClient, ok := s.sdkClients[sdkId]
	if !ok {
		return nil, fmt.Errorf("invalid SDK identifier: '%s'", sdkId), http.StatusNotFound
	}
	if !sdkClient.IsInValidState() {
		return nil, fmt.Errorf("SDK with identifier '%s' is in an invalid state; please check the logs for more details", sdkId), http.StatusInternalServerError
	}
	return sdkClient, nil, http.StatusOK
}
