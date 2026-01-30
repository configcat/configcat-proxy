package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/diag/telemetry"
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcdynamodb "github.com/testcontainers/testcontainers-go/modules/dynamodb"
)

const (
	tableName = "test-table"
)

type dynamoDbTestSuite struct {
	suite.Suite

	db   *tcdynamodb.DynamoDBContainer
	addr string
}

func (s *dynamoDbTestSuite) SetupSuite() {
	dynamoDbContainer, err := tcdynamodb.Run(s.T().Context(), "amazon/dynamodb-local")
	if err != nil {
		panic("failed to start container: " + err.Error() + "")
	}
	s.db = dynamoDbContainer
	str, _ := s.db.ConnectionString(s.T().Context())
	s.addr = "http://" + str

	s.T().Setenv("AWS_ACCESS_KEY_ID", "key")
	s.T().Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	s.T().Setenv("AWS_SESSION_TOKEN", "session")
	s.T().Setenv("AWS_DEFAULT_REGION", "us-east-1")
}

func (s *dynamoDbTestSuite) TearDownSuite() {
	if err := testcontainers.TerminateContainer(s.db); err != nil {
		panic("failed to terminate container: " + err.Error() + "")
	}
}

func TestRunDynamoDbSuite(t *testing.T) {
	suite.Run(t, new(dynamoDbTestSuite))
}

func (s *dynamoDbTestSuite) TestDynamoDbStore() {
	assert.NoError(s.T(), createTableIfNotExist(s.T().Context(), tableName, s.addr))

	s.Run("ok", func() {
		store, err := newDynamoDb(s.T().Context(), &config.DynamoDbConfig{
			Enabled: true,
			Table:   tableName,
			Url:     s.addr,
		}, telemetry.NewEmptyReporter(), log.NewNullLogger())
		assert.NoError(s.T(), err)
		defer store.Shutdown()

		cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`))

		err = store.Set(s.T().Context(), "k1", cacheEntry)
		assert.NoError(s.T(), err)

		res, err := store.Get(s.T().Context(), "k1")
		assert.NoError(s.T(), err)
		assert.Equal(s.T(), cacheEntry, res)

		cacheEntry = configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test2`))

		err = store.Set(s.T().Context(), "k1", cacheEntry)
		assert.NoError(s.T(), err)

		res, err = store.Get(s.T().Context(), "k1")
		assert.NoError(s.T(), err)
		assert.Equal(s.T(), cacheEntry, res)
	})

	s.Run("empty", func() {
		store, err := newDynamoDb(s.T().Context(), &config.DynamoDbConfig{
			Enabled: true,
			Table:   tableName,
			Url:     s.addr,
		}, telemetry.NewEmptyReporter(), log.NewNullLogger())
		assert.NoError(s.T(), err)
		defer store.Shutdown()

		_, err = store.Get(s.T().Context(), "k2")
		assert.Error(s.T(), err)
	})

	s.Run("no-table", func() {
		store, err := newDynamoDb(s.T().Context(), &config.DynamoDbConfig{
			Enabled: true,
			Table:   "nonexisting",
			Url:     s.addr,
		}, telemetry.NewEmptyReporter(), log.NewNullLogger())
		assert.NoError(s.T(), err)
		defer store.Shutdown()

		_, err = store.Get(s.T().Context(), "k3")
		assert.Error(s.T(), err)
	})
}

func createTableIfNotExist(ctx context.Context, table string, addr string) error {
	awsCtx, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	var opts []func(*dynamodb.Options)
	opts = append(opts, func(options *dynamodb.Options) {
		options.BaseEndpoint = aws.String(addr)
	})

	client := dynamodb.NewFromConfig(awsCtx, opts...)

	_, err = client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(table),
	})
	if err == nil {
		return nil
	}
	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(table),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String(keyName),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String(keyName),
				KeyType:       types.KeyTypeHash,
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
	})
	if err != nil {
		return err
	}

	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-timeout:
			return fmt.Errorf("table creation timed out")
		case <-ticker.C:
			res, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
				TableName: aws.String(table),
			})
			if err == nil && res.Table.TableStatus == types.TableStatusActive {
				return nil
			}
		}
	}
}
