package cdnproxy

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxy_Get(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := &http.Request{Method: http.MethodGet}

		srv := newServer(t, config.CdnProxyConfig{Enabled: true})
		utils.AddContextParam(req, "configcat-proxy/test")
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[],"p":null}},"s":null,"p":null}`, res.Body.String())
	})
	t.Run("online error", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := &http.Request{Method: http.MethodGet}

		srv := newErrorServer(t, config.CdnProxyConfig{Enabled: true})
		utils.AddContextParam(req, "configcat-proxy/test")
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":null,"s":null,"p":null}`, res.Body.String())
	})
	t.Run("offline", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":"","v":{"b":true},"t":0}}}`, func(path string) {
			res := httptest.NewRecorder()
			req := &http.Request{Method: http.MethodGet}

			srv := newOfflineServer(t, path, config.CdnProxyConfig{Enabled: true})
			utils.AddContextParam(req, "configcat-proxy/test")
			srv.ServeHTTP(res, req)

			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, res.Body.String())
		})
	})
	t.Run("offline error", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":`, func(path string) {
			res := httptest.NewRecorder()
			req := &http.Request{Method: http.MethodGet}

			srv := newOfflineServer(t, path, config.CdnProxyConfig{Enabled: true})
			utils.AddContextParam(req, "configcat-proxy/test")
			srv.ServeHTTP(res, req)

			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, `{"f":null,"s":null,"p":null}`, res.Body.String())
		})
	})
	t.Run("etag", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := &http.Request{Method: http.MethodGet}

		srv := newServer(t, config.CdnProxyConfig{Enabled: true})
		utils.AddContextParam(req, "configcat-proxy/test")
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[],"p":null}},"s":null,"p":null}`, res.Body.String())

		etag := res.Header().Get("ETag")

		res = httptest.NewRecorder()
		req = &http.Request{Method: http.MethodGet, Header: map[string][]string{}}
		req.Header.Set("If-None-Match", etag)
		utils.AddContextParam(req, "configcat-proxy/test")
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotModified, res.Code)
		assert.Empty(t, res.Body.String())
	})
	t.Run("etag twice", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := &http.Request{Method: http.MethodGet}

		srv, h, key := newServerWithHandler(t, config.CdnProxyConfig{Enabled: true})
		utils.AddContextParam(req, "configcat-proxy/test")
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[],"p":null}},"s":null,"p":null}`, res.Body.String())

		etag := res.Header().Get("ETag")

		res = httptest.NewRecorder()
		req = &http.Request{Method: http.MethodGet, Header: map[string][]string{}}
		req.Header.Set("If-None-Match", etag)
		utils.AddContextParam(req, "configcat-proxy/test")
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotModified, res.Code)
		assert.Empty(t, res.Body.String())

		_ = h.SetFlags(key, map[string]*configcattest.Flag{
			"flag": {
				Default: false,
			},
		})
		_ = srv.sdkClients["test"].Refresh()

		res = httptest.NewRecorder()
		req = &http.Request{Method: http.MethodGet, Header: map[string][]string{}}
		req.Header.Set("If-None-Match", etag)
		utils.AddContextParam(req, "configcat-proxy/test")
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":[],"p":null}},"s":null,"p":null}`, res.Body.String())

		etag = res.Header().Get("ETag")

		res = httptest.NewRecorder()
		req = &http.Request{Method: http.MethodGet, Header: map[string][]string{}}
		req.Header.Set("If-None-Match", etag)
		utils.AddContextParam(req, "configcat-proxy/test")
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotModified, res.Code)
		assert.Empty(t, res.Body.String())
	})
}

func newServer(t *testing.T, proxyConfig config.CdnProxyConfig) *Server {
	client, _, _ := testutils.NewTestSdkClient(t)
	return NewServer(client, &proxyConfig, log.NewNullLogger())
}

func newServerWithHandler(t *testing.T, proxyConfig config.CdnProxyConfig) (*Server, *configcattest.Handler, string) {
	client, h, k := testutils.NewTestSdkClient(t)
	return NewServer(client, &proxyConfig, log.NewNullLogger()), h, k
}

func newErrorServer(t *testing.T, proxyConfig config.CdnProxyConfig) *Server {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	srv := httptest.NewServer(&h)
	ctx := testutils.NewTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	client := sdk.NewClient(ctx, log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return NewServer(map[string]sdk.Client{"test": client}, &proxyConfig, log.NewNullLogger())
}

func newOfflineServer(t *testing.T, path string, proxyConfig config.CdnProxyConfig) *Server {
	ctx := testutils.NewTestSdkContext(&config.SDKConfig{Key: "local", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path}}}, nil)
	client := sdk.NewClient(ctx, log.NewNullLogger())
	t.Cleanup(func() {
		client.Close()
	})
	return NewServer(map[string]sdk.Client{"test": client}, &proxyConfig, log.NewNullLogger())
}
