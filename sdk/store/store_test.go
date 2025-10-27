package store

import (
	"context"
	"testing"
	"time"

	"github.com/configcat/configcat-proxy/diag/status"
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

func TestInMemoryStore(t *testing.T) {
	t.Run("load default", func(t *testing.T) {
		e := NewInMemoryStorage().(*inMemoryStore)
		_, err := e.Get(t.Context(), "")
		assert.Error(t, err)
	})
	t.Run("store, check etag", func(t *testing.T) {
		e := NewInMemoryStorage().(*inMemoryStore)
		data := []byte("test")
		c := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", data)
		err := e.Set(t.Context(), "", c)
		assert.NoError(t, err)
		r, err := e.Get(t.Context(), "")
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(r)
		assert.NoError(t, err)
		assert.Equal(t, data, j)
		assert.Equal(t, "etag", e.LoadEntry().ETag)
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
