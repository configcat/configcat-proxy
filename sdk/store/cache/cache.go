package cache

import (
	"context"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/sdk/store"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
)

type cacheStore struct {
	store.EntryStore

	reporter    status.Reporter
	actualCache configcat.ConfigCache
}

func NewCacheStore(actualCache configcat.ConfigCache, reporter status.Reporter) store.CacheEntryStore {
	return &cacheStore{
		EntryStore:  store.NewEntryStore(),
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

func (c *cacheStore) Close() {
	if closable, ok := c.actualCache.(store.ClosableStore); ok {
		closable.Close()
	}
}
