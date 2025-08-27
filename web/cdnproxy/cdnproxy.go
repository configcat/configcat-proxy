package cdnproxy

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
)

type Server struct {
	sdkRegistrar sdk.Registrar
	config       *config.CdnProxyConfig
	logger       log.Logger
}

func NewServer(sdkRegistrar sdk.Registrar, config *config.CdnProxyConfig, log log.Logger) *Server {
	cdnLogger := log.WithPrefix("cdn-proxy")
	return &Server{
		sdkRegistrar: sdkRegistrar,
		config:       config,
		logger:       cdnLogger,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sdkClient, err, code := s.getSDKClient(r)
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

func (s *Server) getSDKClient(r *http.Request) (sdk.Client, error, int) {
	path := r.PathValue("path")
	var sdkClient sdk.Client

	if strings.HasPrefix(path, "configcat-proxy/") {
		path = strings.TrimPrefix(path, "configcat-proxy/")
		sdkId := strings.TrimSuffix(path, "/config_v6.json")
		sdkClient = s.sdkRegistrar.GetSdkOrNil(sdkId)
	} else {
		sdkKey := strings.TrimSuffix(path, "/config_v6.json")
		sdkClient = s.sdkRegistrar.GetSdkByKeyOrNil(sdkKey)
	}
	if sdkClient == nil {
		return nil, fmt.Errorf("could not identify a configured SDK"), http.StatusNotFound
	}
	if !sdkClient.IsInValidState() {
		return nil, fmt.Errorf("requested SDK is in an invalid state; please check the logs for more details"), http.StatusInternalServerError
	}
	return sdkClient, nil, http.StatusOK
}
