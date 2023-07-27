package sse

import (
	"context"
	"encoding/base64"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v8/configcattest"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSSE_Get(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h = &configcattest.Handler{}
	_ = h.SetFlags(key, map[string]*configcattest.Flag{"flag": {Default: true}})
	client := newClient(t, h, key)
	srv := NewServer(map[string]sdk.Client{"test": client}, nil, &config.SseConfig{Enabled: true}, log.NewNullLogger())
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag"}`))
	params := httprouter.Params{httprouter.Param{Key: streamDataName, Value: data}, httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx = context.WithValue(ctx, httprouter.ParamsKey, params)
	req = req.WithContext(ctx)
	srv.SingleFlag(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	// line breaks are intentional
	assert.Equal(t, `data: {"value":true,"variationId":"v_flag"}

`, res.Body.String())
	assert.Equal(t, "text/event-stream", res.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", res.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", res.Header().Get("Connection"))
}

func TestSSE_Get_All(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h = &configcattest.Handler{}
	_ = h.SetFlags(key, map[string]*configcattest.Flag{"flag": {Default: true}})
	client := newClient(t, h, key)
	srv := NewServer(map[string]sdk.Client{"test": client}, nil, &config.SseConfig{Enabled: true}, log.NewNullLogger())
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag"}`))
	params := httprouter.Params{httprouter.Param{Key: streamDataName, Value: data}, httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx = context.WithValue(ctx, httprouter.ParamsKey, params)
	req = req.WithContext(ctx)
	srv.AllFlags(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	// line breaks are intentional
	assert.Equal(t, `data: {"flag":{"value":true,"variationId":"v_flag"}}

`, res.Body.String())
	assert.Equal(t, "text/event-stream", res.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", res.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", res.Header().Get("Connection"))
}

func TestSSE_Get_User(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h = &configcattest.Handler{}
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
			Rules: []configcattest.Rule{
				{Value: false, Comparator: configcattest.OpContains, ComparisonValue: "test", ComparisonAttribute: "Identifier"},
			},
		},
	})
	client := newClient(t, h, key)
	srv := NewServer(map[string]sdk.Client{"test": client}, nil, &config.SseConfig{Enabled: true}, log.NewNullLogger())
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag","user":{"Identifier":"test"}}`))
	params := httprouter.Params{httprouter.Param{Key: streamDataName, Value: data}, httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx = context.WithValue(ctx, httprouter.ParamsKey, params)
	req = req.WithContext(ctx)
	srv.SingleFlag(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	// line breaks are intentional
	assert.Equal(t, `data: {"value":false,"variationId":"v0_flag"}

`, res.Body.String())
	assert.Equal(t, "text/event-stream", res.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", res.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", res.Header().Get("Connection"))
}

func TestSSE_Get_All_User(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h = &configcattest.Handler{}
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
			Rules: []configcattest.Rule{
				{Value: false, Comparator: configcattest.OpContains, ComparisonValue: "test", ComparisonAttribute: "Identifier"},
			},
		},
	})
	client := newClient(t, h, key)
	srv := NewServer(map[string]sdk.Client{"test": client}, nil, &config.SseConfig{Enabled: true}, log.NewNullLogger())
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag","user":{"Identifier":"test"}}`))
	params := httprouter.Params{httprouter.Param{Key: streamDataName, Value: data}, httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx = context.WithValue(ctx, httprouter.ParamsKey, params)
	req = req.WithContext(ctx)
	srv.AllFlags(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	// line breaks are intentional
	assert.Equal(t, `data: {"flag":{"value":false,"variationId":"v0_flag"}}

`, res.Body.String())
	assert.Equal(t, "text/event-stream", res.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", res.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", res.Header().Get("Connection"))
}

func newClient(t *testing.T, h *configcattest.Handler, key string) sdk.Client {
	srv := httptest.NewServer(h)
	ctx := testutils.NewTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	client := sdk.NewClient(ctx, log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return client
}
