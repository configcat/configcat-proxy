package cache

import (
	"context"
	"crypto/sha1"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store/cache/redis"
	"github.com/configcat/configcat-proxy/status"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRedisNotify(t *testing.T) {
	s := miniredis.RunT(t)
	r := redis.NewRedisStorage("key", config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage(r, config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	cacheKey := produceCacheKey("key")
	err := s.Set(cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	assert.NoError(t, err)
	utils.WithTimeout(2*time.Second, func() {
		<-srv.Modified()
	})
	s.CheckGet(t, cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(r.LoadEntry().CachedJson))
}

func TestRedisNotify_Initial(t *testing.T) {
	s := miniredis.RunT(t)
	cacheKey := produceCacheKey("key")
	err := s.Set(cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	assert.NoError(t, err)
	r := redis.NewRedisStorage("key", config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage(r, config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	s.CheckGet(t, cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(r.LoadEntry().CachedJson))
}

func TestRedisNotify_Notify(t *testing.T) {
	s := miniredis.RunT(t)
	cacheKey := produceCacheKey("key")
	err := s.Set(cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	assert.NoError(t, err)
	r := redis.NewRedisStorage("key", config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage(r, config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	s.CheckGet(t, cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(r.LoadEntry().CachedJson))

	err = s.Set(cacheKey, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`)
	utils.WithTimeout(2*time.Second, func() {
		<-srv.Modified()
	})
	res, err = srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(res))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(r.LoadEntry().CachedJson))
}

func TestRedisNotify_BadJson(t *testing.T) {
	s := miniredis.RunT(t)
	cacheKey := produceCacheKey("key")
	err := s.Set(cacheKey, `{"k":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	assert.NoError(t, err)
	r := redis.NewRedisStorage("key", config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage(r, config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{}}`, string(res))
	assert.Equal(t, `{"f":{}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{}}`, string(r.LoadEntry().CachedJson))
}

func TestRedisNotify_MalformedJson(t *testing.T) {
	s := miniredis.RunT(t)
	cacheKey := fmt.Sprintf("%x", sha1.Sum([]byte("key_config_v5")))
	err := s.Set(cacheKey, `{"k":{"flag`)
	assert.NoError(t, err)
	r := redis.NewRedisStorage("key", config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage(r, config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{}}`, string(res))
	assert.Equal(t, `{"f":{}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{}}`, string(r.LoadEntry().CachedJson))
}

func TestRedisNotify_Unavailable(t *testing.T) {
	r := redis.NewRedisStorage("key", config.RedisConfig{Addresses: []string{"nonexisting"}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage(r, config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger())
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{}}`, string(res))
	assert.Equal(t, `{"f":{}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{}}`, string(r.LoadEntry().CachedJson))
}

func TestRedisNotify_Close(t *testing.T) {
	s := miniredis.RunT(t)
	r := redis.NewRedisStorage("key", config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter())
	srv := NewNotifyingCacheStorage(r, config.OfflineConfig{CachePollInterval: 1}, status.NewNullReporter(), log.NewNullLogger()).(*notifyingCacheStorage)
	go func() {
		srv.Close()
	}()
	utils.WithTimeout(2*time.Second, func() {
		select {
		case <-srv.stop:
		case <-srv.CacheStorage.Modified():
		}
	})
}

func produceCacheKey(key string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("%s_%s", key, "config_v5"))))
}
