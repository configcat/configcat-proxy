package store

import (
	"context"
	"crypto/sha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInMemoryStore(t *testing.T) {
	t.Run("load default", func(t *testing.T) {
		e := InMemoryStorage{EntryStore: NewEntryStore()}
		r, err := e.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.NotNil(t, r)
		assert.Equal(t, r, e.GetLatestJson().CachedJson)
	})
	t.Run("store, check etag", func(t *testing.T) {
		e := InMemoryStorage{EntryStore: NewEntryStore()}
		data := []byte("test")
		etag := sha1.Sum(data)
		err := e.Set(context.Background(), "", data)
		assert.NoError(t, err)
		r, err := e.Get(context.Background(), "")
		assert.NoError(t, err)
		assert.Equal(t, data, r)
		assert.NotNil(t, etag, e.LoadEntry().Etag)
	})
	t.Run("modified channel", func(t *testing.T) {
		e := InMemoryStorage{EntryStore: NewEntryStore()}
		go func() {
			e.Notify()
		}()
		<-e.Modified()
	})
}
