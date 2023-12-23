package cdnproxy

import (
	"context"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"
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
	sdkClient, requestedVersion, err := s.getSDKClient(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if sdkClient.Version() == config.V6 && requestedVersion == config.V5 {
		http.Error(w, "only config V6 available", http.StatusNotFound)
		return
	}

	w.Header().Set("Cache-Control", "max-age=0, must-revalidate")
	etag := r.Header.Get("If-None-Match")
	c := sdkClient.GetCachedJson(requestedVersion)
	if etag == "" || c.ETag != etag {
		w.Header().Set("ETag", c.ETag)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(c.ConfigJson)
	} else {
		w.Header().Set("ETag", c.ETag)
		w.WriteHeader(http.StatusNotModified)
	}
}

func (s *Server) getSDKClient(ctx context.Context) (sdk.Client, config.SDKVersion, error) {
	vars := httprouter.ParamsFromContext(ctx)
	sdkId := vars.ByName("sdkId")
	ver := config.V5
	if sdkId == "" {
		return nil, ver, fmt.Errorf("'sdkId' path parameter must be set")
	}
	if sdkId[0] == '/' {
		sdkId = sdkId[1:] // trim left '/'
	}
	if strings.HasPrefix(sdkId, "configcat-proxy/") {
		ver = config.V6
		sdkId = sdkId[16:]
	}
	nextSlashIndex := strings.IndexByte(sdkId, '/')
	if nextSlashIndex != -1 {
		sdkId = sdkId[:nextSlashIndex]
	}
	sdkClient, ok := s.sdkClients[sdkId]
	if !ok {
		return nil, ver, fmt.Errorf("invalid SDK identifier: '%s'", sdkId)
	}
	return sdkClient, ver, nil
}
