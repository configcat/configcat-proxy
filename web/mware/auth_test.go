package mware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/configcat/configcat-proxy/log"
	"github.com/stretchr/testify/assert"
)

func TestBasicAuth(t *testing.T) {
	handler := BasicAuth("user", "pass", log.NewNullLogger(), func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(handler)
	client := http.Client{}

	t.Run("missing auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("wrong auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		req.SetBasicAuth("user", "wrong")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("auth ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		req.SetBasicAuth("user", "pass")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestHeaderAuth(t *testing.T) {
	handler := HeaderAuth(map[string]string{"auth": "key"}, log.NewNullLogger(), func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(handler)
	client := http.Client{}

	t.Run("missing auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("wrong auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		req.Header.Set("auth", "wrong")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("auth ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		req.Header.Set("auth", "key")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
