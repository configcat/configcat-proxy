package cache

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/configcat-proxy/status"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/configcat/go-sdk/v9/configcatcache"
)

const configJSONNameV5 = "config_v5.json"

type cacheStore struct {
	store.EntryStore

	reporter    status.Reporter
	actualCache configcat.ConfigCache
	v5Key       string
	v6Key       string
	sdkVersion  config.SDKVersion
}

func NewCacheStore(actualCache configcat.ConfigCache, reporter status.Reporter, sdkKey string, sdkVersion config.SDKVersion) store.CacheEntryStore {
	return &cacheStore{
		EntryStore:  store.NewEntryStore(sdkVersion),
		reporter:    reporter,
		actualCache: actualCache,
		v5Key:       configcatcache.ProduceCacheKey(sdkKey, configJSONNameV5, configcatcache.ConfigJSONCacheVersion),
		v6Key:       configcatcache.ProduceCacheKey(sdkKey, configcatcache.ConfigJSONName, configcatcache.ConfigJSONCacheVersion),
		sdkVersion:  sdkVersion,
	}
}

func (c *cacheStore) Get(ctx context.Context, key string) ([]byte, error) {
	b, err := c.actualCache.Get(ctx, key)
	if err != nil {
		c.reporter.ReportError(status.Cache, err)
	} else {
		c.reporter.ReportOk(status.Cache, "cache read succeeded")
	}
	return b, err
}

func (c *cacheStore) Set(ctx context.Context, key string, value []byte) error {
	fetchTime, etag, configJson, err := configcatcache.CacheSegmentsFromBytes(value)
	if err != nil {
		c.reporter.ReportError(status.Cache, err)
	}
	c.StoreEntry(configJson, fetchTime, etag)
	err = c.actualCache.Set(ctx, key, value)
	if err != nil {
		c.reporter.ReportError(status.Cache, err)
	}
	if c.sdkVersion == config.V5 {
		err = c.actualCache.Set(ctx, c.v5Key, c.ComposeBytes(config.V5))
		if err != nil {
			c.reporter.ReportError(status.Cache, err)
		}
	}
	if err == nil {
		c.reporter.ReportOk(status.Cache, "cache write succeeded")
	}
	return err
}

func (c *cacheStore) CacheKey() string {
	return c.v6Key
}

func (c *cacheStore) Close() {
	if closable, ok := c.actualCache.(store.ClosableStore); ok {
		closable.Close()
	}
}
