package cache

import (
	"context"
	"testing"
	"time"

	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

type mongoTestSuite struct {
	suite.Suite

	db   *mongodb.MongoDBContainer
	addr string
}

func (s *mongoTestSuite) SetupSuite() {
	mongodbContainer, err := mongodb.Run(s.T().Context(), "mongo")
	if err != nil {
		panic("failed to start container: " + err.Error() + "")
	}
	s.db = mongodbContainer
	str, _ := s.db.ConnectionString(s.T().Context())
	s.addr = str
}

func (s *mongoTestSuite) TearDownSuite() {
	if err := testcontainers.TerminateContainer(s.db); err != nil {
		panic("failed to terminate container: " + err.Error() + "")
	}
}

func TestRunMongoSuite(t *testing.T) {
	suite.Run(t, new(mongoTestSuite))
}

func (s *mongoTestSuite) TestMongoDbStore() {
	store, err := newMongoDb(context.Background(), &config.MongoDbConfig{
		Enabled:    true,
		Url:        s.addr,
		Database:   "test_db",
		Collection: "coll",
	}, log.NewNullLogger())
	assert.NoError(s.T(), err)
	defer store.Shutdown()

	cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`))

	err = store.Set(context.Background(), "k1", cacheEntry)
	assert.NoError(s.T(), err)

	res, err := store.Get(context.Background(), "k1")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), cacheEntry, res)

	cacheEntry = configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test2`))

	err = store.Set(context.Background(), "k1", cacheEntry)
	assert.NoError(s.T(), err)

	res, err = store.Get(context.Background(), "k1")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), cacheEntry, res)
}

func (s *mongoTestSuite) TestMongoDbStore_Empty() {
	store, err := newMongoDb(context.Background(), &config.MongoDbConfig{
		Enabled:    true,
		Url:        s.addr,
		Database:   "test_db",
		Collection: "coll",
	}, log.NewNullLogger())
	assert.NoError(s.T(), err)
	defer store.Shutdown()

	_, err = store.Get(context.Background(), "k2")
	assert.Error(s.T(), err)
}

func (s *mongoTestSuite) TestMongoDbStore_Invalid() {
	_, err := newMongoDb(context.Background(), &config.MongoDbConfig{
		Enabled:    true,
		Url:        "invalid",
		Database:   "test_db",
		Collection: "coll",
	}, log.NewNullLogger())

	assert.Error(s.T(), err)
}

func (s *mongoTestSuite) TestMongoDbStore_TLS_Invalid() {
	store, err := newMongoDb(context.Background(), &config.MongoDbConfig{
		Enabled:    true,
		Url:        s.addr,
		Database:   "test_db",
		Collection: "coll",
		Tls: config.TlsConfig{
			Enabled:    true,
			MinVersion: 1.1,
			Certificates: []config.CertConfig{
				{Key: "nonexisting", Cert: "nonexisting"},
			},
		},
	}, log.NewNullLogger())
	assert.ErrorContains(s.T(), err, "failed to load certificate and key files")
	assert.Nil(s.T(), store)
}

func (s *mongoTestSuite) TestMongoDbStore_Connect_Fails() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	store, err := newMongoDb(ctx, &config.MongoDbConfig{
		Enabled:    true,
		Url:        "mongodb://localhost:12345",
		Database:   "test_db",
		Collection: "coll",
	}, log.NewNullLogger())
	assert.ErrorContains(s.T(), err, "context deadline exceeded")
	assert.Nil(s.T(), store)
}

func (s *mongoTestSuite) TestSetupExternalCache() {
	store, err := SetupExternalCache(context.Background(), &config.CacheConfig{DynamoDb: config.DynamoDbConfig{
		Enabled: true,
		Table:   tableName,
		Url:     s.addr,
	}}, telemetry.NewEmptyReporter(), log.NewNullLogger())
	assert.NoError(s.T(), err)
	defer store.Shutdown()
	assert.IsType(s.T(), &dynamoDbStore{}, store)
}
