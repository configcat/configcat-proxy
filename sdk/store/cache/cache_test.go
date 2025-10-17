package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
)

func TestCacheStore(t *testing.T) {
	store := NewCacheStore(&testCache{}, status.NewEmptyReporter()).(*cacheStore)

	err := store.Set(t.Context(), "key", configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`)))
	assert.NoError(t, err)
	res, err := store.Get(t.Context(), "key")
	assert.NoError(t, err)
	_, _, j, err := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `test`, string(j))
	assert.Equal(t, `test`, string(store.LoadEntry().ConfigJson))
}

func TestSetupExternalCache_OnlyOneSelected(t *testing.T) {
	s := miniredis.RunT(t)
	store, err := SetupExternalCache(t.Context(), &config.CacheConfig{
		Redis: config.RedisConfig{Addresses: []string{s.Addr()}, Enabled: true},
		MongoDb: config.MongoDbConfig{
			Enabled:    true,
			Url:        "mongodb://localhost:27017",
			Database:   "test_db",
			Collection: "coll",
		},
	}, telemetry.NewEmptyReporter(), log.NewNullLogger())
	assert.NoError(t, err)
	defer store.Shutdown()
	assert.IsType(t, &redisStore{}, store)
}

func (s *mongoTestSuite) TestSetupExternalCache() {
	store, err := SetupExternalCache(s.T().Context(), &config.CacheConfig{MongoDb: config.MongoDbConfig{
		Enabled:    true,
		Url:        s.addr,
		Database:   "test_db",
		Collection: "coll",
	}}, telemetry.NewEmptyReporter(), log.NewNullLogger())
	assert.NoError(s.T(), err)
	defer store.Shutdown()
	assert.IsType(s.T(), &mongoDbStore{}, store)
}

func (s *redisTestSuite) TestSetupExternalCache() {
	store, err := SetupExternalCache(s.T().Context(), &config.CacheConfig{Redis: config.RedisConfig{Addresses: []string{"localhost:" + s.dbPort}, Enabled: true}}, telemetry.NewEmptyReporter(), log.NewNullLogger())
	assert.NoError(s.T(), err)
	defer store.Shutdown()
	assert.IsType(s.T(), &redisStore{}, store)
}

func (s *valkeyTestSuite) TestSetupExternalCache() {
	store, err := SetupExternalCache(s.T().Context(), &config.CacheConfig{Redis: config.RedisConfig{Addresses: []string{"localhost:" + s.dbPort}, Enabled: true}}, telemetry.NewEmptyReporter(), log.NewNullLogger())
	assert.NoError(s.T(), err)
	defer store.Shutdown()
	assert.IsType(s.T(), &redisStore{}, store)
}

func (s *dynamoDbTestSuite) TestSetupExternalCache() {
	store, err := SetupExternalCache(s.T().Context(), &config.CacheConfig{DynamoDb: config.DynamoDbConfig{
		Enabled: true,
		Table:   tableName,
		Url:     s.addr,
	}}, telemetry.NewEmptyReporter(), log.NewNullLogger())
	assert.NoError(s.T(), err)
	defer store.Shutdown()
	assert.IsType(s.T(), &dynamoDbStore{}, store)
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
