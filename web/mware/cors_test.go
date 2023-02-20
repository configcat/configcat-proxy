package mware

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	handler := CORS([]string{http.MethodGet, http.MethodOptions}, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(handler)
	client := http.Client{}

	t.Run("* origin, options", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, srv.URL, http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))
	})
	t.Run("custom origin, options", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, srv.URL, http.NoBody)
		req.Header.Set("Origin", "http://localhost")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "http://localhost", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))
	})
	t.Run("* origin, get", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))
	})
	t.Run("custom origin, get", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
		req.Header.Set("Origin", "http://localhost")
		resp, _ := client.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "http://localhost", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding", resp.Header.Get("Access-Control-Expose-Headers"))
	})
}
