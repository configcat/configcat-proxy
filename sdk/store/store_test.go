package store

import (
	"testing"
	"time"

	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
)

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
