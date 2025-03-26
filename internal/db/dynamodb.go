package db

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/mshort55/prayertexter/internal/utility"
)

const (
	ddbTimeout = 60
)

type DDBConnecter interface {
	GetItem(ctx context.Context,
		input *dynamodb.GetItemInput,
		opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context,
		input *dynamodb.PutItemInput,
		opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context,
		input *dynamodb.DeleteItemInput,
		opts ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
}

func GetDdbClient() (*dynamodb.Client, error) {
	cfg, err := utility.GetAwsConfig()
	if err != nil {
		return nil, utility.WrapError(err, "failed to get dynamodb client")
	}

	var ddbClnt *dynamodb.Client

	if utility.IsAwsLocal() {
		ddbClnt = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String("http://dynamodb:8000")
		})
	} else {
		ddbClnt = dynamodb.NewFromConfig(cfg)
	}

	return ddbClnt, nil
}

func getDdbItem(ddbClnt DDBConnecter, attr, key, table string) (*dynamodb.GetItemOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ddbTimeout*time.Second)
	defer cancel()

	item, err := ddbClnt.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			attr: &types.AttributeValueMemberS{Value: key},
		},
	})

	return item, err
}

func GetDdbObject[T any](ddbClnt DDBConnecter, attr, key, table string) (*T, error) {
	resp, err := getDdbItem(ddbClnt, attr, key, table)
	if err != nil {
		return nil, utility.WrapError(err, fmt.Sprintf("failed to get %T from table %s", *new(T), table))
	}

	var object T
	err = attributevalue.UnmarshalMap(resp.Item, &object)

	return &object, utility.WrapError(err, fmt.Sprintf("failed to unmarshal %T from table %s", *new(T), table))
}

func putDdbItem(ddbClnt DDBConnecter, table string, data map[string]types.AttributeValue) error {
	ctx, cancel := context.WithTimeout(context.Background(), ddbTimeout*time.Second)
	defer cancel()

	_, err := ddbClnt.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &table,
		Item:      data,
	})

	return err
}

func PutDdbObject[T any](ddbClnt DDBConnecter, table string, object *T) error {
	item, err := attributevalue.MarshalMap(object)
	if err != nil {
		return utility.WrapError(err, fmt.Sprintf("failed to marshal %T from table %s", *new(T), table))
	}

	if err := putDdbItem(ddbClnt, table, item); err != nil {
		return utility.WrapError(err, fmt.Sprintf("failed to put %T from table %s", *new(T), table))
	}

	return nil
}

func DelDdbItem(ddbClnt DDBConnecter, attr, key, table string) error {
	ctx, cancel := context.WithTimeout(context.Background(), ddbTimeout*time.Second)
	defer cancel()

	_, err := ddbClnt.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			attr: &types.AttributeValueMemberS{Value: key},
		},
	})

	return utility.WrapError(err, fmt.Sprintf("failed to delete item from table %s", table))
}
