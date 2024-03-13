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
		testutils.AddSdkIdContextParam(req)
		srv.Eval(res, req)

		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"value":true,"variationId":"v_flag"}`, res.Body.String())
	})
	t.Run("flag not found", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"non-existing"}`))

		srv := newServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.Eval(res, req)

		assert.Equal(t, 400, res.Code)
		assert.Equal(t, "feature flag or setting with key 'non-existing' not found\n", res.Body.String())
	})
	t.Run("online error", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

		srv := newErrorServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.Eval(res, req)

		assert.Equal(t, http.StatusInternalServerError, res.Code)
		assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
	})
	t.Run("online user", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag","user":{"Identifier":"test"}}`))

		srv := newServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.Eval(res, req)

		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"value":false,"variationId":"v0_flag"}`, res.Body.String())
	})
	t.Run("online user invalid", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag","user":{"Identifier":false}}`))

		srv := newServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.Eval(res, req)

		assert.Equal(t, 400, res.Code)
		assert.Contains(t, res.Body.String(), `failed to parse JSON body: 'Identifier' has an invalid type, only 'string', 'number', and 'string[]' types are allowed`)
	})
	t.Run("offline", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":"","v":{"b":true},"t":0}}}`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

			srv := newOfflineServer(t, path, config.ApiConfig{Enabled: true})
			testutils.AddSdkIdContextParam(req)
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
			testutils.AddSdkIdContextParam(req)
			srv.Eval(res, req)

			assert.Equal(t, http.StatusInternalServerError, res.Code)
			assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
		})
	})
}

func TestAPI_EvalAll(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

		srv := newServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.EvalAll(res, req)

		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"flag":{"value":true,"variationId":"v_flag"}}`, res.Body.String())
	})
	t.Run("online error", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

		srv := newErrorServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.EvalAll(res, req)

		assert.Equal(t, http.StatusInternalServerError, res.Code)
		assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
	})
	t.Run("online user", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag","user":{"Identifier":"test"}}`))

		srv := newServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.EvalAll(res, req)

		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"flag":{"value":false,"variationId":"v0_flag"}}`, res.Body.String())
	})
	t.Run("online user invalid", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag","user":{"Identifier":false}}`))

		srv := newServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.EvalAll(res, req)

		assert.Equal(t, 400, res.Code)
		assert.Contains(t, res.Body.String(), `failed to parse JSON body: 'Identifier' has an invalid type, only 'string', 'number', and 'string[]' types are allowed`)
	})
	t.Run("offline", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":"","v":{"b":true},"t":0}}}`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

			srv := newOfflineServer(t, path, config.ApiConfig{Enabled: true})
			testutils.AddSdkIdContextParam(req)
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
			testutils.AddSdkIdContextParam(req)
			srv.EvalAll(res, req)

			assert.Equal(t, http.StatusInternalServerError, res.Code)
			assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
		})
	})
}

func TestAPI_ICanHasCoffee(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)

		srv := newServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.ICanHasCoffee(res, req)

		assert.Equal(t, http.StatusTeapot, res.Code)
	})
}

func TestAPI_Keys(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)

		srv := newServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.Keys(res, req)

		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"keys":["flag"]}`, res.Body.String())
	})
	t.Run("online error", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)

		srv := newErrorServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.Keys(res, req)

		assert.Equal(t, http.StatusInternalServerError, res.Code)
		assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
	})
	t.Run("offline", func(t *testing.T) {
		utils.UseTempFile(`{"f":{"flag":{"i":"","v":{"b":true},"t":0}}}`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)

			srv := newOfflineServer(t, path, config.ApiConfig{Enabled: true})
			testutils.AddSdkIdContextParam(req)
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
			testutils.AddSdkIdContextParam(req)
			srv.Keys(res, req)

			assert.Equal(t, http.StatusInternalServerError, res.Code)
			assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
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
	testutils.AddSdkIdContextParam(req)
	srv.Eval(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"value":true,"variationId":"v_flag"}`, res.Body.String())

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: false,
		},
	})

	req = &http.Request{Method: http.MethodPost}
	testutils.AddSdkIdContextParam(req)
	srv.Refresh(httptest.NewRecorder(), req)
	time.Sleep(100 * time.Millisecond)
	res = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))
	testutils.AddSdkIdContextParam(req)
	srv.Eval(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"value":false,"variationId":"v_flag"}`, res.Body.String())
}

func TestAPI_WrongSdkId(t *testing.T) {
	t.Run("Eval", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

		srv := newServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParamWithSdkId(req, "non-existing")
		srv.Eval(res, req)

		assert.Equal(t, 404, res.Code)
		assert.Equal(t, res.Body.String(), "invalid SDK identifier: 'non-existing'\n")
	})
	t.Run("EvalAll", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

		srv := newServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParamWithSdkId(req, "non-existing")
		srv.EvalAll(res, req)

		assert.Equal(t, 404, res.Code)
		assert.Equal(t, res.Body.String(), "invalid SDK identifier: 'non-existing'\n")
	})
	t.Run("Keys", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)

		srv := newServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParamWithSdkId(req, "non-existing")
		srv.Keys(res, req)

		assert.Equal(t, 404, res.Code)
		assert.Equal(t, res.Body.String(), "invalid SDK identifier: 'non-existing'\n")
	})
	t.Run("Refresh", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)

		srv := newServer(t, config.ApiConfig{Enabled: true})
		testutils.AddSdkIdContextParamWithSdkId(req, "non-existing")
		srv.Refresh(res, req)

		assert.Equal(t, 404, res.Code)
		assert.Equal(t, res.Body.String(), "invalid SDK identifier: 'non-existing'\n")
	})
}

func TestAPI_WrongSDKState(t *testing.T) {
	opts := config.SDKConfig{BaseUrl: "http://localhost", Key: configcattest.RandomSDKKey()}
	ctx := testutils.NewTestSdkContext(&opts, &config.CacheConfig{})
	client := sdk.NewClient(ctx, log.NewNullLogger())
	defer client.Close()

	t.Run("Eval", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

		srv := NewServer(map[string]sdk.Client{"test": client}, &config.ApiConfig{Enabled: true}, log.NewNullLogger())
		testutils.AddSdkIdContextParam(req)
		srv.Eval(res, req)

		assert.Equal(t, http.StatusInternalServerError, res.Code)
		assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
	})
	t.Run("EvalAll", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"flag"}`))

		srv := NewServer(map[string]sdk.Client{"test": client}, &config.ApiConfig{Enabled: true}, log.NewNullLogger())
		testutils.AddSdkIdContextParam(req)
		srv.EvalAll(res, req)

		assert.Equal(t, http.StatusInternalServerError, res.Code)
		assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
	})
	t.Run("Keys", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)

		srv := NewServer(map[string]sdk.Client{"test": client}, &config.ApiConfig{Enabled: true}, log.NewNullLogger())
		testutils.AddSdkIdContextParam(req)
		srv.Keys(res, req)

		assert.Equal(t, http.StatusInternalServerError, res.Code)
		assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
	})
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
