package web

import (
	"compress/gzip"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCDNProxy_Options_CORS(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}})
	srv := httptest.NewServer(router.Handler())
	req, _ := http.NewRequest(http.MethodOptions, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match", resp.Header.Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
	assert.Equal(t, "v1", resp.Header.Get("h1"))
}

func TestCDNProxy_Options_NO_CORS(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: false}})
	srv := httptest.NewServer(router.Handler())
	req, _ := http.NewRequest(http.MethodOptions, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Headers"))
	assert.Empty(t, resp.Header.Get("Access-Control-Max-Age"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
	assert.Empty(t, resp.Header.Get("h1"))
}

func TestCDNProxy_GET_CORS(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}})
	srv := httptest.NewServer(router.Handler())
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Headers"))
	assert.Empty(t, resp.Header.Get("Access-Control-Max-Age"))
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
	assert.Equal(t, "v1", resp.Header.Get("h1"))
}

func TestCDNProxy_GET_NO_CORS(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: false}})
	srv := httptest.NewServer(router.Handler())
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Headers"))
	assert.Empty(t, resp.Header.Get("Access-Control-Max-Age"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
	assert.Empty(t, resp.Header.Get("h1"))
}

func TestCDNProxy_Not_Allowed_Methods(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}})
	srv := httptest.NewServer(router.Handler())

	t.Run("put", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("post", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("delete", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("patch", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestCDNProxy_Get_Body(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}})
	srv := httptest.NewServer(router.Handler())
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[{"s":{"v":{"b":false,"s":null,"i":null,"d":null},"i":"v0_flag"},"c":[{"u":{"a":"Identifier","s":"test","d":null,"l":null,"c":28},"s":null,"p":null}],"p":null}],"p":null}},"s":null,"p":null}`, string(body))
	assert.Equal(t, "v1", resp.Header.Get("h1"))
}

func TestCDNProxy_Get_Body_GZip(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}})
	srv := httptest.NewServer(router.Handler())
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
	gzipReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	body, _ := io.ReadAll(gzipReader)
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[{"s":{"v":{"b":false,"s":null,"i":null,"d":null},"i":"v0_flag"},"c":[{"u":{"a":"Identifier","s":"test","d":null,"l":null,"c":28},"s":null,"p":null}],"p":null}],"p":null}},"s":null,"p":null}`, string(body))
	assert.Equal(t, "v1", resp.Header.Get("h1"))
}

func newCDNProxyRouter(t *testing.T, conf config.CdnProxyConfig) *HttpRouter {
	reg, _, _ := testutils.NewTestRegistrarT(t)
	return NewRouter(reg, nil, status.NewEmptyReporter(), &config.HttpConfig{CdnProxy: conf}, log.NewNullLogger())
}
