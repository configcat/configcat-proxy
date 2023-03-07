package web

import (
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v7/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhook_BasicAuth(t *testing.T) {
	router := newWebhookRouter(t, config.WebhookConfig{Enabled: true, Auth: config.AuthConfig{User: "user", Password: "pass"}})
	srv := httptest.NewServer(router.Handler())
	client := http.Client{}

	t.Run("missing auth", func(t *testing.T) {
		resp, _ := http.Get(fmt.Sprintf("%s/hook", srv.URL))
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("wrong auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/hook", srv.URL), http.NoBody)
		req.SetBasicAuth("user", "wrong")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("get auth ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/hook", srv.URL), http.NoBody)
		req.SetBasicAuth("user", "pass")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("post auth ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/hook", srv.URL), http.NoBody)
		req.SetBasicAuth("user", "pass")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestWebhook_HeaderAuth(t *testing.T) {
	router := newWebhookRouter(t, config.WebhookConfig{Enabled: true, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router.Handler())
	client := http.Client{}

	t.Run("missing auth", func(t *testing.T) {
		resp, _ := http.Get(fmt.Sprintf("%s/hook", srv.URL))
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("wrong auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/hook", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "wrong")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("get auth ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/hook", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("post auth ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/hook", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestWebhook_NotAllowed(t *testing.T) {
	router := newWebhookRouter(t, config.WebhookConfig{Enabled: true, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router.Handler())
	client := http.Client{}

	t.Run("put", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/hook", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("patch", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/hook", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("delete", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/hook", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("options", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, fmt.Sprintf("%s/hook", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func newWebhookRouter(t *testing.T, conf config.WebhookConfig) *HttpRouter {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, config.HttpProxyConfig{}, nil, status.NewNullReporter(), log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return NewRouter(client, nil, status.NewNullReporter(), config.HttpConfig{Webhook: conf}, log.NewNullLogger())
}
