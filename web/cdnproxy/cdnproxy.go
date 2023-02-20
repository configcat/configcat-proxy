package cdnproxy

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"net/http"
)

type Server struct {
	sdkClient sdk.Client
	config    config.CdnProxyConfig
	logger    log.Logger
}

func NewServer(client sdk.Client, config config.CdnProxyConfig, log log.Logger) *Server {
	cdnLogger := log.WithPrefix("cdn-proxy")
	return &Server{
		sdkClient: client,
		config:    config,
		logger:    cdnLogger,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=0, must-revalidate")
	etag := r.Header.Get("If-None-Match")
	c := s.sdkClient.GetCachedJson()
	if etag == "" || c.Etag != etag {
		w.Header().Set("ETag", c.Etag)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(c.CachedJson)
	} else {
		w.Header().Set("ETag", c.Etag)
		w.WriteHeader(http.StatusNotModified)
	}
}
