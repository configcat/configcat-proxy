package cache

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRedisStorage(t *testing.T) {
	s := miniredis.RunT(t)
	store, err := newRedis(&config.RedisConfig{Addresses: []string{s.Addr()}}, log.NewNullLogger())
	assert.NoError(t, err)
	srv := store.(*redisStore)
	defer srv.Shutdown()

	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`))
	err = srv.Set(context.Background(), "key", cacheEntry)
	assert.NoError(t, err)
	s.CheckGet(t, "key", string(cacheEntry))
	res, err := srv.Get(context.Background(), "key")
	assert.NoError(t, err)
	_, _, j, err := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `test`, string(j))
}

func TestRedisStorage_Unavailable(t *testing.T) {
	store, err := newRedis(&config.RedisConfig{Addresses: []string{"nonexisting"}}, log.NewNullLogger())
	assert.NoError(t, err)
	srv := store.(*redisStore)
	defer srv.Shutdown()

	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`))
	err = srv.Set(context.Background(), "", cacheEntry)
	assert.Error(t, err)
	_, err = srv.Get(context.Background(), "")
	assert.Error(t, err)
}
