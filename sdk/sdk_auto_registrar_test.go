package sdk

import (
	"encoding/json"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/internal/testutils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/model"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/configcat/go-sdk/v9/configcattest"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func TestAutoRegistrar_Poll(t *testing.T) {
	reg, h := NewTestAutoRegistrarWithAutoConfig(t, config.AutoSDKConfig{PollInterval: 1}, log.NewNullLogger())

	sub := make(chan string)
	reg.Subscribe(sub)

	sdkClient := reg.GetSdkOrNil("test")
	assert.NotNil(t, sdkClient)

	res := sdkClient.Eval("flag", nil)
	assert.True(t, res.Value.(bool))

	h.AddSdk("test2")
	testutils.WaitUntil(5*time.Second, func() bool {
		return nil != reg.GetSdkOrNil("test2")
	})
	testutils.WaitUntil(5*time.Second, func() bool {
		return "test2" == <-sub
	})

	sdkClient = reg.GetSdkOrNil("test2")
	assert.NotNil(t, sdkClient)

	res = sdkClient.Eval("flag", nil)
	assert.True(t, res.Value.(bool))

	h.RemoveSdk("test")
	testutils.WaitUntil(5*time.Second, func() bool {
		return nil == reg.GetSdkOrNil("test")
	})
	testutils.WaitUntil(5*time.Second, func() bool {
		return "test" == <-sub
	})

	sdkClient = reg.GetSdkOrNil("test")
	assert.Nil(t, sdkClient)

	sdks := reg.GetAll()
	assert.Len(t, sdks, 1)
}

func TestAutoRegistrar_Refresh(t *testing.T) {
	reg, h := NewTestAutoRegistrarWithAutoConfig(t, config.AutoSDKConfig{PollInterval: 60}, log.NewNullLogger())

	sub := make(chan string)
	reg.Subscribe(sub)

	sdkClient := reg.GetSdkOrNil("test")
	assert.NotNil(t, sdkClient)

	res := sdkClient.Eval("flag", nil)
	assert.True(t, res.Value.(bool))

	h.AddSdk("test2")
	reg.Refresh()
	testutils.WaitUntil(5*time.Second, func() bool {
		return nil != reg.GetSdkOrNil("test2")
	})
	testutils.WaitUntil(5*time.Second, func() bool {
		return "test2" == <-sub
	})

	sdkClient = reg.GetSdkOrNil("test2")
	assert.NotNil(t, sdkClient)

	res = sdkClient.Eval("flag", nil)
	assert.True(t, res.Value.(bool))

	h.RemoveSdk("test")
	reg.Refresh()
	testutils.WaitUntil(5*time.Second, func() bool {
		return nil == reg.GetSdkOrNil("test")
	})
	testutils.WaitUntil(5*time.Second, func() bool {
		return "test" == <-sub
	})

	sdkClient = reg.GetSdkOrNil("test")
	assert.Nil(t, sdkClient)

	sdks := reg.GetAll()
	assert.Len(t, sdks, 1)
}

func TestAutoRegistrar_Modify_Global_Opts(t *testing.T) {
	reg, h := NewTestAutoRegistrarWithAutoConfig(t, config.AutoSDKConfig{PollInterval: 60}, log.NewNullLogger())

	sub := make(chan string)
	reg.Subscribe(sub)

	sdkClient := reg.GetSdkOrNil("test").(*client)
	assert.NotNil(t, sdkClient)
	assert.Equal(t, 60, sdkClient.sdkCtx.SDKConf.PollInterval)
	assert.Equal(t, "global", sdkClient.sdkCtx.SDKConf.DataGovernance)

	h.ModifyGlobalOpts(model.OptionsModel{
		PollInterval:   120,
		DataGovernance: "eu",
	})
	reg.Refresh()
	testutils.WaitUntil(5*time.Second, func() bool {
		return "test" == <-sub
	})

	// old sdk closed
	testutils.WithTimeout(1*time.Second, func() {
		<-sdkClient.ctx.Done()
	})

	sdkClient2 := reg.GetSdkOrNil("test").(*client)
	assert.NotNil(t, sdkClient2)
	assert.Equal(t, 120, sdkClient2.sdkCtx.SDKConf.PollInterval)
	assert.Equal(t, "eu", sdkClient2.sdkCtx.SDKConf.DataGovernance)
}

func TestAutoRegistrar_Config(t *testing.T) {
	reg, _ := NewTestAutoRegistrar(t, config.Config{
		SDKs: map[string]*config.SDKConfig{"test": {
			PollInterval:   10,
			BaseUrl:        "https://something-unexpected",
			Key:            "sdkKey",
			DataGovernance: "eu",
			DefaultAttrs:   model.UserAttrs{"a": "b"},
			Offline: config.OfflineConfig{
				Enabled:           true,
				CachePollInterval: 15,
				UseCache:          true,
			},
		}},
		AutoSDK: config.AutoSDKConfig{
			PollInterval: 10,
		},
	}, nil, log.NewNullLogger())

	sdkClient := reg.GetSdkOrNil("test").(*client)
	assert.NotNil(t, sdkClient)
	assert.Equal(t, 60, sdkClient.sdkCtx.SDKConf.PollInterval)
	assert.Equal(t, "global", sdkClient.sdkCtx.SDKConf.DataGovernance)
	assert.True(t, strings.HasPrefix(sdkClient.sdkCtx.SDKConf.BaseUrl, "http://127.0.0.1"))
	assert.Equal(t, model.UserAttrs{"a": "b"}, sdkClient.sdkCtx.SDKConf.DefaultAttrs)
	assert.True(t, sdkClient.sdkCtx.SDKConf.Offline.Enabled)
	assert.True(t, sdkClient.sdkCtx.SDKConf.Offline.UseCache)
	assert.Equal(t, 15, sdkClient.sdkCtx.SDKConf.Offline.CachePollInterval)
}

func TestAutoRegistrar_Cache_Poll(t *testing.T) {
	cache := miniredis.RunT(t)
	extCache := newRedisCache(cache.Addr())

	sdkKey := configcattest.RandomSDKKey()
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true},"t":0}}}`))
	_ = cache.Set(cacheKey, string(cacheEntry))

	autoConfig := model.ProxyConfigModel{
		SDKs: map[string]*model.SdkConfigModel{"test": {SDKKey: sdkKey}},
	}

	autConfigCacheEntry, _ := json.Marshal(autoConfig)
	_ = cache.Set("configcat-proxy-conf/test-reg", string(autConfigCacheEntry))

	reg := NewTestAutoRegistrarWithCache(t, 1, extCache, log.NewNullLogger())

	sub := make(chan string)
	reg.Subscribe(sub)

	sdkClient := reg.GetSdkOrNil("test").(*client)
	assert.NotNil(t, sdkClient)
	assert.Equal(t, 0, sdkClient.sdkCtx.SDKConf.PollInterval)
	assert.Equal(t, "", sdkClient.sdkCtx.SDKConf.DataGovernance)

	res := sdkClient.Eval("flag", nil)
	assert.True(t, res.Value.(bool))

	sdkKey2 := configcattest.RandomSDKKey()
	cacheKey2 := configcatcache.ProduceCacheKey(sdkKey2, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry2 := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true},"t":0}}}`))
	_ = cache.Set(cacheKey2, string(cacheEntry2))

	autoConfig = model.ProxyConfigModel{
		SDKs: map[string]*model.SdkConfigModel{"test": {SDKKey: sdkKey}, "test2": {SDKKey: sdkKey2}},
	}

	autConfigCacheEntry, _ = json.Marshal(autoConfig)
	_ = cache.Set("configcat-proxy-conf/test-reg", string(autConfigCacheEntry))

	testutils.WaitUntil(5*time.Second, func() bool {
		return nil != reg.GetSdkOrNil("test2")
	})
	testutils.WaitUntil(5*time.Second, func() bool {
		return "test2" == <-sub
	})

	sdkClient2 := reg.GetSdkOrNil("test2").(*client)
	assert.NotNil(t, sdkClient2)
	assert.Equal(t, 0, sdkClient2.sdkCtx.SDKConf.PollInterval)
	assert.Equal(t, "", sdkClient2.sdkCtx.SDKConf.DataGovernance)

	res2 := sdkClient2.Eval("flag", nil)
	assert.True(t, res2.Value.(bool))
}

