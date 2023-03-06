package store

import (
	"context"
)

type CacheStorage interface {
	EntryStore

	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
	Close()
}

type NotifyingStorage interface {
	CacheStorage
	Notifier
}

type inMemoryStorage struct {
	EntryStore
}

func NewInMemoryStorage() CacheStorage {
	return &inMemoryStorage{EntryStore: NewEntryStore()}
}

func (r *inMemoryStorage) Get(_ context.Context, _ string) ([]byte, error) {
	return r.LoadEntry().CachedJson, nil
}

func (r *inMemoryStorage) Set(_ context.Context, _ string, value []byte) error {
	r.StoreEntry(value)
	return nil
}

func (r *inMemoryStorage) Close() {
	// do nothing
}
