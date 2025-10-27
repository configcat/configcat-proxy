package cache

import (
	"context"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	configcat "github.com/configcat/go-sdk/v9"
)

const (
	keyName     = "key"
	payloadName = "payload"
)

type ReaderWriter = configcat.ConfigCache

type External interface {
	ReaderWriter
	Shutdown()
}

func SetupExternalCache(ctx context.Context, conf *config.CacheConfig, telemetryReporter telemetry.Reporter, log log.Logger) (External, error) {
	cacheLog := log.WithPrefix("cache")
	if conf.Redis.Enabled {
		redis, err := newRedis(&conf.Redis, telemetryReporter, cacheLog)
		if err != nil {
			return nil, err
		}
		return redis, nil
	} else if conf.MongoDb.Enabled {
		mongoDb, err := newMongoDb(ctx, &conf.MongoDb, telemetryReporter, cacheLog)
		if err != nil {
			return nil, err
		}
		return mongoDb, nil
	} else if conf.DynamoDb.Enabled {
		dynamoDb, err := newDynamoDb(ctx, &conf.DynamoDb, telemetryReporter, cacheLog)
		if err != nil {
			return nil, err
		}
		return dynamoDb, nil
	}
	return nil, nil
}
