package cache

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCacheStore(t *testing.T) {
	store := NewCacheStore(&testCache{}, status.NewEmptyReporter()).(*cacheStore)

	err := store.Set(context.Background(), "key", configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`)))
	assert.NoError(t, err)
	res, err := store.Get(context.Background(), "key")
	assert.NoError(t, err)
	_, _, j, err := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `test`, string(j))
	assert.Equal(t, `test`, string(store.LoadEntry().ConfigJson))
}

func TestSetupExternalCache(t *testing.T) {
	t.Run("redis", func(t *testing.T) {
		s := miniredis.RunT(t)
		store, err := SetupExternalCache(context.Background(), &config.CacheConfig{Redis: config.RedisConfig{Addresses: []string{s.Addr()}, Enabled: true}}, log.NewNullLogger())
		defer store.Shutdown()
		assert.NoError(t, err)
		assert.IsType(t, &redisStore{}, store)
	})
	t.Run("mongodb", func(t *testing.T) {
		store, err := SetupExternalCache(context.Background(), &config.CacheConfig{MongoDb: config.MongoDbConfig{
			Enabled:    true,
			Url:        "mongodb://localhost:27017",
			Database:   "test_db",
			Collection: "coll",
		}}, log.NewNullLogger())
		defer store.Shutdown()
		assert.NoError(t, err)
		assert.IsType(t, &mongoDbStore{}, store)
	})
	t.Run("dynamodb", func(t *testing.T) {
		store, err := SetupExternalCache(context.Background(), &config.CacheConfig{DynamoDb: config.DynamoDbConfig{
			Enabled: true,
			Table:   tableName,
			Url:     endpoint,
		}}, log.NewNullLogger())
		defer store.Shutdown()
		assert.NoError(t, err)
		assert.IsType(t, &dynamoDbStore{}, store)
	})
	t.Run("only one selected", func(t *testing.T) {
		s := miniredis.RunT(t)
		store, err := SetupExternalCache(context.Background(), &config.CacheConfig{
			Redis: config.RedisConfig{Addresses: []string{s.Addr()}, Enabled: true},
			MongoDb: config.MongoDbConfig{
				Enabled:    true,
				Url:        "mongodb://localhost:27017",
				Database:   "test_db",
				Collection: "coll",
			},
		}, log.NewNullLogger())
		defer store.Shutdown()
		assert.NoError(t, err)
		assert.IsType(t, &redisStore{}, store)
	})
}

type testCache struct {
	v []byte
}

func (r *testCache) Get(_ context.Context, _ string) ([]byte, error) {
	return r.v, nil
}

func (r *testCache) Set(_ context.Context, _ string, val []byte) error {
	r.v = val
	return nil
}

func (r *testCache) Close() {}
