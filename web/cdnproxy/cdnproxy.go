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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Cache-Control", "max-age=0, must-revalidate")
	etag := r.Header.Get("If-None-Match")
	c := sdkClient.GetCachedJson()
	if etag == "" || c.Etag != etag {
		w.Header().Set("ETag", c.Etag)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(c.CachedJson)
	} else {
		w.Header().Set("ETag", c.Etag)
		w.WriteHeader(http.StatusNotModified)
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
