package store

import (
	"github.com/configcat/configcat-proxy/internal/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestEntryStore(t *testing.T) {
	t.Run("load default", func(t *testing.T) {
		e := NewEntryStore()
		assert.NotNil(t, e.LoadEntry().ConfigJson)
	})
	t.Run("store, check etag", func(t *testing.T) {
		e := NewEntryStore()
		data := []byte("test")
		etag := "W/" + "\"" + utils.FastHashHex(data) + "\""
		e.StoreEntry(data, time.Time{}, "etag")
		assert.Equal(t, data, e.LoadEntry().ConfigJson)
		assert.Equal(t, "etag", e.LoadEntry().CachedETag)
		assert.Equal(t, etag, e.LoadEntry().GeneratedETag)
		assert.Equal(t, "-62135596800000\netag\ntest", string(e.ComposeBytes()))
	})
}
