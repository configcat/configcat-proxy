package redis

import (
	"context"
	"crypto/tls"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/configcat-proxy/status"
	"github.com/configcat/go-sdk/v8/configcatcache"
	"github.com/redis/go-redis/v9"
)

type redisStorage struct {
	store.EntryStore

	redisDb  redis.UniversalClient
	reporter status.Reporter
}

func NewRedisStorage(conf *config.RedisConfig, reporter status.Reporter) store.CacheStorage {
	opts := &redis.UniversalOptions{
		Addrs:    conf.Addresses,
		Password: conf.Password,
		DB:       conf.DB,
	}
	if conf.User != "" {
		opts.Username = conf.User
	}
	if conf.Tls.Enabled {
		t := &tls.Config{
			MinVersion: conf.Tls.GetVersion(),
			ServerName: conf.Tls.ServerName,
		}
		for _, c := range conf.Tls.Certificates {
			if cert, err := tls.LoadX509KeyPair(c.Cert, c.Key); err == nil {
				t.Certificates = append(t.Certificates, cert)
			}
		}
		opts.TLSConfig = t
	}
	return &redisStorage{
		redisDb:    redis.NewUniversalClient(opts),
		EntryStore: store.NewEntryStore(),
		reporter:   reporter,
	}
}

func (r *redisStorage) Get(ctx context.Context, key string) ([]byte, error) {
	b, err := r.redisDb.Get(ctx, key).Bytes()
	if err != nil {
		r.reporter.ReportError(status.Cache, err)
	} else {
		r.reporter.ReportOk(status.Cache, "cache read succeeded")
	}
	return b, err
}

func (r *redisStorage) Set(ctx context.Context, key string, value []byte) error {
	fetchTime, etag, configJson, err := configcatcache.CacheSegmentsFromBytes(value)
	if err != nil {
		r.reporter.ReportError(status.Cache, err)
	}
	r.StoreEntry(configJson, fetchTime, etag)
	err = r.redisDb.Set(ctx, key, value, 0).Err()
	if err != nil {
		r.reporter.ReportError(status.Cache, err)
	} else {
		r.reporter.ReportOk(status.Cache, "cache write succeeded")
	}
	return err
}

func (r *redisStorage) Close() {
	_ = r.redisDb.Close()
}
