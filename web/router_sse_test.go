package web

import (
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v7/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSSE_Options_CORS(t *testing.T) {
	router := newSSERouter(t, config.SseConfig{Enabled: true, AllowCORS: true, Headers: map[string]string{"h1": "v1"}})
	srv := httptest.NewServer(router.Handler())
	client := http.Client{}
	req, _ := http.NewRequest(http.MethodOptions, fmt.Sprintf("%s/sse/flag", srv.URL), http.NoBody)
	resp, _ := client.Do(req)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match", resp.Header.Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))
	assert.Equal(t, "v1", resp.Header.Get("h1"))
}

func TestSSE_GET_CORS(t *testing.T) {
	router := newSSERouter(t, config.SseConfig{Enabled: true, AllowCORS: true, Headers: map[string]string{"h1": "v1"}})
	srv := httptest.NewServer(router.Handler())
	client := http.Client{}
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/sse/flag", srv.URL), http.NoBody)
	resp, _ := client.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))
	assert.Equal(t, "keep-alive", resp.Header.Get("Connection"))
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))
	assert.Equal(t, "v1", resp.Header.Get("h1"))
}

func TestSSE_Not_Allowed_Methods(t *testing.T) {
	router := newSSERouter(t, config.SseConfig{Enabled: true, AllowCORS: true})
	srv := httptest.NewServer(router.Handler())
	client := http.Client{}

	t.Run("put", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/sse/flag", srv.URL), http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("post", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/sse/flag", srv.URL), http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("delete", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/sse/flag", srv.URL), http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("patch", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/sse/flag", srv.URL), http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func newSSERouter(t *testing.T, conf config.SseConfig) *HttpRouter {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return NewRouter(client, nil, config.HttpConfig{Sse: conf}, log.NewNullLogger())
}
