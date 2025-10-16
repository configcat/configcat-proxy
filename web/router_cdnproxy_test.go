package web

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/stretchr/testify/assert"
)

func TestCDNProxy_Integration(t *testing.T) {
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

	reg.RefreshAll()
	router := NewRouter(reg, telemetry.NewEmptyReporter(), status.NewEmptyReporter(), &config.HttpConfig{CdnProxy: config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}}}, &config.ProfileConfig{}, log.NewNullLogger())
	srv := httptest.NewServer(router)
	defer srv.Close()

	user := &configcat.UserData{Identifier: "id"}

	t.Run("sdk-key", func(t *testing.T) {
		client := configcat.NewCustomClient(configcat.Config{
			BaseURL: srv.URL,
			SDKKey:  key,
		})
		defer client.Close()

		boolVal := client.GetBoolValueDetails("bool", false, user)
		assert.True(t, boolVal.Value)
		assert.Equal(t, "v0_bool", boolVal.Data.VariationID)
		assert.NotNil(t, boolVal.Data.MatchedTargetingRule)
		assert.Equal(t, "bool", boolVal.Data.Key)

		strVal := client.GetStringValueDetails("str", "", user)
		assert.Equal(t, "test", strVal.Value)
		assert.Equal(t, "v0_str", strVal.Data.VariationID)
		assert.NotNil(t, strVal.Data.MatchedTargetingRule)
		assert.Equal(t, "str", strVal.Data.Key)

		intVal := client.GetIntValueDetails("int", 0, user)
		assert.Equal(t, 42, intVal.Value)
		assert.Equal(t, "v0_int", intVal.Data.VariationID)
		assert.NotNil(t, intVal.Data.MatchedTargetingRule)
		assert.Equal(t, "int", intVal.Data.Key)

		doubleVal := client.GetFloatValueDetails("double", 0.0, user)
		assert.Equal(t, 3.14, doubleVal.Value)
		assert.Equal(t, "v0_double", doubleVal.Data.VariationID)
		assert.NotNil(t, doubleVal.Data.MatchedTargetingRule)
		assert.Equal(t, "double", doubleVal.Data.Key)
	})

	t.Run("sdk-id", func(t *testing.T) {
		client := configcat.NewCustomClient(configcat.Config{
			BaseURL: srv.URL,
			SDKKey:  "configcat-proxy/test",
		})
		defer client.Close()

		boolVal := client.GetBoolValueDetails("bool", false, user)
		assert.True(t, boolVal.Value)
		assert.Equal(t, "v0_bool", boolVal.Data.VariationID)
		assert.NotNil(t, boolVal.Data.MatchedTargetingRule)
		assert.Equal(t, "bool", boolVal.Data.Key)

		strVal := client.GetStringValueDetails("str", "", user)
		assert.Equal(t, "test", strVal.Value)
		assert.Equal(t, "v0_str", strVal.Data.VariationID)
		assert.NotNil(t, strVal.Data.MatchedTargetingRule)
		assert.Equal(t, "str", strVal.Data.Key)

		intVal := client.GetIntValueDetails("int", 0, user)
		assert.Equal(t, 42, intVal.Value)
		assert.Equal(t, "v0_int", intVal.Data.VariationID)
		assert.NotNil(t, intVal.Data.MatchedTargetingRule)
		assert.Equal(t, "int", intVal.Data.Key)

		doubleVal := client.GetFloatValueDetails("double", 0.0, user)
		assert.Equal(t, 3.14, doubleVal.Value)
		assert.Equal(t, "v0_double", doubleVal.Data.VariationID)
		assert.NotNil(t, doubleVal.Data.MatchedTargetingRule)
		assert.Equal(t, "double", doubleVal.Data.Key)
	})
}

func TestCDNProxy_Options_CORS(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}})
	srv := httptest.NewServer(router)
	req, _ := http.NewRequest(http.MethodOptions, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	assert.Equal(t, "GET,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "false", resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Cache-Control,Content-Type,Content-Length,Accept-Encoding,If-None-Match", resp.Header.Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "600", resp.Header.Get("Access-Control-Max-Age"))
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
	assert.Equal(t, "v1", resp.Header.Get("h1"))
}

