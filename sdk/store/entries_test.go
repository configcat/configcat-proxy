package store

import (
	"crypto/sha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEntryStore(t *testing.T) {
	t.Run("load default", func(t *testing.T) {
		e := NewEntryStore()
		assert.NotNil(t, e.LoadEntry().CachedJson)
		assert.Equal(t, e.GetLatestJson(), e.LoadEntry())
	})
	t.Run("store, check etag", func(t *testing.T) {
		e := NewEntryStore()
		data := []byte("test")
		etag := sha1.Sum(data)
		e.StoreEntry(data)
		assert.Equal(t, data, e.LoadEntry().CachedJson)
		assert.NotNil(t, etag, e.LoadEntry().Etag)
	})
}
