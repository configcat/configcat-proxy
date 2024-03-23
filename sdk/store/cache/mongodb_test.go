package cache

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMongoDbStore(t *testing.T) {
	store, err := newMongoDb(context.Background(), &config.MongoDbConfig{
		Enabled:    true,
		Url:        "mongodb://localhost:27017",
		Database:   "test_db",
		Collection: "coll",
	}, log.NewNullLogger())
	defer store.Shutdown()

	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`))

	err = store.Set(context.Background(), "k1", cacheEntry)
	assert.NoError(t, err)

	res, err := store.Get(context.Background(), "k1")
	assert.NoError(t, err)
	assert.Equal(t, cacheEntry, res)

	cacheEntry = configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test2`))

	err = store.Set(context.Background(), "k1", cacheEntry)
	assert.NoError(t, err)

	res, err = store.Get(context.Background(), "k1")
	assert.NoError(t, err)
	assert.Equal(t, cacheEntry, res)
}

func TestMongoDbStore_Empty(t *testing.T) {
	store, err := newMongoDb(context.Background(), &config.MongoDbConfig{
		Enabled:    true,
		Url:        "mongodb://localhost:27017",
		Database:   "test_db",
		Collection: "coll",
	}, log.NewNullLogger())
	defer store.Shutdown()

	_, err = store.Get(context.Background(), "k2")
	assert.Error(t, err)
}

func TestMongoDbStore_Invalid(t *testing.T) {
	_, err := newMongoDb(context.Background(), &config.MongoDbConfig{
		Enabled:    true,
		Url:        "invalid",
		Database:   "test_db",
		Collection: "coll",
	}, log.NewNullLogger())

	assert.Error(t, err)
}
