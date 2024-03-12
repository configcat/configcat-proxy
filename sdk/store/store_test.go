package store

import (
	"context"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestInMemoryStore(t *testing.T) {
	t.Run("load default", func(t *testing.T) {
		e := NewInMemoryStorage().(*inMemoryStore)
		_, err := e.Get(context.Background(), "")
		assert.Error(t, err)
	})
	t.Run("store, check etag", func(t *testing.T) {
		e := NewInMemoryStorage().(*inMemoryStore)
		data := []byte("test")
		c := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", data)
		err := e.Set(context.Background(), "", c)
		assert.NoError(t, err)
		r, err := e.Get(context.Background(), "")
		_, _, j, _ := configcatcache.CacheSegmentsFromBytes(r)
		assert.NoError(t, err)
		assert.Equal(t, data, j)
		assert.Equal(t, "etag", e.LoadEntry().ETag)
	})
}
