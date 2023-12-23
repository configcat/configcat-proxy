package api

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
	"strings"
	"testing"
	"time"
)

func TestAPI_Eval(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

		srv := newServer(t, config.ApiConfig{Enabled: true})
		utils.AddSdkIdContextParam(req)
		srv.Eval(res, req)

		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"value":true,"variationId":"v_flag"}`, res.Body.String())
	})
	t.Run("online error", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

		srv := newErrorServer(t, config.ApiConfig{Enabled: true})
		utils.AddSdkIdContextParam(req)
		srv.Eval(res, req)

		assert.Equal(t, http.StatusInternalServerError, res.Code)
	})
	t.Run("offline", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":"","v":{"b":true},"t":0}}}`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

			srv := newOfflineServer(t, path, config.ApiConfig{Enabled: true})
			utils.AddSdkIdContextParam(req)
			srv.Eval(res, req)

			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, `{"value":true,"variationId":""}`, res.Body.String())
		})
	})
	t.Run("offline error", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":""`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

			srv := newOfflineServer(t, path, config.ApiConfig{Enabled: true})
			utils.AddSdkIdContextParam(req)
			srv.Eval(res, req)

			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})
	})
}

func TestAPI_EvalAll(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

		srv := newServer(t, config.ApiConfig{Enabled: true})
		utils.AddSdkIdContextParam(req)
		srv.EvalAll(res, req)

		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"flag":{"value":true,"variationId":"v_flag"}}`, res.Body.String())
	})
	t.Run("online error", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

		srv := newErrorServer(t, config.ApiConfig{Enabled: true})
		utils.AddSdkIdContextParam(req)
		srv.EvalAll(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, "{}", res.Body.String())
	})
	t.Run("offline", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":"","v":{"b":true},"t":0}}}`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

			srv := newOfflineServer(t, path, config.ApiConfig{Enabled: true})
			utils.AddSdkIdContextParam(req)
			srv.EvalAll(res, req)

			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, `{"flag":{"value":true,"variationId":""}}`, res.Body.String())
		})
	})
	t.Run("offline error", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":""`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

			srv := newOfflineServer(t, path, config.ApiConfig{Enabled: true})
			utils.AddSdkIdContextParam(req)
			srv.EvalAll(res, req)

			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, "{}", res.Body.String())
		})
	})
}

func TestAPI_Keys(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)

		srv := newServer(t, config.ApiConfig{Enabled: true})
		utils.AddSdkIdContextParam(req)
		srv.Keys(res, req)

		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"keys":["flag"]}`, res.Body.String())
	})
	t.Run("online error", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)

		srv := newErrorServer(t, config.ApiConfig{Enabled: true})
		utils.AddSdkIdContextParam(req)
		srv.Keys(res, req)

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, `{"keys":[]}`, res.Body.String())
	})
	t.Run("offline", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":"","v":{"b":true},"t":0}}}`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)

			srv := newOfflineServer(t, path, config.ApiConfig{Enabled: true})
			utils.AddSdkIdContextParam(req)
			srv.Keys(res, req)

			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, `{"keys":["flag"]}`, res.Body.String())
		})
	})
	t.Run("offline error", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":""`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)

			srv := newOfflineServer(t, path, config.ApiConfig{Enabled: true})
			utils.AddSdkIdContextParam(req)
			srv.Keys(res, req)

			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, `{"keys":[]}`, res.Body.String())
		})
	})
}

func TestAPI_Refresh(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})

	srv := newServerWithHandler(t, &h, key, config.ApiConfig{Enabled: true})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))
	utils.AddSdkIdContextParam(req)
	srv.Eval(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"value":true,"variationId":"v_flag"}`, res.Body.String())

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: false,
		},
	})

	req = &http.Request{Method: http.MethodPost}
	utils.AddSdkIdContextParam(req)
	srv.Refresh(httptest.NewRecorder(), req)
	time.Sleep(100 * time.Millisecond)
	res = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))
	utils.AddSdkIdContextParam(req)
	srv.Eval(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"value":false,"variationId":"v_flag"}`, res.Body.String())
}

func newServer(t *testing.T, conf config.ApiConfig) *Server {
	client, _, _ := testutils.NewTestSdkClient(t)
	return NewServer(client, &conf, log.NewNullLogger())
}

func newServerWithHandler(t *testing.T, h *configcattest.Handler, key string, conf config.ApiConfig) *Server {
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(h)
	ctx := testutils.NewTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	client := sdk.NewClient(ctx, log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return NewServer(map[string]sdk.Client{"test": client}, &conf, log.NewNullLogger())
}

func newErrorServer(t *testing.T, conf config.ApiConfig) *Server {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	srv := httptest.NewServer(&h)
	ctx := testutils.NewTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	client := sdk.NewClient(ctx, log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return NewServer(map[string]sdk.Client{"test": client}, &conf, log.NewNullLogger())
}

func newOfflineServer(t *testing.T, path string, conf config.ApiConfig) *Server {
	ctx := testutils.NewTestSdkContext(&config.SDKConfig{Key: "local", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path, Polling: true, PollInterval: 30}}}, nil)
	client := sdk.NewClient(ctx, log.NewNullLogger())
	t.Cleanup(func() {
		client.Close()
	})
	return NewServer(map[string]sdk.Client{"test": client}, &conf, log.NewNullLogger())
}
