package redis

import (
	"context"
	"crypto/sha1"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRedisNotify(t *testing.T) {
	s := miniredis.RunT(t)
	srv := NewNotifyingRedisStorage("key", config.SDKConfig{Cache: config.CacheConfig{
		Redis: config.RedisConfig{Addresses: []string{s.Addr()}},
	}, Offline: config.OfflineConfig{CachePollInterval: 1}}, log.NewNullLogger()).(*notifyingRedisStorage)
	err := s.Set(srv.cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	assert.NoError(t, err)
	utils.WithTimeout(2*time.Second, func() {
		<-srv.Modified()
	})
	s.CheckGet(t, srv.cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.LoadEntry().CachedJson))
}

func TestRedisNotify_Initial(t *testing.T) {
	s := miniredis.RunT(t)
	cacheKey := fmt.Sprintf("%x", sha1.Sum([]byte("key_config_v5")))
	err := s.Set(cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	assert.NoError(t, err)
	srv := NewNotifyingRedisStorage("key", config.SDKConfig{Cache: config.CacheConfig{
		Redis: config.RedisConfig{Addresses: []string{s.Addr()}},
	}, Offline: config.OfflineConfig{CachePollInterval: 1}}, log.NewNullLogger()).(*notifyingRedisStorage)
	s.CheckGet(t, srv.cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.LoadEntry().CachedJson))
}

func TestRedisNotify_Notify(t *testing.T) {
	s := miniredis.RunT(t)
	cacheKey := fmt.Sprintf("%x", sha1.Sum([]byte("key_config_v5")))
	err := s.Set(cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	assert.NoError(t, err)
	srv := NewNotifyingRedisStorage("key", config.SDKConfig{Cache: config.CacheConfig{
		Redis: config.RedisConfig{Addresses: []string{s.Addr()}},
	}, Offline: config.OfflineConfig{CachePollInterval: 1}}, log.NewNullLogger()).(*notifyingRedisStorage)
	s.CheckGet(t, srv.cacheKey, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(res))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.LoadEntry().CachedJson))

	err = s.Set(cacheKey, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`)
	utils.WithTimeout(2*time.Second, func() {
		<-srv.Modified()
	})
	res, err = srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(res))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":true,"t":0,"r":[],"p":[]}}}`, string(srv.LoadEntry().CachedJson))
}

func TestRedisNotify_BadJson(t *testing.T) {
	s := miniredis.RunT(t)
	cacheKey := fmt.Sprintf("%x", sha1.Sum([]byte("key_config_v5")))
	err := s.Set(cacheKey, `{"k":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`)
	assert.NoError(t, err)
	srv := NewNotifyingRedisStorage("key", config.SDKConfig{Cache: config.CacheConfig{
		Redis: config.RedisConfig{Addresses: []string{s.Addr()}},
	}, Offline: config.OfflineConfig{CachePollInterval: 1}}, log.NewNullLogger()).(*notifyingRedisStorage)
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{}}`, string(res))
	assert.Equal(t, `{"f":{}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{}}`, string(srv.LoadEntry().CachedJson))
}

func TestRedisNotify_MalformedJson(t *testing.T) {
	s := miniredis.RunT(t)
	cacheKey := fmt.Sprintf("%x", sha1.Sum([]byte("key_config_v5")))
	err := s.Set(cacheKey, `{"k":{"flag`)
	assert.NoError(t, err)
	srv := NewNotifyingRedisStorage("key", config.SDKConfig{Cache: config.CacheConfig{
		Redis: config.RedisConfig{Addresses: []string{s.Addr()}},
	}, Offline: config.OfflineConfig{CachePollInterval: 1}}, log.NewNullLogger()).(*notifyingRedisStorage)
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{}}`, string(res))
	assert.Equal(t, `{"f":{}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{}}`, string(srv.LoadEntry().CachedJson))
}

func TestRedisNotify_Unavailable(t *testing.T) {
	srv := NewNotifyingRedisStorage("key", config.SDKConfig{Cache: config.CacheConfig{
		Redis: config.RedisConfig{Addresses: []string{"nonexisting"}},
	}, Offline: config.OfflineConfig{CachePollInterval: 1}}, log.NewNullLogger()).(*notifyingRedisStorage)
	res, err := srv.Get(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{}}`, string(res))
	assert.Equal(t, `{"f":{}}`, string(srv.GetLatestJson().CachedJson))
	assert.Equal(t, `{"f":{}}`, string(srv.LoadEntry().CachedJson))
}

func TestRedisNotify_Close(t *testing.T) {
	s := miniredis.RunT(t)
	srv := NewNotifyingRedisStorage("key", config.SDKConfig{Cache: config.CacheConfig{
		Redis: config.RedisConfig{Addresses: []string{s.Addr()}},
	}, Offline: config.OfflineConfig{CachePollInterval: 1}}, log.NewNullLogger()).(*notifyingRedisStorage)
	go func() {
		srv.Close()
	}()
	utils.WithTimeout(2*time.Second, func() {
		select {
		case <-srv.closed:
		case <-srv.Modified():
		}
	})
}
