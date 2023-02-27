package cdnproxy

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v7/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxy_Get(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := http.Request{Method: http.MethodGet}

		srv := newServer(t, config.CdnProxyConfig{Enabled: true})
		srv.ServeHTTP(res, &req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":{"flag":{"i":"v_flag","v":true,"t":0,"r":[],"p":null}},"p":null}`, res.Body.String())
	})
	t.Run("online error", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := http.Request{Method: http.MethodGet}

		srv := newErrorServer(t, config.CdnProxyConfig{Enabled: true})
		srv.ServeHTTP(res, &req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":{}}`, res.Body.String())
	})
	t.Run("offline", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, func(path string) {
			res := httptest.NewRecorder()
			req := http.Request{Method: http.MethodGet}

			srv := newOfflineServer(t, path, config.CdnProxyConfig{Enabled: true})
			srv.ServeHTTP(res, &req)

			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, res.Body.String())
		})
	})
	t.Run("offline error", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":`, func(path string) {
			res := httptest.NewRecorder()
			req := http.Request{Method: http.MethodGet}

			srv := newOfflineServer(t, path, config.CdnProxyConfig{Enabled: true})
			srv.ServeHTTP(res, &req)

			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, `{"f":{}}`, res.Body.String())
		})
	})
	t.Run("etag", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := http.Request{Method: http.MethodGet}

		srv := newServer(t, config.CdnProxyConfig{Enabled: true})
		srv.ServeHTTP(res, &req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":{"flag":{"i":"v_flag","v":true,"t":0,"r":[],"p":null}},"p":null}`, res.Body.String())

		etag := res.Header().Get("ETag")

		res = httptest.NewRecorder()
		req = http.Request{Method: http.MethodGet, Header: map[string][]string{}}
		req.Header.Set("If-None-Match", etag)
		srv.ServeHTTP(res, &req)

		assert.Equal(t, http.StatusNotModified, res.Code)
		assert.Empty(t, res.Body.String())
	})
}

func newServer(t *testing.T, proxyConfig config.CdnProxyConfig) *Server {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return NewServer(client, proxyConfig, log.NewNullLogger())
}

func newErrorServer(t *testing.T, proxyConfig config.CdnProxyConfig) *Server {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	srv := httptest.NewServer(&h)
	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return NewServer(client, proxyConfig, log.NewNullLogger())
}

func newOfflineServer(t *testing.T, path string, proxyConfig config.CdnProxyConfig) *Server {
	opts := config.SDKConfig{Key: "local", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path}}}
	client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
	t.Cleanup(func() {
		client.Close()
	})
	return NewServer(client, proxyConfig, log.NewNullLogger())
}
