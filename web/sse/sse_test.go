package sse

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v7/configcattest"
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
	srv := NewServer(client, nil, config.SseConfig{Enabled: true}, log.NewNullLogger())
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	params := httprouter.Params{httprouter.Param{Key: "key", Value: "flag"}}
	ctx = context.WithValue(ctx, httprouter.ParamsKey, params)
	req = req.WithContext(ctx)
	srv.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	// line breaks are intentional
	assert.Equal(t, `data: {"value":true,"variationId":"v_flag"}

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
	srv := NewServer(client, nil, config.SseConfig{Enabled: true}, log.NewNullLogger())
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	params := httprouter.Params{httprouter.Param{Key: "key", Value: "flag"}}
	ctx = context.WithValue(ctx, httprouter.ParamsKey, params)
	req = req.WithContext(ctx)
	q := req.URL.Query()
	q.Add("Identifier", "test")
	req.URL.RawQuery = q.Encode()
	srv.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	// line breaks are intentional
	assert.Equal(t, `data: {"value":false,"variationId":"v0_flag"}

`, res.Body.String())
	assert.Equal(t, "text/event-stream", res.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", res.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", res.Header().Get("Connection"))
}

func newClient(t *testing.T, h *configcattest.Handler, key string) sdk.Client {
	srv := httptest.NewServer(h)
	opts := config.SDKConfig{BaseUrl: srv.URL, Key: key}
	client := sdk.NewClient(opts, nil, status.NewNullReporter(), log.NewNullLogger())
	t.Cleanup(func() {
		srv.Close()
		client.Close()
	})
	return client
}
