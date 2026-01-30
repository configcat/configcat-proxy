package store

import (
	"context"
	"fmt"

	"github.com/configcat/configcat-proxy/diag/status"
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

type cacheStore struct {
	EntryStore

	reporter    status.Reporter
	actualCache configcat.ConfigCache
}

func NewCacheStore(actualCache configcat.ConfigCache, reporter status.Reporter) CacheEntryStore {
	return &cacheStore{
		EntryStore:  NewEntryStore(),
		reporter:    reporter,
		actualCache: actualCache,
	}
}

func (c *cacheStore) Get(ctx context.Context, key string) ([]byte, error) {
	b, err := c.actualCache.Get(ctx, key)
	if err != nil {
		c.reporter.ReportError(status.Cache, "cache read failed")
	} else {
		c.reporter.ReportOk(status.Cache, "cache read succeeded")
	}
	return b, err
}

func (c *cacheStore) Set(ctx context.Context, key string, value []byte) error {
	fetchTime, etag, configJson, err := configcatcache.CacheSegmentsFromBytes(value)
	if err != nil {
		c.reporter.ReportError(status.Cache, "cache write failed")
		return err
	}
	c.StoreEntry(configJson, fetchTime, etag)
	err = c.actualCache.Set(ctx, key, value)
	if err != nil {
		c.reporter.ReportError(status.Cache, "cache write failed")
		return err
	}
	c.reporter.ReportOk(status.Cache, "cache write succeeded")
	return nil
}

type inMemoryStore struct {
	EntryStore
}

func NewInMemoryStorage() configcat.ConfigCache {
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
