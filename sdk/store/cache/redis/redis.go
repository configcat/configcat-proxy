package redis

import (
	"context"
	"crypto/tls"
	"github.com/configcat/configcat-proxy/config"
	configcat "github.com/configcat/go-sdk/v9"
	"github.com/redis/go-redis/v9"
)

type redisStore struct {
	redisDb redis.UniversalClient
}

func NewRedisStore(conf *config.RedisConfig) configcat.ConfigCache {
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
	return &redisStore{
		redisDb: redis.NewUniversalClient(opts),
	}
}

func (r *redisStore) Get(ctx context.Context, key string) ([]byte, error) {
	return r.redisDb.Get(ctx, key).Bytes()
}

func (r *redisStore) Set(ctx context.Context, key string, value []byte) error {
	return r.redisDb.Set(ctx, key, value, 0).Err()
}

func (r *redisStore) Close() {
	_ = r.redisDb.Close()
}