func TestAutoRegistrar_Cache_Refresh(t *testing.T) {
	cache := miniredis.RunT(t)
	extCache := newRedisCache(cache.Addr())

	sdkKey := configcattest.RandomSDKKey()
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true},"t":0}}}`))
	_ = cache.Set(cacheKey, string(cacheEntry))

	autoConfig := model.ProxyConfigModel{
		SDKs: map[string]*model.SdkConfigModel{"test": {SDKKey: sdkKey}},
	}

	autConfigCacheEntry, _ := json.Marshal(autoConfig)
	_ = cache.Set("configcat-proxy-conf/test-reg", string(autConfigCacheEntry))

	reg := NewTestAutoRegistrarWithCache(t, 60, extCache, log.NewNullLogger())

	sub := make(chan string)
	reg.Subscribe(sub)

	sdkClient := reg.GetSdkOrNil("test").(*client)
	assert.NotNil(t, sdkClient)
	assert.Equal(t, 0, sdkClient.sdkCtx.SDKConf.PollInterval)
	assert.Equal(t, "", sdkClient.sdkCtx.SDKConf.DataGovernance)

	res := sdkClient.Eval("flag", nil)
	assert.True(t, res.Value.(bool))

	sdkKey2 := configcattest.RandomSDKKey()
	cacheKey2 := configcatcache.ProduceCacheKey(sdkKey2, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry2 := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true},"t":0}}}`))
	_ = cache.Set(cacheKey2, string(cacheEntry2))

	autoConfig = model.ProxyConfigModel{
		SDKs: map[string]*model.SdkConfigModel{"test": {SDKKey: sdkKey}, "test2": {SDKKey: sdkKey2}},
	}

	autConfigCacheEntry, _ = json.Marshal(autoConfig)
	_ = cache.Set("configcat-proxy-conf/test-reg", string(autConfigCacheEntry))

	reg.Refresh()
	testutils.WaitUntil(5*time.Second, func() bool {
		return nil != reg.GetSdkOrNil("test2")
	})
	testutils.WaitUntil(5*time.Second, func() bool {
		return "test2" == <-sub
	})

	sdkClient2 := reg.GetSdkOrNil("test2").(*client)
	assert.NotNil(t, sdkClient2)
	assert.Equal(t, 0, sdkClient2.sdkCtx.SDKConf.PollInterval)
	assert.Equal(t, "", sdkClient2.sdkCtx.SDKConf.DataGovernance)

	res2 := sdkClient2.Eval("flag", nil)
	assert.True(t, res2.Value.(bool))
}

