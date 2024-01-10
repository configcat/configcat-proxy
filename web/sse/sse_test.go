package sse

import (
	"context"
	"encoding/base64"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
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
	assert.Contains(t, res.Body.String(), "Failed to deserialize incoming 'data': 'Identifier' has an invalid type, only 'string', 'number', and 'string[]' types are allowed")
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
	assert.Contains(t, res.Body.String(), "Failed to deserialize incoming 'data': 'Identifier' has an invalid type, only 'string', 'number', and 'string[]' types are allowed")
}

func newServer(t *testing.T, conf *config.SseConfig) *Server {
	client, _, _ := testutils.NewTestSdkClient(t)
	return NewServer(client, nil, conf, log.NewNullLogger())
}
