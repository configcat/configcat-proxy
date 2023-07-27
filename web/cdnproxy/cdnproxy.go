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
	sdkClient, err := s.getSDKClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Cache-Control", "max-age=0, must-revalidate")
	etag := r.Header.Get("If-None-Match")
	c := sdkClient.GetCachedJson()
	if etag == "" || c.GeneratedETag != etag {
		w.Header().Set("ETag", c.GeneratedETag)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(c.ConfigJson)
	} else {
		w.Header().Set("ETag", c.GeneratedETag)
		w.WriteHeader(http.StatusNotModified)
	}
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
