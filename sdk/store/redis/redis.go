package redis

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/configcat/configcat-proxy/status"
	"github.com/redis/go-redis/v9"
)

type redisStorage struct {
	redisDb  redis.UniversalClient
	cacheKey string
	reporter status.Reporter
	*store.EntryStore
}

func NewRedisStorage(sdkKey string, conf config.RedisConfig, reporter status.Reporter) store.Storage {
	r := newRedisStorage(sdkKey, conf, reporter)
	return &r
}

func newRedisStorage(sdkKey string, conf config.RedisConfig, reporter status.Reporter) redisStorage {
	opts := &redis.UniversalOptions{
		Addrs:    conf.Addresses,
		Password: conf.Password,
		DB:       conf.DB,
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
	return redisStorage{
		redisDb:    redis.NewUniversalClient(opts),
		cacheKey:   fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("%s_%s", sdkKey, "config_v5")))),
		EntryStore: store.NewEntryStore(),
		reporter:   reporter,
	}
}

func (r *redisStorage) Get(ctx context.Context, _ string) ([]byte, error) {
	b, err := r.redisDb.Get(ctx, r.cacheKey).Bytes()
	if err != nil {
		r.reporter.ReportError(status.Cache, err)
	} else {
		r.reporter.ReportOk(status.Cache, "cache read succeeded")
	}
	return b, err
}

func (r *redisStorage) Set(ctx context.Context, _ string, value []byte) error {
	r.StoreEntry(value)
	err := r.redisDb.Set(ctx, r.cacheKey, value, 0).Err()
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
