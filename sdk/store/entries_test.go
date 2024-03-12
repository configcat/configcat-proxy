package store

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestEntryStore(t *testing.T) {
	t.Run("load default", func(t *testing.T) {
		e := NewEntryStore()
		assert.NotNil(t, e.LoadEntry().ConfigJson)
		assert.True(t, e.LoadEntry().Empty)
	})
	t.Run("store, check etag", func(t *testing.T) {
		e := NewEntryStore()
		data := []byte("test")
		e.StoreEntry(data, time.Time{}, "etag")
		assert.Equal(t, data, e.LoadEntry().ConfigJson)
		assert.Equal(t, "etag", e.LoadEntry().ETag)
		assert.Equal(t, "-62135596800000\netag\ntest", string(e.ComposeBytes()))
		assert.False(t, e.LoadEntry().Empty)
	})
}
