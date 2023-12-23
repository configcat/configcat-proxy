package store

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
)

type CacheEntryStore interface {
	EntryStore
	configcat.ConfigCache

	CacheKey() string
}

type ClosableStore interface {
	Close()
}

type NotifyingStore interface {
	EntryStore
	Notifier
	configcat.ConfigCache
}

type inMemoryStore struct {
	EntryStore
}

func NewInMemoryStorage(version config.SDKVersion) configcat.ConfigCache {
	return &inMemoryStore{EntryStore: NewEntryStore(version)}
}

func (r *inMemoryStore) Get(_ context.Context, _ string) ([]byte, error) {
	return r.ComposeBytes(config.V6), nil
}

func (r *inMemoryStore) Set(_ context.Context, _ string, value []byte) error {
	fetchTime, etag, configJson, _ := configcatcache.CacheSegmentsFromBytes(value)
	r.StoreEntry(configJson, fetchTime, etag)
	return nil
}
