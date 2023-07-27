package sdk

import (
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v8/configcatcache"
	"github.com/configcat/go-sdk/v8/configcattest"
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
	assert.Equal(t, `{"f":{"flag":{"i":"v_flag","v":true,"t":0,"r":[],"p":null}},"p":null}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)

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
	assert.Equal(t, `{"f":{"flag":{"i":"v_flag","v":false,"t":0,"r":[],"p":null}},"p":null}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)
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
	assert.Equal(t, `{"f":{"flag":{"i":"v_flag","v":true,"t":0,"r":[],"p":null}},"p":null}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)
}

func TestSdk_Ready_Offline(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, func(path string) {
		ctx := newTestSdkContext(&config.SDKConfig{Key: "key", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path}}}, nil)
		client := NewClient(ctx, log.NewNullLogger())
		defer client.Close()
		utils.WithTimeout(2*time.Second, func() {
			<-client.Ready()
		})
		j := client.GetCachedJson()
		assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(j.ConfigJson))
		assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)
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
	assert.Equal(t, `{"f":{"flag":{"i":"v_flag","v":true,"t":0,"r":[],"p":null}},"p":null}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)

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
	assert.Equal(t, `{"f":{"flag":{"i":"v_flag","v":false,"t":0,"r":[],"p":null}},"p":null}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)
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
	assert.Equal(t, `{"f":{}}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)
}

func TestSdk_BadConfig_WithCache(t *testing.T) {
	key := configcattest.RandomSDKKey()
	var h configcattest.Handler
	srv := httptest.NewServer(&h)
	defer srv.Close()

	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(key)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`))
	err := s.Set(cacheKey, string(cacheEntry))

	ctx := newTestSdkContext(&config.SDKConfig{BaseUrl: srv.URL, Key: key, Log: config.LogConfig{Level: "debug"}}, &config.CacheConfig{Redis: config.RedisConfig{Enabled: true, Addresses: []string{s.Addr()}}})
	client := NewClient(ctx, log.NewDebugLogger())
	defer client.Close()
	data, err := client.Eval("flag", nil)
	j := client.GetCachedJson()
	assert.NoError(t, err)
	assert.True(t, data.Value.(bool))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)
}

func TestSdk_Signal_Offline_File_Watch(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, func(path string) {
		ctx := newTestSdkContext(&config.SDKConfig{Key: "key", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path}}}, nil)
		client := NewClient(ctx, log.NewNullLogger())
		defer client.Close()
		sub := client.SubConfigChanged("id")
		data, err := client.Eval("flag", nil)
		j := client.GetCachedJson()
		assert.NoError(t, err)
		assert.True(t, data.Value.(bool))
		assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(j.ConfigJson))
		assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)

		utils.WriteIntoFile(path, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
		utils.WithTimeout(2*time.Second, func() {
			<-sub
		})
		data, err = client.Eval("flag", nil)
		j = client.GetCachedJson()
		assert.NoError(t, err)
		assert.False(t, data.Value.(bool))
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(j.ConfigJson))
		assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)
	})
}

func TestSdk_Signal_Offline_Poll_Watch(t *testing.T) {
	utils.UseTempFile(`{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, func(path string) {
		ctx := newTestSdkContext(&config.SDKConfig{Key: "key", Offline: config.OfflineConfig{Enabled: true, Local: config.LocalConfig{FilePath: path, Polling: true, PollInterval: 1}}}, nil)
		client := NewClient(ctx, log.NewNullLogger())
		defer client.Close()
		sub := client.SubConfigChanged("id")
		data, err := client.Eval("flag", nil)
		j := client.GetCachedJson()
		assert.NoError(t, err)
		assert.True(t, data.Value.(bool))
		assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(j.ConfigJson))
		assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)

		utils.WriteIntoFile(path, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
		utils.WithTimeout(2*time.Second, func() {
			<-sub
		})
		data, err = client.Eval("flag", nil)
		j = client.GetCachedJson()
		assert.NoError(t, err)
		assert.False(t, data.Value.(bool))
		assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(j.ConfigJson))
		assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)
	})
}

func TestSdk_Signal_Offline_Redis_Watch(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`))
	_ = s.Set(cacheKey, string(cacheEntry))

	ctx := newTestSdkContext(&config.SDKConfig{
		Key:     sdkKey,
		Offline: config.OfflineConfig{Enabled: true, UseCache: true, CachePollInterval: 1},
	}, &config.CacheConfig{Redis: config.RedisConfig{Enabled: true, Addresses: []string{s.Addr()}}})
	client := NewClient(ctx, log.NewNullLogger())
	defer client.Close()
	sub := client.SubConfigChanged("id")
	data, err := client.Eval("flag", nil)
	j := client.GetCachedJson()
	assert.NoError(t, err)
	assert.True(t, data.Value.(bool))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)

	cacheEntry = configcatcache.CacheSegmentsToBytes(time.Now(), "etag2", []byte(`{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`))
	_ = s.Set(cacheKey, string(cacheEntry))
	utils.WithTimeout(2*time.Second, func() {
		<-sub
	})
	data, err = client.Eval("flag", nil)
	j = client.GetCachedJson()
	assert.NoError(t, err)
	assert.False(t, data.Value.(bool))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(j.ConfigJson))
	assert.Equal(t, fmt.Sprintf("W/\"%x\"", utils.FastHash(j.ConfigJson)), j.GeneratedETag)
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

func newTestSdkContext(conf *config.SDKConfig, cacheConf *config.CacheConfig) *Context {
	if cacheConf == nil {
		cacheConf = &config.CacheConfig{}
	}
	return &Context{
		SDKConf:        conf,
		ProxyConf:      &config.HttpProxyConfig{},
		CacheConf:      cacheConf,
		StatusReporter: status.NewNullReporter(),
		MetricsHandler: nil,
		EvalReporter:   nil,
		SdkId:          "test",
	}
}