func TestAutoRegistrar_Cache_When_Fail(t *testing.T) {
	cache := miniredis.RunT(t)
	extCache := newRedisCache(cache.Addr())

	sdkKey := configcattest.RandomSDKKey()
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"a":"","i":"v_flag","v":{"b":true},"t":0}}}`))
	_ = cache.Set(cacheKey, string(cacheEntry))

	autoConfig := model.ProxyConfigModel{
		SDKs: map[string]*model.SdkConfigModel{"test": {SDKKey: sdkKey}},
	}

	autConfigCacheEntry, _ := json.Marshal(autoConfig)
	_ = cache.Set("configcat-proxy-conf/test-reg", string(autConfigCacheEntry))

	conf := config.Config{AutoSDK: config.AutoSDKConfig{Key: "test-reg", PollInterval: 60}}
	reg, _ := newAutoRegistrar(&conf, nil, status.NewEmptyReporter(), extCache, log.NewNullLogger())
	defer reg.Close()

	sdkClient := reg.GetSdkOrNil("test").(*client)
	assert.NotNil(t, sdkClient)
	assert.Equal(t, 0, sdkClient.sdkCtx.SDKConf.PollInterval)
	assert.Equal(t, "", sdkClient.sdkCtx.SDKConf.DataGovernance)

	res := sdkClient.Eval("flag", nil)
	assert.True(t, res.Value.(bool))
}

func TestAutoRegistrar_Saves_To_Cache(t *testing.T) {
	cache := miniredis.RunT(t)
	extCache := newRedisCache(cache.Addr())

	conf := config.Config{AutoSDK: config.AutoSDKConfig{Key: "test-reg", PollInterval: 60}}
	reg, _ := NewTestAutoRegistrar(t, conf, extCache, log.NewNullLogger())

	sdkClient := reg.GetSdkOrNil("test").(*client)
	assert.NotNil(t, sdkClient)

	cached, _ := cache.Get("configcat-proxy-conf/test-reg")
	assert.Equal(t, `{"SDKs":{"test":{"SDKKey":"`+sdkClient.sdkCtx.SDKConf.Key+`"}},"Options":{"PollInterval":60,"DataGovernance":"global"}}`, cached)
}
