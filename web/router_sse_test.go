package web

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/stretchr/testify/assert"
)

func TestSSE_EvalFlag_CORS(t *testing.T) {
	router, key := newSSERouter(t, config.SseConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}})
	defer router.Close()
	srv := httptest.NewServer(router)
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag"}`))
	dataK := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag", "sdkKey": "` + key + `"}`))

	methods := []string{http.MethodOptions, http.MethodGet}
	urls := []string{fmt.Sprintf("%s/sse/test/eval/%s", srv.URL, data), fmt.Sprintf("%s/sse/eval/k/%s", srv.URL, dataK)}

	for _, method := range methods {
		for _, url := range urls {
			t.Run(fmt.Sprintf("%s %s", url, method), func(t *testing.T) {
				req, _ := http.NewRequest(method, url, http.NoBody)
				resp, _ := http.DefaultClient.Do(req)
				var respCode int
				if method == http.MethodOptions {
					respCode = http.StatusNoContent
				} else {
					respCode = http.StatusOK
				}
				assert.Equal(t, respCode, resp.StatusCode)

				if method == http.MethodOptions {
					assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
					assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
					assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match", resp.Header.Get("Access-Control-Allow-Headers"))
					assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
					assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
					assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
					assert.Equal(t, "v1", resp.Header.Get("h1"))
				} else {
					assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
					assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))
					assert.Equal(t, "keep-alive", resp.Header.Get("Connection"))
					assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
					assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
					assert.Equal(t, "v1", resp.Header.Get("h1"))
				}
			})
		}
	}
}

func TestSSE_EvalFlag_Not_Allowed_Methods(t *testing.T) {
	router, key := newSSERouter(t, config.SseConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}})
	defer router.Close()
	srv := httptest.NewServer(router)
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag"}`))
	dataK := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag", "sdkKey": "` + key + `"}`))

	methods := []string{http.MethodPut, http.MethodPost, http.MethodDelete, http.MethodPatch}
	urls := []string{fmt.Sprintf("%s/sse/test/eval/%s", srv.URL, data), fmt.Sprintf("%s/sse/eval/k/%s", srv.URL, dataK)}

	for _, method := range methods {
		for _, url := range urls {
			t.Run(fmt.Sprintf("%s %s", url, method), func(t *testing.T) {
				req, _ := http.NewRequest(method, url, http.NoBody)
				resp, _ := http.DefaultClient.Do(req)
				assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
			})
		}
	}
}

func TestSSE_EvalAllFlags_CORS(t *testing.T) {
	router, key := newSSERouter(t, config.SseConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}})
	defer router.Close()
	srv := httptest.NewServer(router)
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag"}`))
	dataK := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag", "sdkKey": "` + key + `"}`))

	methods := []string{http.MethodOptions, http.MethodGet}
	urls := []string{
		fmt.Sprintf("%s/sse/test/eval-all", srv.URL),
		fmt.Sprintf("%s/sse/test/eval-all/%s", srv.URL, data),
		fmt.Sprintf("%s/sse/eval-all/k/%s", srv.URL, dataK),
	}

	for _, method := range methods {
		for _, url := range urls {
			t.Run(fmt.Sprintf("%s %s", url, method), func(t *testing.T) {
				req, _ := http.NewRequest(method, url, http.NoBody)
				resp, _ := http.DefaultClient.Do(req)
				var respCode int
				if method == http.MethodOptions {
					respCode = http.StatusNoContent
				} else {
					respCode = http.StatusOK
				}
				assert.Equal(t, respCode, resp.StatusCode)

				if method == http.MethodOptions {
					assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
					assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
					assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match", resp.Header.Get("Access-Control-Allow-Headers"))
					assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
					assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
					assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
					assert.Equal(t, "v1", resp.Header.Get("h1"))
				} else {
					assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
					assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))
					assert.Equal(t, "keep-alive", resp.Header.Get("Connection"))
					assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
					assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
					assert.Equal(t, "v1", resp.Header.Get("h1"))
				}
			})
		}
	}
}

func TestSSE_EvalAllFlags_Not_Allowed_Methods(t *testing.T) {
	router, key := newSSERouter(t, config.SseConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}})
	defer router.Close()
	srv := httptest.NewServer(router)
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag"}`))
	dataK := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag", "sdkKey": "` + key + `"}`))

	methods := []string{http.MethodPut, http.MethodPost, http.MethodDelete, http.MethodPatch}
	urls := []string{
		fmt.Sprintf("%s/sse/test/eval-all", srv.URL),
		fmt.Sprintf("%s/sse/test/eval-all/%s", srv.URL, data),
		fmt.Sprintf("%s/sse/eval-all/k/%s", srv.URL, dataK),
	}

	for _, method := range methods {
		for _, url := range urls {
			t.Run(fmt.Sprintf("%s %s", url, method), func(t *testing.T) {
				req, _ := http.NewRequest(method, url, http.NoBody)
				resp, _ := http.DefaultClient.Do(req)
				assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
			})
		}
	}
}

func newSSERouter(t *testing.T, conf config.SseConfig) (*HttpRouter, string) {
	reg, _, k := sdk.NewTestRegistrarT(t)
	return NewRouter(reg, telemetry.NewEmptyReporter(), status.NewEmptyReporter(), &config.HttpConfig{Sse: conf}, &config.ProfileConfig{}, log.NewNullLogger()), k
}
