package web

import (
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhook_BasicAuth(t *testing.T) {
	router := newWebhookRouter(t, config.WebhookConfig{Enabled: true, Auth: config.AuthConfig{User: "user", Password: "pass"}})
	srv := httptest.NewServer(router.Handler())

	t.Run("missing auth", func(t *testing.T) {
		resp, _ := http.Get(fmt.Sprintf("%s/hook/test", srv.URL))
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("wrong auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/hook/test", srv.URL), http.NoBody)
		req.SetBasicAuth("user", "wrong")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("get auth ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/hook/test", srv.URL), http.NoBody)
		req.SetBasicAuth("user", "pass")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("post auth ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/hook/test", srv.URL), http.NoBody)
		req.SetBasicAuth("user", "pass")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestWebhook_HeaderAuth(t *testing.T) {
	router := newWebhookRouter(t, config.WebhookConfig{Enabled: true, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router.Handler())

	t.Run("missing auth", func(t *testing.T) {
		resp, _ := http.Get(fmt.Sprintf("%s/hook/test", srv.URL))
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("wrong auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/hook/test", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "wrong")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("get auth ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/hook/test", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
	t.Run("post auth ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/hook/test", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestWebhook_NotAllowed(t *testing.T) {
	router := newWebhookRouter(t, config.WebhookConfig{Enabled: true, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router.Handler())

	t.Run("put", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/hook/test", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("patch", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/hook/test", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("delete", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/hook/test", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("options", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, fmt.Sprintf("%s/hook/test", srv.URL), http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func newWebhookRouter(t *testing.T, conf config.WebhookConfig) *HttpRouter {
	clients, _, _ := testutils.NewTestSdkClient(t)
	return NewRouter(clients, nil, status.NewNullReporter(), &config.HttpConfig{Webhook: conf}, log.NewNullLogger())
}
