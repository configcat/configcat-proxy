package web

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/web/ofrep"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcattest"
	ofrepprovider "github.com/open-feature/go-sdk-contrib/providers/ofrep"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOFREP_Integration(t *testing.T) {
	reg, h, key := sdk.NewTestRegistrarT(t)
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"bool": {
			Default: false,
			Rules:   []configcattest.Rule{{ComparisonAttribute: "Identifier", ComparisonValue: "id", Comparator: configcat.OpEq, Value: true}},
		},
		"str": {
			Default: "default",
			Rules:   []configcattest.Rule{{ComparisonAttribute: "Identifier", ComparisonValue: "id", Comparator: configcat.OpEq, Value: "test"}},
		},
		"int": {
			Default: 0,
			Rules:   []configcattest.Rule{{ComparisonAttribute: "Identifier", ComparisonValue: "id", Comparator: configcat.OpEq, Value: 42}},
		},
		"double": {
			Default: 0.0,
			Rules:   []configcattest.Rule{{ComparisonAttribute: "Identifier", ComparisonValue: "id", Comparator: configcat.OpEq, Value: 3.14}},
		},
	})
	router := NewRouter(reg, nil, status.NewEmptyReporter(), &config.HttpConfig{OFREP: config.OFREPConfig{Enabled: true, AuthHeaders: map[string]string{"X-API-Key": "secret"}}}, &config.ProfileConfig{}, log.NewNullLogger())
	srv := httptest.NewServer(router)
	defer srv.Close()

	provider := ofrepprovider.NewProvider(srv.URL, ofrepprovider.WithHeaderProvider(func() (string, string) {
		return ofrep.SdkIdHeader, "test"
	}), ofrepprovider.WithApiKeyAuth("secret"))
	_ = openfeature.SetProviderAndWait(provider)
	ctx := openfeature.NewEvaluationContext("id", nil)
	client := openfeature.NewClient("cl")

	boolVal, _ := client.BooleanValueDetails(context.Background(), "bool", false, ctx)
	assert.True(t, boolVal.Value)
	assert.Equal(t, "v0_bool", boolVal.Variant)
	assert.Equal(t, openfeature.TargetingMatchReason, boolVal.Reason)
	assert.Equal(t, "bool", boolVal.FlagKey)

	strVal, _ := client.StringValueDetails(context.Background(), "str", "", ctx)
	assert.Equal(t, "test", strVal.Value)
	assert.Equal(t, "v0_str", strVal.Variant)
	assert.Equal(t, openfeature.TargetingMatchReason, strVal.Reason)
	assert.Equal(t, "str", strVal.FlagKey)

	intVal, _ := client.IntValueDetails(context.Background(), "int", 0, ctx)
	assert.Equal(t, int64(42), intVal.Value)
	assert.Equal(t, "v0_int", intVal.Variant)
	assert.Equal(t, openfeature.TargetingMatchReason, intVal.Reason)
	assert.Equal(t, "int", intVal.FlagKey)

	doubleVal, _ := client.FloatValueDetails(context.Background(), "double", 0.0, ctx)
	assert.Equal(t, 3.14, doubleVal.Value)
	assert.Equal(t, "v0_double", doubleVal.Variant)
	assert.Equal(t, openfeature.TargetingMatchReason, doubleVal.Reason)
	assert.Equal(t, "double", doubleVal.FlagKey)
}

