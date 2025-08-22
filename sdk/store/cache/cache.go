package cache

import (
	"context"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/status"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/go-sdk/v9/configcatcache"
)

const (
	keyName     = "key"
	payloadName = "payload"
)

type External interface {
	store.Cache
	Shutdown()
}

type cacheStore struct {
	store.EntryStore

	reporter    status.Reporter
	actualCache store.Cache
}

func SetupExternalCache(ctx context.Context, conf *config.CacheConfig, log log.Logger) (External, error) {
	cacheLog := log.WithPrefix("cache")
	if conf.Redis.Enabled {
		redis, err := newRedis(&conf.Redis, cacheLog)
		if err != nil {
			return nil, err
		}
		return redis, nil
	} else if conf.MongoDb.Enabled {
		mongoDb, err := newMongoDb(ctx, &conf.MongoDb, cacheLog)
		if err != nil {
			return nil, err
		}
		return mongoDb, nil
	} else if conf.DynamoDb.Enabled {
		dynamoDb, err := newDynamoDb(ctx, &conf.DynamoDb, cacheLog)
		if err != nil {
			return nil, err
		}
		return dynamoDb, nil
	}
	return nil, nil
}

func NewCacheStore(actualCache store.Cache, reporter status.Reporter) store.CacheEntryStore {
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
