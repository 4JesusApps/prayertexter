package prayertexter

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
	cfg, err := getAwsConfig()
	if err != nil {
		return nil, fmt.Errorf("GetDdbClient: %w", err)
	}

	var ddbClnt *dynamodb.Client

	if isAwsLocal() {
		ddbClnt = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String("http://dynamodb:8000")
		})
	} else {
		ddbClnt = dynamodb.NewFromConfig(cfg)
	}

	return ddbClnt, nil
}

func getDdbItem(ddbClnt DDBConnecter, attr, key, table string) (*dynamodb.GetItemOutput, error) {
	item, err := ddbClnt.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			attr: &types.AttributeValueMemberS{Value: key},
		},
	})

	return item, err
}

func getDdbObject[T any](ddbClnt DDBConnecter, attr, key, table string) (*T, error) {
	resp, err := getDdbItem(ddbClnt, attr, key, table)
	if err != nil {
		return nil, fmt.Errorf("getDdbItem: %w", err)
	}

	var object T
	if err := attributevalue.UnmarshalMap(resp.Item, &object); err != nil {
		return nil, fmt.Errorf("getDdbObject failed unmarshal: %w", err)
	}

	return &object, nil
}

func putDdbItem(ddbClnt DDBConnecter, table string, data map[string]types.AttributeValue) error {
	_, err := ddbClnt.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &table,
		Item:      data,
	})

	return err
}

func putDdbObject[T any](ddbClnt DDBConnecter, table string, object *T) error {
	item, err := attributevalue.MarshalMap(object)
	if err != nil {
		return fmt.Errorf("putDdbObject failed marshal: %w", err)
	}

	if err := putDdbItem(ddbClnt, table, item); err != nil {
		return fmt.Errorf("putDdbItem: %w", err)
	}

	return nil
}

func delDdbItem(ddbClnt DDBConnecter, attr, key, table string) error {
	_, err := ddbClnt.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			attr: &types.AttributeValueMemberS{Value: key},
		},
	})

	return err
}
