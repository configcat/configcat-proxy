package sse

import (
	"context"
	"encoding/base64"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSSE_Get(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv := newServer(t, &config.SseConfig{Enabled: true})

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

func TestSSE_Get_SdkRemoved(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv, h := newServerWithAutoRegistrar(t, &config.SseConfig{Enabled: true})

	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag"}`))
	params := httprouter.Params{httprouter.Param{Key: streamDataName, Value: data}, httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)
	req = req.WithContext(ctx)

	// the removal closes the connection, so it won't block indefinitely
	h.RemoveSdk("test")
	testutils.WithTimeout(5*time.Second, func() {
		srv.SingleFlag(res, req)
	})

	assert.Equal(t, http.StatusOK, res.Code)
	// line breaks are intentional
	assert.Equal(t, `data: {"value":true,"variationId":"v_flag"}

`, res.Body.String())
	assert.Equal(t, "text/event-stream", res.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", res.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", res.Header().Get("Connection"))
}

func TestSSE_NonExisting_SDK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv := newServer(t, &config.SseConfig{Enabled: true})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag"}`))
	params := httprouter.Params{httprouter.Param{Key: streamDataName, Value: data}, httprouter.Param{Key: "sdkId", Value: "non-existing"}}
	ctx = context.WithValue(ctx, httprouter.ParamsKey, params)
	req = req.WithContext(ctx)

	res := httptest.NewRecorder()
	srv.SingleFlag(res, req)
	assert.Equal(t, http.StatusNotFound, res.Code)

	res = httptest.NewRecorder()
	srv.AllFlags(res, req)
	assert.Equal(t, http.StatusNotFound, res.Code)
}

func TestSSE_NonExisting_Flag(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv := newServer(t, &config.SseConfig{Enabled: true})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"non-existing"}`))
	params := httprouter.Params{httprouter.Param{Key: streamDataName, Value: data}, httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx = context.WithValue(ctx, httprouter.ParamsKey, params)
	req = req.WithContext(ctx)

	res := httptest.NewRecorder()
	srv.SingleFlag(res, req)
	assert.Equal(t, http.StatusBadRequest, res.Code)
}

func TestSSE_SDK_InvalidState(t *testing.T) {
	reg := sdk.NewTestRegistrar(&config.SDKConfig{BaseUrl: "http://localhost", Key: configcattest.RandomSDKKey()}, nil)
	defer reg.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv := NewServer(reg, nil, &config.SseConfig{Enabled: true}, log.NewNullLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag"}`))
	params := httprouter.Params{httprouter.Param{Key: streamDataName, Value: data}, httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx = context.WithValue(ctx, httprouter.ParamsKey, params)
	req = req.WithContext(ctx)

	res := httptest.NewRecorder()
	srv.SingleFlag(res, req)
	assert.Equal(t, http.StatusInternalServerError, res.Code)
	assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())

	res = httptest.NewRecorder()
	srv.AllFlags(res, req)
	assert.Equal(t, http.StatusInternalServerError, res.Code)
	assert.Equal(t, "SDK with identifier 'test' is in an invalid state; please check the logs for more details\n", res.Body.String())
}

func TestSSE_Get_All(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv := newServer(t, &config.SseConfig{Enabled: true})

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

func TestSSE_Get_All_SdkRemoved(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv, h := newServerWithAutoRegistrar(t, &config.SseConfig{Enabled: true})

	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag"}`))
	params := httprouter.Params{httprouter.Param{Key: streamDataName, Value: data}, httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)
	req = req.WithContext(ctx)

	// the removal closes the connection, so it won't block indefinitely
	h.RemoveSdk("test")
	testutils.WithTimeout(5*time.Second, func() {
		srv.AllFlags(res, req)
	})

	assert.Equal(t, http.StatusOK, res.Code)
	// line breaks are intentional
	assert.Equal(t, `data: {"flag":{"value":true,"variationId":"v_flag"}}

`, res.Body.String())
	assert.Equal(t, "text/event-stream", res.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", res.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", res.Header().Get("Connection"))
}

func TestSSE_Get_User(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv := newServer(t, &config.SseConfig{Enabled: true})

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

func TestSSE_Get_User_Invalid(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv := newServer(t, &config.SseConfig{Enabled: true})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag","user":{"Identifier":false}}`))
	params := httprouter.Params{httprouter.Param{Key: streamDataName, Value: data}, httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx = context.WithValue(ctx, httprouter.ParamsKey, params)
	req = req.WithContext(ctx)
	srv.SingleFlag(res, req)

	assert.Equal(t, http.StatusBadRequest, res.Code)
	assert.Contains(t, res.Body.String(), "failed to deserialize incoming 'data': 'Identifier' has an invalid type, only 'string', 'number', and 'string[]' types are allowed")
}

func TestSSE_Get_All_User(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv := newServer(t, &config.SseConfig{Enabled: true})

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

func TestSSE_Get_All_User_Invalid(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv := newServer(t, &config.SseConfig{Enabled: true})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	data := base64.URLEncoding.EncodeToString([]byte(`{"key":"flag","user":{"Identifier":false}}`))
	params := httprouter.Params{httprouter.Param{Key: streamDataName, Value: data}, httprouter.Param{Key: "sdkId", Value: "test"}}
	ctx = context.WithValue(ctx, httprouter.ParamsKey, params)
	req = req.WithContext(ctx)
	srv.AllFlags(res, req)

	assert.Equal(t, http.StatusBadRequest, res.Code)
	assert.Contains(t, res.Body.String(), "failed to deserialize incoming 'data': 'Identifier' has an invalid type, only 'string', 'number', and 'string[]' types are allowed")
}

func newServer(t *testing.T, conf *config.SseConfig) *Server {
	reg, _, _ := sdk.NewTestRegistrarT(t)
	server := NewServer(reg, nil, conf, log.NewNullLogger())
	t.Cleanup(func() {
		server.Close()
	})
	return server
}

func newServerWithAutoRegistrar(t *testing.T, conf *config.SseConfig) (*Server, *sdk.TestSdkRegistrarHandler) {
	reg, h := sdk.NewTestAutoRegistrarWithAutoConfig(t, config.AutoSDKConfig{PollInterval: 1}, log.NewNullLogger())
	server := NewServer(reg, nil, conf, log.NewNullLogger())
	t.Cleanup(func() {
		server.Close()
	})
	return server, h
}
