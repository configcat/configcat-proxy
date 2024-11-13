package cdnproxy

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestProxy_Get(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := &http.Request{Method: http.MethodGet}

		srv := newServer(t, config.CdnProxyConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[{"s":{"v":{"b":false,"s":null,"i":null,"d":null},"i":"v0_flag"},"c":[{"u":{"a":"Identifier","s":"test","d":null,"l":null,"c":28},"s":null,"p":null}],"p":null}],"p":null}},"s":null,"p":null}`, res.Body.String())
	})
	t.Run("online error", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := &http.Request{Method: http.MethodGet}

		srv := newErrorServer(t, config.CdnProxyConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusInternalServerError, res.Code)
		assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
	})
	t.Run("offline", func(t *testing.T) {
		testutils.UseTempFile(`{"f":{"flag":{"i":"","v":{"b":true},"t":0}}}`, func(path string) {
			res := httptest.NewRecorder()
			req := &http.Request{Method: http.MethodGet}

			srv := newOfflineServer(t, path, config.CdnProxyConfig{Enabled: true})
			testutils.AddSdkIdContextParam(req)
			srv.ServeHTTP(res, req)

			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, res.Body.String())
		})
	})
	t.Run("offline error", func(t *testing.T) {
		testutils.UseTempFile(`{"f":{"flag":{"i":`, func(path string) {
			res := httptest.NewRecorder()
			req := &http.Request{Method: http.MethodGet}

			srv := newOfflineServer(t, path, config.CdnProxyConfig{Enabled: true})
			testutils.AddSdkIdContextParam(req)
			srv.ServeHTTP(res, req)

			assert.Equal(t, http.StatusInternalServerError, res.Code)
			assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
		})
	})
	t.Run("etag", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := &http.Request{Method: http.MethodGet}

		srv := newServer(t, config.CdnProxyConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[{"s":{"v":{"b":false,"s":null,"i":null,"d":null},"i":"v0_flag"},"c":[{"u":{"a":"Identifier","s":"test","d":null,"l":null,"c":28},"s":null,"p":null}],"p":null}],"p":null}},"s":null,"p":null}`, res.Body.String())

		etag := res.Header().Get("ETag")

		res = httptest.NewRecorder()
		req = &http.Request{Method: http.MethodGet, Header: map[string][]string{}}
		req.Header.Set("If-None-Match", etag)
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotModified, res.Code)
		assert.Empty(t, res.Body.String())
	})
	t.Run("etag query", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := &http.Request{Method: http.MethodGet}

		srv := newServer(t, config.CdnProxyConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[{"s":{"v":{"b":false,"s":null,"i":null,"d":null},"i":"v0_flag"},"c":[{"u":{"a":"Identifier","s":"test","d":null,"l":null,"c":28},"s":null,"p":null}],"p":null}],"p":null}},"s":null,"p":null}`, res.Body.String())

		etag := res.Header().Get("ETag")

		res = httptest.NewRecorder()
		req = &http.Request{Method: http.MethodGet, URL: &url.URL{RawQuery: "ccetag=" + etag}}
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotModified, res.Code)
		assert.Empty(t, res.Body.String())
	})
	t.Run("etag twice", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := &http.Request{Method: http.MethodGet}

		srv, h, key := newServerWithHandler(t, config.CdnProxyConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[{"s":{"v":{"b":false,"s":null,"i":null,"d":null},"i":"v0_flag"},"c":[{"u":{"a":"Identifier","s":"test","d":null,"l":null,"c":28},"s":null,"p":null}],"p":null}],"p":null}},"s":null,"p":null}`, res.Body.String())

		etag := res.Header().Get("ETag")

		res = httptest.NewRecorder()
		req = &http.Request{Method: http.MethodGet, Header: map[string][]string{}}
		req.Header.Set("If-None-Match", etag)
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotModified, res.Code)
		assert.Empty(t, res.Body.String())

		_ = h.SetFlags(key, map[string]*configcattest.Flag{
			"flag": {
				Default: false,
			},
		})
		_ = srv.sdkRegistrar.GetSdkOrNil("test").Refresh()

		res = httptest.NewRecorder()
		req = &http.Request{Method: http.MethodGet, Header: map[string][]string{}}
		req.Header.Set("If-None-Match", etag)
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":[],"p":null}},"s":null,"p":null}`, res.Body.String())

		etag = res.Header().Get("ETag")

		res = httptest.NewRecorder()
		req = &http.Request{Method: http.MethodGet, Header: map[string][]string{}}
		req.Header.Set("If-None-Match", etag)
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotModified, res.Code)
		assert.Empty(t, res.Body.String())
	})
	t.Run("invalid SDK ID", func(t *testing.T) {
		res := httptest.NewRecorder()
		req := &http.Request{Method: http.MethodGet}

		srv := newServer(t, config.CdnProxyConfig{Enabled: true})
		testutils.AddSdkIdContextParamWithSdkId(req, "non-existing")
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusNotFound, res.Code)
	})
	t.Run("SDK invalid state", func(t *testing.T) {
		reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: "http://localhost", Key: configcattest.RandomSDKKey()}, nil)
		defer reg.Close()

		res := httptest.NewRecorder()
		req := &http.Request{Method: http.MethodGet}

		srv := NewServer(reg, &config.CdnProxyConfig{Enabled: true}, log.NewNullLogger())
		testutils.AddSdkIdContextParam(req)
		srv.ServeHTTP(res, req)

		assert.Equal(t, http.StatusInternalServerError, res.Code)
		assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
	})
}

func newServer(t *testing.T, proxyConfig config.CdnProxyConfig) *Server {
	reg, _, _ := sdk.NewTestRegistrarT(t)
	return NewServer(reg, &proxyConfig, log.NewNullLogger())
}

func newServerWithHandler(t *testing.T, proxyConfig config.CdnProxyConfig) (*Server, *configcattest.Handler, string) {
	reg, h, k := sdk.NewTestRegistrarT(t)
	return NewServer(reg, &proxyConfig, log.NewNullLogger()), h, k
}

func newErrorServer(t *testing.T, proxyConfig config.CdnProxyConfig) *Server {
	reg := sdk.NewTestRegistrarTWithErrorServer(t)
	return NewServer(reg, &proxyConfig, log.NewNullLogger())
}

func newOfflineServer(t *testing.T, path string, proxyConfig config.CdnProxyConfig) *Server {
	reg := sdk.NewTestRegistrar(&config.SDKConfig{Key: "local", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path}}}, nil)
	t.Cleanup(func() {
		reg.Close()
	})
	return NewServer(reg, &proxyConfig, log.NewNullLogger())
}
