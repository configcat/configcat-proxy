package store

import (
	"context"
)

type Storage interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error

	GetLatestJson() *EntryWithEtag
	Modified() <-chan struct{}

	Close()
}

type CacheStorage interface {
	EntryStore
	Storage
}

type InMemoryStorage struct {
	EntryStore
}

func (r *InMemoryStorage) Get(_ context.Context, _ string) ([]byte, error) {
	return r.LoadEntry().CachedJson, nil
}

func (r *InMemoryStorage) Set(_ context.Context, _ string, value []byte) error {
	r.StoreEntry(value)
	return nil
}

func (r *InMemoryStorage) Close() {
	// do nothing
}
