package ofrep

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/web/api"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/stretchr/testify/assert"
)

func TestOFREP_Eval(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
		req.Header.Set(SdkIdHeader, "test")
		srv := newServer(t, config.OFREPConfig{Enabled: true})
		req.SetPathValue("key", "flag")
		srv.Eval(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"key":"flag","reason":"DEFAULT","variant":"v_flag","value":true}`, res.Body.String())
	})
	t.Run("online with sdk key", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
		srv, _, sdkKey := newServerWithHandler(t, config.OFREPConfig{Enabled: true})
		req.Header.Set(api.SdkKeyHeader, sdkKey)
		req.SetPathValue("key", "flag")
		srv.Eval(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"key":"flag","reason":"DEFAULT","variant":"v_flag","value":true}`, res.Body.String())
	})
	t.Run("without identification", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
		srv := newServer(t, config.OFREPConfig{Enabled: true})
		req.SetPathValue("key", "flag")
		srv.Eval(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 400, res.Code)
		assert.Equal(t, `{"key":"flag","errorCode":"GENERAL","errorDetails":"either the 'X-ConfigCat-SdkId' or 'X-ConfigCat-SdkKey' header must be set"}`, res.Body.String())
	})
	t.Run("online targeting", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"context": { "targetingKey":"id" }}`))
		req.Header.Set(SdkIdHeader, "test")
		srv, h, k := newServerWithHandler(t, config.OFREPConfig{Enabled: true})
		_ = h.SetFlags(k, map[string]*configcattest.Flag{
			"flag": {
				Default: false,
				Rules: []configcattest.Rule{{
					ComparisonAttribute: "Identifier",
					Comparator:          configcat.OpEq,
					ComparisonValue:     "id",
					Value:               true,
				}},
			},
		})
		_ = srv.sdkRegistrar.GetSdkOrNil("test").Refresh(t.Context())
		req.SetPathValue("key", "flag")
		srv.Eval(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"key":"flag","reason":"TARGETING_MATCH","variant":"v0_flag","value":true}`, res.Body.String())
	})
	t.Run("online targeting with sdk key", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"context": { "targetingKey":"id" }}`))
		srv, h, k := newServerWithHandler(t, config.OFREPConfig{Enabled: true})
		req.Header.Set(api.SdkKeyHeader, k)
		_ = h.SetFlags(k, map[string]*configcattest.Flag{
			"flag": {
				Default: false,
				Rules: []configcattest.Rule{{
					ComparisonAttribute: "Identifier",
					Comparator:          configcat.OpEq,
					ComparisonValue:     "id",
					Value:               true,
				}},
			},
		})
		_ = srv.sdkRegistrar.GetSdkOrNil("test").Refresh(t.Context())
		req.SetPathValue("key", "flag")
		srv.Eval(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"key":"flag","reason":"TARGETING_MATCH","variant":"v0_flag","value":true}`, res.Body.String())
	})
	t.Run("flag not found", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
		req.Header.Set(SdkIdHeader, "test")
		srv := newServer(t, config.OFREPConfig{Enabled: true})
		req.SetPathValue("key", "non-existing")
		srv.Eval(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 404, res.Code)
		assert.Equal(t, `{"key":"non-existing","errorCode":"FLAG_NOT_FOUND","errorDetails":"feature flag or setting with key 'non-existing' not found"}`, res.Body.String())
	})
	t.Run("online error", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
		req.Header.Set(SdkIdHeader, "test")
		srv := newErrorServer(t, config.OFREPConfig{Enabled: true})
		req.SetPathValue("key", "flag")
		srv.Eval(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, http.StatusInternalServerError, res.Code)
		assert.Equal(t, `{"errorDetails":"requested SDK is in an invalid state; please check the logs for more details"}`, res.Body.String())
	})
	t.Run("online user invalid", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"context":{"targetingKey":false}}`))
		req.Header.Set(SdkIdHeader, "test")
		srv := newServer(t, config.OFREPConfig{Enabled: true})
		req.SetPathValue("key", "flag")
		srv.Eval(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 400, res.Code)
		assert.Equal(t, `{"key":"flag","errorCode":"INVALID_CONTEXT","errorDetails":"failed to parse JSON body: 'targetingKey' has an invalid type, only 'string', 'number', and 'string[]' types are allowed"}`, res.Body.String())
	})
	t.Run("offline", func(t *testing.T) {
		testutils.UseTempFile(`{"f":{"flag":{"i":"","v":{"b":true},"t":0}}}`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
			req.Header.Set(SdkIdHeader, "test")
			srv := newOfflineServer(t, path, config.OFREPConfig{Enabled: true})
			req.SetPathValue("key", "flag")
			srv.Eval(res, req)

			assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, `{"key":"flag","reason":"DEFAULT","variant":"","value":true}`, res.Body.String())
		})
	})
	t.Run("offline error", func(t *testing.T) {
		testutils.UseTempFile(`{"f":{"flag":{"i":""`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
			req.Header.Set(SdkIdHeader, "test")
			srv := newOfflineServer(t, path, config.OFREPConfig{Enabled: true})
			req.SetPathValue("key", "flag")
			srv.Eval(res, req)

			assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
			assert.Equal(t, http.StatusInternalServerError, res.Code)
			assert.Equal(t, `{"errorDetails":"requested SDK is in an invalid state; please check the logs for more details"}`, res.Body.String())
		})
	})
}

func TestOFREP_EvalAll(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
		req.Header.Set(SdkIdHeader, "test")
		srv := newServer(t, config.OFREPConfig{Enabled: true})
		srv.EvalAll(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"flags":[{"key":"flag","reason":"DEFAULT","variant":"v_flag","value":true}]}`, res.Body.String())
	})
	t.Run("online error", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
		req.Header.Set(SdkIdHeader, "test")
		srv := newErrorServer(t, config.OFREPConfig{Enabled: true})
		testutils.AddSdkIdContextParam(req)
		srv.EvalAll(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, http.StatusInternalServerError, res.Code)
		assert.Equal(t, `{"errorDetails":"requested SDK is in an invalid state; please check the logs for more details"}`, res.Body.String())
	})
	t.Run("online user", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"context":{"targetingKey":"test"}}`))
		req.Header.Set(SdkIdHeader, "test")
		srv := newServer(t, config.OFREPConfig{Enabled: true})
		srv.EvalAll(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"flags":[{"key":"flag","reason":"TARGETING_MATCH","variant":"v0_flag","value":false}]}`, res.Body.String())
	})
	t.Run("online user invalid", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"context":{"targetingKey":false}}`))
		req.Header.Set(SdkIdHeader, "test")
		srv := newServer(t, config.OFREPConfig{Enabled: true})
		srv.EvalAll(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 400, res.Code)
		assert.Equal(t, `{"errorCode":"INVALID_CONTEXT","errorDetails":"failed to parse JSON body: 'targetingKey' has an invalid type, only 'string', 'number', and 'string[]' types are allowed"}`, res.Body.String())
	})
	t.Run("offline", func(t *testing.T) {
		testutils.UseTempFile(`{"f":{"flag":{"i":"","v":{"b":true},"t":0}}}`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
			req.Header.Set(SdkIdHeader, "test")
			srv := newOfflineServer(t, path, config.OFREPConfig{Enabled: true})
			srv.EvalAll(res, req)

			assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, `{"flags":[{"key":"flag","reason":"DEFAULT","variant":"","value":true}]}`, res.Body.String())
		})
	})
	t.Run("offline error", func(t *testing.T) {
		testutils.UseTempFile(`{"f":{"flag":{"i":""`, func(path string) {
			res := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
			req.Header.Set(SdkIdHeader, "test")
			srv := newOfflineServer(t, path, config.OFREPConfig{Enabled: true})
			testutils.AddSdkIdContextParam(req)
			srv.EvalAll(res, req)

			assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
			assert.Equal(t, http.StatusInternalServerError, res.Code)
			assert.Equal(t, `{"errorDetails":"requested SDK is in an invalid state; please check the logs for more details"}`, res.Body.String())
		})
	})
	t.Run("etag", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
		req.Header.Set(SdkIdHeader, "test")
		srv := newServer(t, config.OFREPConfig{Enabled: true})
		srv.EvalAll(res, req)

		etag := res.Header().Get("ETag")
		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, `W/"c172e8affdbb5db9"`, etag)
		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"flags":[{"key":"flag","reason":"DEFAULT","variant":"v_flag","value":true}]}`, res.Body.String())

		req, _ = http.NewRequest(http.MethodPost, "/", http.NoBody)
		req.Header.Set(SdkIdHeader, "test")
		req.Header.Set("If-None-Match", etag)

		res = httptest.NewRecorder()
		srv.EvalAll(res, req)

		etag = res.Header().Get("ETag")
		assert.Equal(t, 304, res.Code)
		assert.Equal(t, `W/"c172e8affdbb5db9"`, etag)
		assert.Equal(t, "", res.Body.String())
	})
	t.Run("etag user", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"context":{"Identifier":"test", "Email": "a@b.com"}}`))
		req.Header.Set(SdkIdHeader, "test")
		srv := newServer(t, config.OFREPConfig{Enabled: true})
		srv.EvalAll(res, req)

		etag := res.Header().Get("ETag")
		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 200, res.Code)
		assert.Equal(t, `{"flags":[{"key":"flag","reason":"TARGETING_MATCH","variant":"v0_flag","value":false}]}`, res.Body.String())

		req, _ = http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"context":{"Email": "a@b.com", "Identifier":"test"}}`))
		req.Header.Set(SdkIdHeader, "test")
		req.Header.Set("If-None-Match", etag)

		res = httptest.NewRecorder()
		srv.EvalAll(res, req)

		etag2 := res.Header().Get("ETag")
		assert.Equal(t, 304, res.Code)
		assert.Equal(t, etag, etag2)
		assert.Equal(t, "", res.Body.String())

		req, _ = http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"context":{"Email": "c@d.com", "Identifier":"test"}}`))
		req.Header.Set(SdkIdHeader, "test")
		req.Header.Set("If-None-Match", etag)

		res = httptest.NewRecorder()
		srv.EvalAll(res, req)

		etag3 := res.Header().Get("ETag")
		assert.Equal(t, 200, res.Code)
		assert.NotEqual(t, etag, etag3)
		assert.Equal(t, `{"flags":[{"key":"flag","reason":"TARGETING_MATCH","variant":"v0_flag","value":false}]}`, res.Body.String())
	})
}

func TestOFREP_WrongSdkId(t *testing.T) {
	t.Run("Eval", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)

		srv := newServer(t, config.OFREPConfig{Enabled: true})
		req.Header.Set(SdkIdHeader, "non-existing")
		req.SetPathValue("key", "flag")
		srv.Eval(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 400, res.Code)
		assert.Equal(t, `{"key":"flag","errorCode":"GENERAL","errorDetails":"could not identify a configured SDK"}`, res.Body.String())
	})
	t.Run("EvalAll", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
		req.Header.Set(SdkIdHeader, "non-existing")
		srv := newServer(t, config.OFREPConfig{Enabled: true})
		srv.EvalAll(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, 400, res.Code)
		assert.Equal(t, `{"errorCode":"GENERAL","errorDetails":"could not identify a configured SDK"}`, res.Body.String())
	})
}

func TestOFREP_WrongSDKState(t *testing.T) {
	reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: "http://localhost", Key: configcattest.RandomSDKKey()}, nil)
	defer reg.Close()

	t.Run("Eval", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
		req.Header.Set(SdkIdHeader, "test")
		srv := NewServer(reg, &config.OFREPConfig{Enabled: true}, log.NewNullLogger())
		req.SetPathValue("key", "flag")
		srv.Eval(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, http.StatusInternalServerError, res.Code)
		assert.Equal(t, `{"errorDetails":"requested SDK is in an invalid state; please check the logs for more details"}`, res.Body.String())
	})
	t.Run("EvalAll", func(t *testing.T) {
		res := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/", http.NoBody)
		req.Header.Set(SdkIdHeader, "test")
		srv := NewServer(reg, &config.OFREPConfig{Enabled: true}, log.NewNullLogger())
		srv.EvalAll(res, req)

		assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
		assert.Equal(t, http.StatusInternalServerError, res.Code)
		assert.Equal(t, `{"errorDetails":"requested SDK is in an invalid state; please check the logs for more details"}`, res.Body.String())
	})
}

func newServer(t *testing.T, conf config.OFREPConfig) *Server {
	reg, _, _ := sdk.NewTestRegistrarT(t)
	return NewServer(reg, &conf, log.NewNullLogger())
}

func newServerWithHandler(t *testing.T, conf config.OFREPConfig) (*Server, *configcattest.Handler, string) {
	reg, h, k := sdk.NewTestRegistrarT(t)
	return NewServer(reg, &conf, log.NewNullLogger()), h, k
}

func newErrorServer(t *testing.T, conf config.OFREPConfig) *Server {
	reg := sdk.NewTestRegistrarTWithErrorServer(t)
	return NewServer(reg, &conf, log.NewNullLogger())
}

func newOfflineServer(t *testing.T, path string, conf config.OFREPConfig) *Server {
	reg := sdk.NewTestRegistrar(&config.SDKConfig{Key: "local", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path, Polling: true, PollInterval: 30}}}, nil)
	t.Cleanup(func() {
		reg.Close()
	})
	return NewServer(reg, &conf, log.NewNullLogger())
}
