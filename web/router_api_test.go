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
	"strings"
	"testing"
)

func TestAPI_Eval(t *testing.T) {
	router := newAPIRouter(t, config.ApiConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router.Handler())
	path := fmt.Sprintf("%s/api/test/eval", srv.URL)

	t.Run("options cors", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		assert.Equal(t, "POST,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match,X-AUTH", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("missing auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(`{"key":"flag"}`))
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, `{"value":true,"variationId":"v_flag"}`, string(body))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("ok gzip", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(`{"key":"flag"}`))
		req.Header.Set("X-AUTH", "key")
		req.Header.Set("Accept-Encoding", "gzip")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
		gzipReader, err := gzip.NewReader(resp.Body)
		assert.NoError(t, err)
		body, _ := io.ReadAll(gzipReader)
		assert.Equal(t, `{"value":true,"variationId":"v_flag"}`, string(body))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("get not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("put not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("patch not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("delete not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestAPI_Eval_Headers(t *testing.T) {
	router := newAPIRouter(t, config.ApiConfig{Enabled: true, CORS: config.CORSConfig{Enabled: false}, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router.Handler())
	path := fmt.Sprintf("%s/api/test/eval", srv.URL)

	t.Run("options", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Empty(t, resp.Header.Get("Access-Control-Max-Age"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Empty(t, resp.Header.Get("h1"))
	})
	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(`{"key":"flag"}`))
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, `{"value":true,"variationId":"v_flag"}`, string(body))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Empty(t, resp.Header.Get("h1"))
	})
}

func TestAPI_EvalAll(t *testing.T) {
	router := newAPIRouter(t, config.ApiConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router.Handler())
	path := fmt.Sprintf("%s/api/test/eval-all", srv.URL)

	t.Run("options cors", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		assert.Equal(t, "POST,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match,X-AUTH", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("missing auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(`{"key":"flag"}`))
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, `{"flag":{"value":true,"variationId":"v_flag"}}`, string(body))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("ok gzip", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(`{"key":"flag"}`))
		req.Header.Set("X-AUTH", "key")
		req.Header.Set("Accept-Encoding", "gzip")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
		gzipReader, err := gzip.NewReader(resp.Body)
		assert.NoError(t, err)
		body, _ := io.ReadAll(gzipReader)
		assert.Equal(t, `{"flag":{"value":true,"variationId":"v_flag"}}`, string(body))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("get not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("put not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("patch not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("delete not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestAPI_EvalAll_Headers(t *testing.T) {
	router := newAPIRouter(t, config.ApiConfig{Enabled: true, CORS: config.CORSConfig{Enabled: false}, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router.Handler())
	path := fmt.Sprintf("%s/api/test/eval-all", srv.URL)

	t.Run("options", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Empty(t, resp.Header.Get("Access-Control-Max-Age"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Empty(t, resp.Header.Get("h1"))
	})
	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(`{"key":"flag"}`))
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, `{"flag":{"value":true,"variationId":"v_flag"}}`, string(body))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Empty(t, resp.Header.Get("h1"))
	})
}

func TestAPI_Keys(t *testing.T) {
	router := newAPIRouter(t, config.ApiConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router.Handler())
	path := fmt.Sprintf("%s/api/test/keys", srv.URL)

	t.Run("options cors", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match,X-AUTH", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("missing auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, path, http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, `{"keys":["flag"]}`, string(body))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("ok gzip", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, path, http.NoBody)
		req.Header.Set("X-AUTH", "key")
		req.Header.Set("Accept-Encoding", "gzip")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
		gzipReader, err := gzip.NewReader(resp.Body)
		assert.NoError(t, err)
		body, _ := io.ReadAll(gzipReader)
		assert.Equal(t, `{"keys":["flag"]}`, string(body))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("post not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("put not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("patch not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("delete not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestAPI_Keys_Headers(t *testing.T) {
	router := newAPIRouter(t, config.ApiConfig{Enabled: true, CORS: config.CORSConfig{Enabled: false}, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router.Handler())
	path := fmt.Sprintf("%s/api/test/keys", srv.URL)

	t.Run("options", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Empty(t, resp.Header.Get("Access-Control-Max-Age"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Empty(t, resp.Header.Get("h1"))
	})
	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, path, http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, `{"keys":["flag"]}`, string(body))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Empty(t, resp.Header.Get("h1"))
	})
}

func TestAPI_Refresh(t *testing.T) {
	router := newAPIRouter(t, config.ApiConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router.Handler())
	path := fmt.Sprintf("%s/api/test/refresh", srv.URL)

	t.Run("options cors", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		assert.Equal(t, "POST,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match,X-AUTH", resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("missing auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, "", string(body))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("get not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("put not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("patch not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("delete not allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestAPI_Refresh_Headers(t *testing.T) {
	router := newAPIRouter(t, config.ApiConfig{Enabled: true, CORS: config.CORSConfig{Enabled: false}, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router.Handler())
	path := fmt.Sprintf("%s/api/test/refresh", srv.URL)

	t.Run("options", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, path, http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Empty(t, resp.Header.Get("Access-Control-Max-Age"))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Empty(t, resp.Header.Get("h1"))
	})
	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, http.NoBody)
		req.Header.Set("X-AUTH", "key")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, "", string(body))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Empty(t, resp.Header.Get("h1"))
	})
}

func newAPIRouter(t *testing.T, conf config.ApiConfig) *HttpRouter {
	reg, _, _ := testutils.NewTestRegistrarT(t)
	return NewRouter(reg, nil, status.NewEmptyReporter(), &config.HttpConfig{Api: conf}, log.NewNullLogger())
}
