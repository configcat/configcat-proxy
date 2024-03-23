package sdk

import (
	"context"
	"crypto/sha1"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/configcat-proxy/sdk/statistics"
	"github.com/configcat/configcat-proxy/sdk/store/cache"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSdk_Signal(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key, PollInterval: 1}, nil)
	client := NewClient(ctx, log.NewNullLogger())
	defer client.Close()
	sub := client.SubConfigChanged("id")
	data, err := client.Eval("flag", nil)
	j := client.GetCachedJson()
	assert.NoError(t, err)
	assert.True(t, data.Value.(bool))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[],"p":null}},"s":null,"p":null}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("%x", sha1.Sum(j.ConfigJson)), j.ETag)

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: false,
		},
	})
	utils.WithTimeout(2*time.Second, func() {
		<-sub
	})
	data, err = client.Eval("flag", nil)
	j = client.GetCachedJson()
	assert.NoError(t, err)
	assert.False(t, data.Value.(bool))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":[],"p":null}},"s":null,"p":null}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("%x", sha1.Sum(j.ConfigJson)), j.ETag)
}

func TestSdk_Ready_Online(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	client := NewClient(ctx, log.NewNullLogger())
	defer client.Close()
	utils.WithTimeout(2*time.Second, func() {
		<-client.Ready()
	})
	j := client.GetCachedJson()
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[],"p":null}},"s":null,"p":null}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("%x", sha1.Sum(j.ConfigJson)), j.ETag)
}

func TestSdk_Ready_Offline(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true},"t":0}}}`, func(path string) {
		ctx := newTestSdkContext(&config.SDKConfig{Key: "key", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path}}}, nil)
		client := NewClient(ctx, log.NewNullLogger())
		defer client.Close()
		utils.WithTimeout(2*time.Second, func() {
			<-client.Ready()
		})
		j := client.GetCachedJson()
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j.ConfigJson))
		assert.Equal(t, utils.GenerateEtag(j.ConfigJson), j.ETag)
	})
}

func TestSdk_Signal_Refresh(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	client := NewClient(ctx, log.NewNullLogger())
	defer client.Close()
	sub := client.SubConfigChanged("id")
	data, err := client.Eval("flag", nil)
	j := client.GetCachedJson()
	assert.NoError(t, err)
	assert.True(t, data.Value.(bool))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":[],"p":null}},"s":null,"p":null}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("%x", sha1.Sum(j.ConfigJson)), j.ETag)

	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: false,
		},
	})
	_ = client.Refresh()
	utils.WithTimeout(2*time.Second, func() {
		<-sub
	})
	data, err = client.Eval("flag", nil)
	j = client.GetCachedJson()
	assert.NoError(t, err)
	assert.False(t, data.Value.(bool))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":[],"p":null}},"s":null,"p":null}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("%x", sha1.Sum(j.ConfigJson)), j.ETag)
}

func TestSdk_BadConfig(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	srv := httptest.NewServer(&h)
	defer srv.Close()

	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key, Log: config.LogConfig{Level: "debug"}}, nil)
	client := NewClient(ctx, log.NewDebugLogger())
	defer client.Close()
	data, err := client.Eval("flag", nil)
	j := client.GetCachedJson()
	assert.Error(t, err)
	assert.Nil(t, data.Value)
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j.ConfigJson))
	assert.Equal(t, utils.GenerateEtag(j.ConfigJson), j.ETag)
}

func TestSdk_BadConfig_WithCache(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	srv := httptest.NewServer(&h)
	defer srv.Close()

	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(key, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true},"t":0}}}`))
	err := s.Set(cacheKey, string(cacheEntry))

	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key, Log: config.LogConfig{Level: "debug"}}, newRedisCache(s.Addr()))
	client := NewClient(ctx, log.NewDebugLogger())
	defer client.Close()
	data, err := client.Eval("flag", nil)
	j := client.GetCachedJson()
	assert.NoError(t, err)
	assert.True(t, data.Value.(bool))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true},"t":0}}}`, string(j.ConfigJson))
	assert.Equal(t, "etag", j.ETag)
}

func TestSdk_Signal_Offline_File_Watch(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true},"t":0}}}`, func(path string) {
		ctx := newTestSdkContext(&config.SDKConfig{Key: "key", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path}}}, nil)
		client := NewClient(ctx, log.NewNullLogger())
		defer client.Close()
		sub := client.SubConfigChanged("id")
		data, err := client.Eval("flag", nil)
		j := client.GetCachedJson()
		assert.NoError(t, err)
		assert.True(t, data.Value.(bool))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j.ConfigJson))
		assert.Equal(t, fmt.Sprintf("W/\"%s\"", utils.FastHashHex(j.ConfigJson)), j.ETag)

		utils.WriteIntoFile(path, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":false},"t":0}}}`)
		utils.WithTimeout(2*time.Second, func() {
			<-sub
		})
		data, err = client.Eval("flag", nil)
		j = client.GetCachedJson()
		assert.NoError(t, err)
		assert.False(t, data.Value.(bool))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j.ConfigJson))
		assert.Equal(t, utils.GenerateEtag(j.ConfigJson), j.ETag)
	})
}

