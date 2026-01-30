package cache

import (
	"context"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoDbStore struct {
	mongoDb    *mongo.Client
	collection *mongo.Collection
	log        log.Logger
}

type entry struct {
	Key     string
	Payload []byte
}

func newMongoDb(ctx context.Context, conf *config.MongoDbConfig, telemetryReporter telemetry.Reporter, log log.Logger) (External, error) {
	opts := options.Client().ApplyURI(conf.Url)
	telemetryReporter.InstrumentMongoDb(opts)
	if conf.Tls.Enabled {
		t, err := conf.Tls.LoadTlsOptions()
		if err != nil {
			log.Errorf("failed to configure TLS for MongoDB: %s", err)
			return nil, err
		}
		opts.SetTLSConfig(t)
	}
	client, err := mongo.Connect(opts)
	if err != nil {
		log.Errorf("couldn't connect to MongoDB: %s", err)
		return nil, err
	}
	collection := client.Database(conf.Database).Collection(conf.Collection)
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{keyName: 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Errorf("couldn't create the 'key' index in the '%s' MongoDB collection: %s", conf.Collection, err)
		return nil, err
	}
	log.Reportf("using MongoDB for cache storage")
	return &mongoDbStore{
		mongoDb:    client,
		collection: collection,
		log:        log,
	}, nil
}

func (m *mongoDbStore) Get(ctx context.Context, key string) ([]byte, error) {
	var result entry
	err := m.collection.FindOne(ctx, bson.M{keyName: key}).Decode(&result)
	return result.Payload, err
}

func (m *mongoDbStore) Set(ctx context.Context, key string, value []byte) error {
	_, err := m.collection.ReplaceOne(ctx, bson.M{keyName: key}, entry{Key: key, Payload: value}, options.Replace().SetUpsert(true))
	return err
}

func (m *mongoDbStore) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.mongoDb.Disconnect(ctx)
	if err != nil {
		m.log.Errorf("shutdown error: %s", err)
	}
	m.log.Reportf("shutdown complete")
}
