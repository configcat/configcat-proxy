package cache

import (
	"context"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"github.com/redis/go-redis/v9"
)

type redisStore struct {
	redisDb redis.UniversalClient
	log     log.Logger
}

func newRedis(conf *config.RedisConfig, telemetryReporter telemetry.Reporter, log log.Logger) (External, error) {
	opts := &redis.UniversalOptions{
		Addrs:    conf.Addresses,
		Password: conf.Password,
		DB:       conf.DB,
	}
	if conf.User != "" {
		opts.Username = conf.User
	}
	if conf.Tls.Enabled {
		t, err := conf.Tls.LoadTlsOptions()
		if err != nil {
			log.Errorf("failed to configure TLS for Redis: %s", err)
			return nil, err
		}
		opts.TLSConfig = t
	}
	rdb := redis.NewUniversalClient(opts)
	telemetryReporter.InstrumentRedis(rdb)
	log.Reportf("using Redis for cache storage")
	return &redisStore{
		redisDb: rdb,
		log:     log,
	}, nil
}

func (r *redisStore) Get(ctx context.Context, key string) ([]byte, error) {
	return r.redisDb.Get(ctx, key).Bytes()
}

func (r *redisStore) Set(ctx context.Context, key string, value []byte) error {
	return r.redisDb.Set(ctx, key, value, 0).Err()
}

func (r *redisStore) Shutdown() {
	err := r.redisDb.Close()
	if err != nil {
		r.log.Errorf("shutdown error: %s", err)
	}
	r.log.Reportf("shutdown complete")
}
