package cache

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestCacheStore(t *testing.T) {
	store := NewCacheStore(&testCache{}, status.NewNullReporter(), "key", config.V6).(*cacheStore)

	err := store.Set(context.Background(), store.v6Key, configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`)))
	assert.NoError(t, err)
	res, err := store.Get(context.Background(), store.v6Key)
	assert.NoError(t, err)
	_, _, j, err := configcatcache.CacheSegmentsFromBytes(res)
	assert.NoError(t, err)
	assert.Equal(t, `test`, string(j))
	assert.Equal(t, `test`, string(store.LoadEntry(config.V6).ConfigJson))
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
