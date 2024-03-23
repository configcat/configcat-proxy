package cache

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/configcat/configcat-proxy/config"
	"github.com/configcat/configcat-proxy/log"
)

type dynamoDbStore struct {
	dynamoDb *dynamodb.Client
	table    *string
	log      log.Logger
}

func newDynamoDb(ctx context.Context, conf *config.DynamoDbConfig, log log.Logger) (External, error) {
	dynamoLog := log.WithPrefix("dynamodb")
	awsCtx, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		dynamoLog.Errorf("couldn't read aws config for DynamoDB: %s", err)
		return nil, err
	}
	var opts []func(*dynamodb.Options)
	if conf.Url != "" {
		opts = append(opts, func(options *dynamodb.Options) {
			options.BaseEndpoint = aws.String(conf.Url)
		})
	}
	log.Reportf("using DynamoDB for cache storage")
	return &dynamoDbStore{
		dynamoDb: dynamodb.NewFromConfig(awsCtx, opts...),
		table:    aws.String(conf.Table),
	}, nil
}

func (d *dynamoDbStore) Get(ctx context.Context, key string) ([]byte, error) {
	res, err := d.dynamoDb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: d.table,
		Key: map[string]types.AttributeValue{
			keyName: &types.AttributeValueMemberS{Value: key},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if payload, ok := res.Item[payloadName]; ok {
		switch v := payload.(type) {
		case *types.AttributeValueMemberB:
			return v.Value, nil
		default:
			return nil, fmt.Errorf("invalid item under key '%s'", key)
		}
	}
	return nil, fmt.Errorf("cache item not found for key '%s'", key)
}

func (d *dynamoDbStore) Set(ctx context.Context, key string, value []byte) error {
	_, err := d.dynamoDb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: d.table,
		Item: map[string]types.AttributeValue{
			keyName:     &types.AttributeValueMemberS{Value: key},
			payloadName: &types.AttributeValueMemberB{Value: value},
		},
	})
	return err
}

func (d *dynamoDbStore) Shutdown() {}
