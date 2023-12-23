package store

import (
	"github.com/configcat/configcat-proxy/config"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestEntryStore(t *testing.T) {
	t.Run("load default", func(t *testing.T) {
		e := NewEntryStore(config.V6)
		assert.NotNil(t, e.LoadEntry(config.V6).ConfigJson)
	})
	t.Run("store, check etag", func(t *testing.T) {
		e := NewEntryStore(config.V6)
		data := []byte("test")
		e.StoreEntry(data, time.Time{}, "etag")
		assert.Equal(t, data, e.LoadEntry(config.V6).ConfigJson)
		assert.Equal(t, "etag", e.LoadEntry(config.V6).ETag)
		assert.Equal(t, "-62135596800000\netag\ntest", string(e.ComposeBytes(config.V6)))
	})
}