func TestOFREP_Eval(t *testing.T) {
	router := newOFREPRouter(t, config.OFREPConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router)
	path := fmt.Sprintf("%s/ofrep/v1/evaluate/flags/flag", srv.URL)

	t.Run("options cors", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, path, http.NoBody)
		req.Header.Add(ofrep.SdkIdHeader, "test")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		assert.Equal(t, "POST,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match,X-AUTH,"+ofrep.SdkIdHeader, resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("missing auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, http.NoBody)
		req.Header.Set(ofrep.SdkIdHeader, "test")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, http.NoBody)
		req.Header.Set("X-AUTH", "key")
		req.Header.Set(ofrep.SdkIdHeader, "test")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, `{"key":"flag","reason":"DEFAULT","variant":"v_flag","value":true}`, string(body))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("ok gzip", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, http.NoBody)
		req.Header.Set("X-AUTH", "key")
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set(ofrep.SdkIdHeader, "test")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
		gzipReader, err := gzip.NewReader(resp.Body)
		assert.NoError(t, err)
		body, _ := io.ReadAll(gzipReader)
		assert.Equal(t, `{"key":"flag","reason":"DEFAULT","variant":"v_flag","value":true}`, string(body))
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

func TestOFREP_Eval_Headers(t *testing.T) {
	router := newOFREPRouter(t, config.OFREPConfig{Enabled: true, CORS: config.CORSConfig{Enabled: false}, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router)
	path := fmt.Sprintf("%s/ofrep/v1/evaluate/flags/flag", srv.URL)

	t.Run("options", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, path, http.NoBody)
		req.Header.Set(ofrep.SdkIdHeader, "test")
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
		req.Header.Set(ofrep.SdkIdHeader, "test")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, `{"key":"flag","reason":"DEFAULT","variant":"v_flag","value":true}`, string(body))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Empty(t, resp.Header.Get("h1"))
	})
}

func TestOFREP_EvalAll(t *testing.T) {
	router := newOFREPRouter(t, config.OFREPConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router)
	path := fmt.Sprintf("%s/ofrep/v1/evaluate/flags", srv.URL)

	t.Run("options cors", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, path, http.NoBody)
		req.Header.Set(ofrep.SdkIdHeader, "test")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		assert.Equal(t, "POST,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match,X-AUTH,"+ofrep.SdkIdHeader, resp.Header.Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("missing auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, http.NoBody)
		req.Header.Set(ofrep.SdkIdHeader, "test")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, http.NoBody)
		req.Header.Set("X-AUTH", "key")
		req.Header.Set(ofrep.SdkIdHeader, "test")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, `{"flags":[{"key":"flag","reason":"DEFAULT","variant":"v_flag","value":true}]}`, string(body))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Equal(t, "v1", resp.Header.Get("h1"))
	})
	t.Run("ok gzip", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, path, http.NoBody)
		req.Header.Set("X-AUTH", "key")
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set(ofrep.SdkIdHeader, "test")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
		gzipReader, err := gzip.NewReader(resp.Body)
		assert.NoError(t, err)
		body, _ := io.ReadAll(gzipReader)
		assert.Equal(t, `{"flags":[{"key":"flag","reason":"DEFAULT","variant":"v_flag","value":true}]}`, string(body))
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

func TestOFREP_EvalAll_Headers(t *testing.T) {
	router := newOFREPRouter(t, config.OFREPConfig{Enabled: true, CORS: config.CORSConfig{Enabled: false}, AuthHeaders: map[string]string{"X-AUTH": "key"}})
	srv := httptest.NewServer(router)
	path := fmt.Sprintf("%s/ofrep/v1/evaluate/flags", srv.URL)

	t.Run("options", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, path, http.NoBody)
		req.Header.Set(ofrep.SdkIdHeader, "test")
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
		req.Header.Set(ofrep.SdkIdHeader, "test")
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, `{"flags":[{"key":"flag","reason":"DEFAULT","variant":"v_flag","value":true}]}`, string(body))
		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
		assert.Empty(t, resp.Header.Get("h1"))
	})
}

func newOFREPRouter(t *testing.T, conf config.OFREPConfig) *HttpRouter {
	reg, _, _ := sdk.NewTestRegistrarT(t)
	return NewRouter(reg, nil, status.NewEmptyReporter(), &config.HttpConfig{OFREP: conf}, &config.ProfileConfig{}, log.NewNullLogger())
}
