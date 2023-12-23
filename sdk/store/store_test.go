package store

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestInMemoryStore(t *testing.T) {
	t.Run("load default", func(t *testing.T) {
		e := NewInMemoryStorage(config.V6).(*inMemoryStore)
		r, err := e.Get(context.Background(), "")
		assert.NoError(t, err)
		_, _, j, err := configcatcache.CacheSegmentsFromBytes(r)
		assert.NotNil(t, r)
		assert.Equal(t, j, e.LoadEntry(config.V6).ConfigJson)
	})
	t.Run("store, check etag", func(t *testing.T) {
		e := NewInMemoryStorage(config.V6).(*inMemoryStore)
		data := []byte("test")
		c := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", data)
		err := e.Set(context.Background(), "", c)
		assert.NoError(t, err)
		r, err := e.Get(context.Background(), "")
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(r)
		assert.NoError(t, err)
		assert.Equal(t, data, j)
		assert.Equal(t, "etag", e.LoadEntry(config.V6).ETag)
	})
}
