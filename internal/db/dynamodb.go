/*
Package db implements dynamodb operations (get, put, delete).
*/
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/spf13/viper"
)

// Default values for configuration that has been exposed to be used with the config package.
const (
	DefaultTimeout    = 60
	TimeoutConfigPath = "conf.aws.db.timeout"
)

type DDBConnecter interface {
	GetItem(ctx context.Context, input *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (
		*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, input *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (
		*dynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context, input *dynamodb.DeleteItemInput, opts ...func(*dynamodb.Options)) (
		*dynamodb.DeleteItemOutput, error)
}

// GetDdbClient returns a dynamodb client that can be used for various dynamodb operations.
func GetDdbClient(ctx context.Context) (*dynamodb.Client, error) {
	cfg, err := utility.GetAwsConfig(ctx)
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

func getDdbItem(ctx context.Context, ddbClnt DDBConnecter, key, keyVal, table string) (*dynamodb.GetItemOutput, error) {
	timeout := viper.GetInt(TimeoutConfigPath)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	item, err := ddbClnt.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			key: &types.AttributeValueMemberS{Value: keyVal},
		},
	})

	return item, err
}

// GetDdbObject returns an object of various types from dynamodb. If the object key value does not exist in the
// dynamodb table, this will return an empty object.
func GetDdbObject[T any](ctx context.Context, ddbClnt DDBConnecter, key, keyVal, table string) (*T, error) {
	resp, err := getDdbItem(ctx, ddbClnt, key, keyVal, table)
	if err != nil {
		return nil, utility.WrapError(err, fmt.Sprintf("failed to get %T from table %s", *new(T), table))
	}

	var object T
	err = attributevalue.UnmarshalMap(resp.Item, &object)

	return &object, utility.WrapError(err, fmt.Sprintf("failed to unmarshal %T from table %s", *new(T), table))
}

func putDdbItem(ctx context.Context, ddbClnt DDBConnecter, table string, data map[string]types.AttributeValue) error {
	timeout := viper.GetInt(TimeoutConfigPath)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	_, err := ddbClnt.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &table,
		Item:      data,
	})

	return err
}

// PutDdbObject saves an object of various types to a dynamodb table.
func PutDdbObject[T any](ctx context.Context, ddbClnt DDBConnecter, table string, object *T) error {
	item, err := attributevalue.MarshalMap(object)
	if err != nil {
		return utility.WrapError(err, fmt.Sprintf("failed to marshal %T from table %s", *new(T), table))
	}

	if err := putDdbItem(ctx, ddbClnt, table, item); err != nil {
		return utility.WrapError(err, fmt.Sprintf("failed to put %T from table %s", *new(T), table))
	}

	return nil
}

// DelDdbItem deletes an item from a dynamodb table. If the item key value does not exist in the dynamodb table, it will
// not return an error.
func DelDdbItem(ctx context.Context, ddbClnt DDBConnecter, key, keyVal, table string) error {
	timeout := viper.GetInt(TimeoutConfigPath)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	_, err := ddbClnt.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			key: &types.AttributeValueMemberS{Value: keyVal},
		},
	})

	return utility.WrapError(err, fmt.Sprintf("failed to delete item from table %s", table))
}