func TestSdk_Signal_Offline_Poll_Watch(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true},"t":0}}}`, func(path string) {
		ctx := newTestSdkContext(&config.SDKConfig{Key: "key", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path, Polling: true, PollInterval: 1}}}, nil)
		client := NewClient(ctx, log.NewNullLogger())
		defer client.Close()
		sub := client.SubConfigChanged("id")
		data, err := client.Eval("flag", nil)
		j := client.GetCachedJson()
		assert.NoError(t, err)
		assert.True(t, data.Value.(bool))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j.ConfigJson))
		assert.Equal(t, fmt.Sprintf("W/\"%s\"", utils.FastHashHex(j.ConfigJson)), j.ETag)

		utils.WriteIntoFile(path, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":false},"t":0}}}`)
		utils.WithTimeout(2*time.Second, func() {
			<-sub
		})
		data, err = client.Eval("flag", nil)
		j = client.GetCachedJson()
		assert.NoError(t, err)
		assert.False(t, data.Value.(bool))
		assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j.ConfigJson))
		assert.Equal(t, utils.GenerateEtag(j.ConfigJson), j.ETag)
	})
}

func TestSdk_Signal_Offline_Redis_Watch(t *testing.T) {
	sdkKey := configcattest.RandomSDKKey()
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true},"t":0}}}`))
	_ = s.Set(cacheKey, string(cacheEntry))

	ctx := newTestSdkContext(&config.SDKConfig{
		Key:     sdkKey,
		Offline: config.OfflineConfig{Enabled: true, UseCache: true, CachePollInterval: 1},
	}, newRedisCache(s.Addr()))
	client := NewClient(ctx, log.NewNullLogger())
	defer client.Close()
	sub := client.SubConfigChanged("id")
	data, err := client.Eval("flag", nil)
	j := client.GetCachedJson()
	assert.NoError(t, err)
	assert.True(t, data.Value.(bool))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j.ConfigJson))
	assert.Equal(t, "etag", j.ETag)

	cacheEntry = configcatcache.CacheSegmentsToBytes(time.Now(), "etag2", []byte(`{"f":{"flag":{"a":"","i":"v_flag","v":{"b":false},"t":0}}}`))
	_ = s.Set(cacheKey, string(cacheEntry))
	utils.WithTimeout(2*time.Second, func() {
		<-sub
	})
	data, err = client.Eval("flag", nil)
	j = client.GetCachedJson()
	assert.NoError(t, err)
	assert.False(t, data.Value.(bool))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"v_flag","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j.ConfigJson))
	assert.Equal(t, "etag2", j.ETag)
}

func TestSdk_Sub_Unsub(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key, PollInterval: 1}, nil)
	client := NewClient(ctx, log.NewNullLogger()).(*client)
	defer client.Close()
	_ = client.SubConfigChanged("id")
	assert.NotEmpty(t, client.subscriptions)
	client.UnsubConfigChanged("id")
	assert.Empty(t, client.subscriptions)
}

func TestSdk_EvalAll(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag1": {
			Default: "v1",
		},
		"flag2": {
			Default: "v2",
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	client := NewClient(ctx, log.NewNullLogger())
	defer client.Close()

	details := client.EvalAll(nil)
	assert.Equal(t, 2, len(details))
	assert.Equal(t, "v1", details["flag1"].Value)
	assert.Equal(t, "v2", details["flag2"].Value)
}

func TestSdk_Keys(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag1": {
			Default: "v1",
		},
		"flag2": {
			Default: "v2",
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	client := NewClient(ctx, log.NewNullLogger())
	defer client.Close()

	keys := client.Keys()
	assert.Equal(t, 2, len(keys))
	assert.Equal(t, "flag1", keys[0])
	assert.Equal(t, "flag2", keys[1])
}

func TestSdk_EvalStatsReporter(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag1": {
			Default: "v1",
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	reporter := NewTestReporter()
	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	ctx.EvalReporter = reporter
	client := NewClient(ctx, log.NewNullLogger())
	defer client.Close()

	_, _ = client.Eval("flag1", model.UserAttrs{"e": "h"})

	var event *statistics.EvalEvent
	utils.WithTimeout(2*time.Second, func() {
		event = <-reporter.Latest()
	})
	assert.Equal(t, map[string]interface{}{"e": "h"}, event.UserAttrs)
}

func TestSdk_DefaultAttrs(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag1": {
			Default: "v1",
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key, DefaultAttrs: map[string]interface{}{"a": "g"}}, nil)
	ctx.GlobalDefaultAttrs = map[string]interface{}{"a": "b", "c": "d", "e": "f"}
	client := NewClient(ctx, log.NewNullLogger())
	defer client.Close()

	evalData, _ := client.Eval("flag1", model.UserAttrs{"e": "h"})
	assert.Equal(t, model.UserAttrs{"a": "g", "c": "d", "e": "h"}, evalData.User.(model.UserAttrs))
}

func TestSdk_WebHookParams(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key, WebhookSigningKey: "key", WebhookSignatureValidFor: 5}, nil)
	client := NewClient(ctx, log.NewNullLogger())
	defer client.Close()

	assert.Equal(t, "key", client.WebhookSigningKey())
	assert.Equal(t, 5, client.WebhookSignatureValidFor())
}

func TestSdk_IsInValidState_True(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	_ = h.SetFlags(key, map[string]*configcattest.Flag{
		"flag": {
			Default: true,
		},
	})
	srv := httptest.NewServer(&h)
	defer srv.Close()

	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key}, nil)
	client := NewClient(ctx, log.NewNullLogger())
	defer client.Close()

	assert.True(t, client.IsInValidState())
}

func TestSdk_IsInValidState_False(t *testing.T) {
	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: "https://localhost", Key: configcattest.RandomSDKKey()}, nil)
	client := NewClient(ctx, log.NewDebugLogger())
	defer client.Close()

	assert.False(t, client.IsInValidState())
}

func TestSdk_IsInValidState_EmptyCache_False(t *testing.T) {
	r := miniredis.RunT(t)
	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: "https://localhost", Key: configcattest.RandomSDKKey()}, newRedisCache(r.Addr()))
	client := NewClient(ctx, log.NewDebugLogger())
	defer client.Close()

	assert.False(t, client.IsInValidState())
}

func newTestSdkContext(conf *config.SDKConfig, externalCache configcat.ConfigCache) *Context {
	return &Context{
		SDKConf:         conf,
		ProxyConf:       &config.HttpProxyConfig{},
		StatusReporter:  status.NewNullReporter(),
		MetricsReporter: nil,
		EvalReporter:    nil,
		SdkId:           "test",
		ExternalCache:   externalCache,
	}
}

func newRedisCache(addr string) configcat.ConfigCache {
	c, _ := cache.SetupExternalCache(context.Background(), &config.CacheConfig{Redis: config.RedisConfig{Enabled: true, Addresses: []string{addr}}}, log.NewNullLogger())
	return c
}

type TestReporter struct {
	events chan *statistics.EvalEvent
}

func NewTestReporter() *TestReporter {
	return &TestReporter{events: make(chan *statistics.EvalEvent)}
}

func (r *TestReporter) ReportEvaluation(event *statistics.EvalEvent) {
	r.events <- event
}

func (r *TestReporter) Latest() <-chan *statistics.EvalEvent {
	return r.events
}

func (r *TestReporter) Close() {}
