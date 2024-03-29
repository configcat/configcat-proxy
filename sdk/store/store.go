package store

import (
	"context"
	"fmt"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
)

type CacheEntryStore interface {
	EntryStore
	configcat.ConfigCache
}

type NotifyingStore interface {
	CacheEntryStore
	Notifier
}

type inMemoryStore struct {
	EntryStore
}

func NewInMemoryStorage() CacheEntryStore {
	return &inMemoryStore{EntryStore: NewEntryStore()}
}

func (r *inMemoryStore) Get(_ context.Context, _ string) ([]byte, error) {
	if r.LoadEntry().Empty {
		return nil, fmt.Errorf("no entry in cache")
	}
	return r.ComposeBytes(), nil
}

func (r *inMemoryStore) Set(_ context.Context, _ string, value []byte) error {
	fetchTime, etag, configJson, _ := configcatcache.CacheSegmentsFromBytes(value)
	r.StoreEntry(configJson, fetchTime, etag)
	return nil
}
