package cache

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store/cache/redis"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v8/configcatcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRedisNotify(t *testing.T) {
	s := miniredis.RunT(t)
	r := redis.NewRedisStorage(&config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage("test", "key", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	cacheKey := configcatcache.ProduceCacheKey("key")
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`))
	err := s.Set(cacheKey, string(cacheEntry))
	assert.NoError(t, err)
	utils.WithTimeout(2*time.Second, func() {
		<-srv.Modified()
	})
	s.CheckGet(t, cacheKey, string(cacheEntry))
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_Initial(t *testing.T) {
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey("key")
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`))
	err := s.Set(cacheKey, string(cacheEntry))
	assert.NoError(t, err)
	r := redis.NewRedisStorage(&config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage("test", "key", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	s.CheckGet(t, cacheKey, string(cacheEntry))
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_Notify(t *testing.T) {
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey("key")
	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`))
	err := s.Set(cacheKey, string(cacheEntry))
	assert.NoError(t, err)
	r := redis.NewRedisStorage(&config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage("test", "key", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	s.CheckGet(t, cacheKey, string(cacheEntry))
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(r.LoadEntry().ConfigJson))

	cacheEntry = configcatcache.CacheSegmentsToBytes(time.Now(), "etag2", []byte(`{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`))
	err = s.Set(cacheKey, string(cacheEntry))
	utils.WithTimeout(2*time.Second, func() {
		<-srv.Modified()
	})
	res, err = srv.Get(context.Background(), "")
	_, _, j, _ = configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_BadJson(t *testing.T) {
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey("key")
	err := s.Set(cacheKey, `{"k":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	assert.NoError(t, err)
	r := redis.NewRedisStorage(&config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage("test", "key", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{}}`, string(j))
	assert.Equal(t, `{"f":{}}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_MalformedJson(t *testing.T) {
	s := miniredis.RunT(t)
	cacheKey := configcatcache.ProduceCacheKey("key")
	err := s.Set(cacheKey, `{"k":{"flag`)
	assert.NoError(t, err)
	r := redis.NewRedisStorage(&config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage("test", "key", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{}}`, string(j))
	assert.Equal(t, `{"f":{}}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_Unavailable(t *testing.T) {
	r := redis.NewRedisStorage(&config.RedisConfig{Addresses: []string{"nonexisting"}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage("test", "key", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	_, _, j, _ := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{}}`, string(j))
	assert.Equal(t, `{"f":{}}`, string(r.LoadEntry().ConfigJson))
}

func TestRedisNotify_Close(t *testing.T) {
	s := miniredis.RunT(t)
	r := redis.NewRedisStorage(&config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage("test", "key", r, &config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger()).(*notifyingCacheStorage)
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
