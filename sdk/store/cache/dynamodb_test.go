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
	"github.com/configcat/configcat-proxy/log"
	"github.com/configcat/go-sdk/v9/configcatcache"
	"github.com/stretchr/testify/assert"
)

const (
	tableName = "test-table"
	endpoint  = "http://localhost:8000"
)

func TestDynamoDbStore(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_SESSION_TOKEN", "session")
	t.Setenv("AWS_DEFAULT_REGION", "us-east-1")
	assert.NoError(t, createTableIfNotExist())

	t.Run("ok", func(t *testing.T) {
		store, err := newDynamoDb(context.Background(), &config.DynamoDbConfig{
			Enabled: true,
			Table:   tableName,
			Url:     endpoint,
		}, log.NewNullLogger())
		assert.NoError(t, err)
		defer store.Shutdown()

		cacheEntry := configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test`))

		err = store.Set(context.Background(), "k1", cacheEntry)
		assert.NoError(t, err)

		res, err := store.Get(context.Background(), "k1")
		assert.NoError(t, err)
		assert.Equal(t, cacheEntry, res)

		cacheEntry = configcatcache.CacheSegmentsToBytes(time.Now(), "etag", []byte(`test2`))

		err = store.Set(context.Background(), "k1", cacheEntry)
		assert.NoError(t, err)

		res, err = store.Get(context.Background(), "k1")
		assert.NoError(t, err)
		assert.Equal(t, cacheEntry, res)
	})

	t.Run("empty", func(t *testing.T) {
		store, err := newDynamoDb(context.Background(), &config.DynamoDbConfig{
			Enabled: true,
			Table:   tableName,
			Url:     endpoint,
		}, log.NewNullLogger())
		assert.NoError(t, err)
		defer store.Shutdown()

		_, err = store.Get(context.Background(), "k2")
		assert.Error(t, err)
	})

	t.Run("no-table", func(t *testing.T) {
		store, err := newDynamoDb(context.Background(), &config.DynamoDbConfig{
			Enabled: true,
			Table:   "nonexisting",
			Url:     endpoint,
		}, log.NewNullLogger())
		assert.NoError(t, err)
		defer store.Shutdown()

		_, err = store.Get(context.Background(), "k3")
		assert.Error(t, err)
	})
}

func createTableIfNotExist() error {
	awsCtx, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		return err
	}
	var opts []func(*dynamodb.Options)
	opts = append(opts, func(options *dynamodb.Options) {
		options.BaseEndpoint = aws.String(endpoint)
	})

	client := dynamodb.NewFromConfig(awsCtx, opts...)

	_, err = client.DescribeTable(context.Background(), &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err == nil {
		return nil
	}
	_, err = client.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
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
			res, err := client.DescribeTable(context.Background(), &dynamodb.DescribeTableInput{
				TableName: aws.String(tableName),
			})
			if err == nil && res.Table.TableStatus == types.TableStatusActive {
				return nil
			}
		}
	}
}
