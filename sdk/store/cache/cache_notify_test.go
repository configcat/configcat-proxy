package cache

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store/cache/redis"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRedisNotify(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	r := NewCacheStore(redis.NewRedisStore(&config.RedisConfig{Addresses: []string{s.Addr()}}), status.NewNullReporter(), sdkKey, config.V6)
	srv := NewNotifyingCacheStore("test", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger()).(*notifyingCacheStore)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"v":{"b":true}}},"p":null}`))
	err := s.Set(cacheKey, string(cacheEntry))
	assert.NoError(t, err)
	utils.WithTimeout(2*time.Second, func() {
		<-srv.Modified()
	})
	s.CheckGet(t, cacheKey, string(cacheEntry))
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(r.LoadEntry(config.V6).ConfigJson))
}

func TestRedisNotify_Initial(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"v":{"b":true}}},"p":null}`))
	err := s.Set(cacheKey, string(cacheEntry))
	assert.NoError(t, err)
	r := NewCacheStore(redis.NewRedisStore(&config.RedisConfig{Addresses: []string{s.Addr()}}), status.NewNullReporter(), sdkKey, config.V6)
	srv := NewNotifyingCacheStore("test", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	s.CheckGet(t, cacheKey, string(cacheEntry))
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(r.LoadEntry(config.V6).ConfigJson))
}

func TestRedisNotify_Notify(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"v":{"b":false}}},"p":null}`))
	err := s.Set(cacheKey, string(cacheEntry))
	assert.NoError(t, err)
	r := NewCacheStore(redis.NewRedisStore(&config.RedisConfig{Addresses: []string{s.Addr()}}), status.NewNullReporter(), sdkKey, config.V6)
	srv := NewNotifyingCacheStore("test", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger()).(*notifyingCacheStore)
	s.CheckGet(t, cacheKey, string(cacheEntry))
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":false,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(r.LoadEntry(config.V6).ConfigJson))

	cacheEntry = configcatcache.CacheSegmentsToBytes(time.Now(), "etag2", []byte(`{"f":{"flag":{"v":{"b":true}}},"p":null}`))
	err = s.Set(cacheKey, string(cacheEntry))
	utils.WithTimeout(2*time.Second, func() {
		<-srv.Modified()
	})
	res, err = srv.Get(context.Background(), "")
	_, _, j, _ = configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"a":"","i":"","v":{"b":true,"s":null,"i":null,"d":null},"t":0,"r":null,"p":null}},"s":null,"p":null}`, string(r.LoadEntry(config.V6).ConfigJson))
}

func TestRedisNotify_BadJson(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	err := s.Set(cacheKey, `{"f":{"flag":{"v":{"b":false}}},"p":null}`)
	assert.NoError(t, err)
	r := NewCacheStore(redis.NewRedisStore(&config.RedisConfig{Addresses: []string{s.Addr()}}), status.NewNullReporter(), sdkKey, config.V6)
	srv := NewNotifyingCacheStore("test", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(r.LoadEntry(config.V6).ConfigJson))
}

func TestRedisNotify_MalformedJson(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion)
	err := s.Set(cacheKey, `{"k":{"flag`)
	assert.NoError(t, err)
	r := NewCacheStore(redis.NewRedisStore(&config.RedisConfig{Addresses: []string{s.Addr()}}), status.NewNullReporter(), sdkKey, config.V6)
	srv := NewNotifyingCacheStore("test", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(r.LoadEntry(config.V6).ConfigJson))
}

func TestRedisNotify_Unavailable(t *testing.T) {
	sdkKey := "key"
	r := NewCacheStore(redis.NewRedisStore(&config.RedisConfig{Addresses: []string{"nonexisting"}}), status.NewNullReporter(), sdkKey, config.V6)
	srv := NewNotifyingCacheStore("test", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(j))
	assert.Equal(t, `{"f":null,"s":null,"p":null}`, string(r.LoadEntry(config.V6).ConfigJson))
}

func TestRedisNotify_Close(t *testing.T) {
	sdkKey := "key"
	s := miniredis.RunT(t)
	r := NewCacheStore(redis.NewRedisStore(&config.RedisConfig{Addresses: []string{s.Addr()}}), status.NewNullReporter(), sdkKey, config.V6)
	srv := NewNotifyingCacheStore("test", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger()).(*notifyingCacheStore)
	go func() {
		srv.Close()
	}()
	utils.WithTimeout(2*time.Second, func() {
		select {
		case <-srv.Closed():
		case <-srv.Modified():
		}
	})
}