func TestCDNProxy_Options_NO_CORS(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: false}})
	srv := httptest.NewServer(router)
	req, _ := http.NewRequest(http.MethodOptions, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Headers"))
	assert.Empty(t, resp.Header.Get("Access-Control-Max-Age"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
	assert.Empty(t, resp.Header.Get("h1"))
}

func TestCDNProxy_GET_CORS(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}})
	srv := httptest.NewServer(router)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Headers"))
	assert.Empty(t, resp.Header.Get("Access-Control-Max-Age"))
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Content-Length,ETag,Date,Content-Encoding,h1", resp.Header.Get("Access-Control-Expose-Headers"))
	assert.Equal(t, "v1", resp.Header.Get("h1"))
}

func TestCDNProxy_GET_NO_CORS(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: false}})
	srv := httptest.NewServer(router)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Headers"))
	assert.Empty(t, resp.Header.Get("Access-Control-Max-Age"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp.Header.Get("Access-Control-Expose-Headers"))
	assert.Empty(t, resp.Header.Get("h1"))
}

func TestCDNProxy_Not_Allowed_Methods(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}})
	srv := httptest.NewServer(router)

	t.Run("put", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("post", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("delete", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
	t.Run("patch", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
		resp, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestCDNProxy_Get_Body(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}})
	srv := httptest.NewServer(router)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[{"s":{"v":{"b":false,"s":null,"i":null,"d":null},"i":"v0_flag"},"c":[{"u":{"a":"Identifier","s":"test","d":null,"l":null,"c":28},"s":null,"p":null}],"p":null}],"p":null}},"s":null,"p":null}`, string(body))
	assert.Equal(t, "v1", resp.Header.Get("h1"))
}

func TestCDNProxy_GetByKey_Body(t *testing.T) {
	router, sdkKey := newCDNProxyRouterWithSdkKey(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}})
	srv := httptest.NewServer(router)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/configuration-files/%s/config_v6.json", srv.URL, sdkKey), http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[{"s":{"v":{"b":false,"s":null,"i":null,"d":null},"i":"v0_flag"},"c":[{"u":{"a":"Identifier","s":"test","d":null,"l":null,"c":28},"s":null,"p":null}],"p":null}],"p":null}},"s":null,"p":null}`, string(body))
	assert.Equal(t, "v1", resp.Header.Get("h1"))
}

func TestCDNProxy_Get_Body_GZip(t *testing.T) {
	router := newCDNProxyRouter(t, config.CdnProxyConfig{Enabled: true, CORS: config.CORSConfig{Enabled: true}, Headers: map[string]string{"h1": "v1"}})
	srv := httptest.NewServer(router)
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/configuration-files/configcat-proxy/test/config_v6.json", srv.URL), http.NoBody)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
	gzipReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	body, _ := io.ReadAll(gzipReader)
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[{"s":{"v":{"b":false,"s":null,"i":null,"d":null},"i":"v0_flag"},"c":[{"u":{"a":"Identifier","s":"test","d":null,"l":null,"c":28},"s":null,"p":null}],"p":null}],"p":null}},"s":null,"p":null}`, string(body))
	assert.Equal(t, "v1", resp.Header.Get("h1"))
}

func newCDNProxyRouter(t *testing.T, conf config.CdnProxyConfig) *HttpRouter {
	reg, _, _ := sdk.NewTestRegistrarT(t)
	return NewRouter(reg, telemetry.NewEmptyReporter(), status.NewEmptyReporter(), &config.HttpConfig{CdnProxy: conf}, &config.ProfileConfig{}, log.NewNullLogger())
}

func newCDNProxyRouterWithSdkKey(t *testing.T, conf config.CdnProxyConfig) (*HttpRouter, string) {
	reg, _, sdkKey := sdk.NewTestRegistrarT(t)
	return NewRouter(reg, telemetry.NewEmptyReporter(), status.NewEmptyReporter(), &config.HttpConfig{CdnProxy: conf}, &config.ProfileConfig{}, log.NewNullLogger()), sdkKey
}
