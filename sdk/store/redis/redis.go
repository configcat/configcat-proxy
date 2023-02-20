package redis

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/sdk/store"
	"github.com/redis/go-redis/v9"
)

type redisStorage struct {
	redisDb  redis.UniversalClient
	cacheKey string
	*store.EntryStore
}

func NewRedisStorage(sdkKey string, conf config.RedisConfig) store.Storage {
	r := newRedisStorage(sdkKey, conf)
	return &r
}

func newRedisStorage(sdkKey string, conf config.RedisConfig) redisStorage {
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
		EntryStore: store.NewEntryStore()}
}

func (r *redisStorage) Get(ctx context.Context, _ string) ([]byte, error) {
	return r.redisDb.Get(ctx, r.cacheKey).Bytes()
}

func (r *redisStorage) Set(ctx context.Context, _ string, value []byte) error {
	r.StoreEntry(value)
	return r.redisDb.Set(ctx, r.cacheKey, value, 0).Err()
}

func (r *redisStorage) Close() {
	_ = r.redisDb.Close()
}
