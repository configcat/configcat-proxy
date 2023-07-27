package redis

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v8/configcatcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRedisStorage(t *testing.T) {
	s := miniredis.RunT(t)
	srv := NewRedisStorage(&config.RedisConfig{Addresses: []string{s.Addr()}}, status.NewNullReporter()).(*redisStorage)

	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`))
	err := srv.Set(context.Background(), "key", cacheEntry)
	assert.NoError(t, err)
	s.CheckGet(t, "key", string(cacheEntry))
	res, err := srv.Get(context.Background(), "key")
	assert.NoError(t, err)
	_, _, j, err := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(j))
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.LoadEntry().ConfigJson))
}

func TestRedisStorage_Unavailable(t *testing.T) {
	srv := NewRedisStorage(&config.RedisConfig{Addresses: []string{"nonexisting"}}, status.NewNullReporter()).(*redisStorage)

	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`))
	err := srv.Set(context.Background(), "", cacheEntry)
	assert.Error(t, err)
	_, err = srv.Get(context.Background(), "")
	assert.Error(t, err)
	assert.Equal(t, `{"f":{"flag":{"i":"","v":false,"t":0,"r":[],"p":[]}}}`, string(srv.LoadEntry().ConfigJson))
}
