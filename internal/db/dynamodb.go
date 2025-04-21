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
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (
		*dynamodb.ScanOutput, error)
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

	input := &dynamodb.GetItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			key: &types.AttributeValueMemberS{Value: keyVal},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}

	item, err := ddbClnt.GetItem(ctx, input)

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

	input := &dynamodb.PutItemInput{
		TableName:              &table,
		Item:                   data,
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}

	_, err := ddbClnt.PutItem(ctx, input)

	return err
}

// PutDdbObject saves an object of various types to a dynamodb table.
func PutDdbObject[T any](ctx context.Context, ddbClnt DDBConnecter, table string, object *T) error {
	item, err := attributevalue.MarshalMap(object)
	if err != nil {
		return utility.WrapError(err, fmt.Sprintf("failed to marshal %T from table %s", *new(T), table))
	}

	if err = putDdbItem(ctx, ddbClnt, table, item); err != nil {
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

	input := &dynamodb.DeleteItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			key: &types.AttributeValueMemberS{Value: keyVal},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}

	_, err := ddbClnt.DeleteItem(ctx, input)

	return utility.WrapError(err, fmt.Sprintf("failed to delete item from table %s", table))
}

func getAllItems(ctx context.Context, ddbClnt DDBConnecter, table string) (*dynamodb.ScanOutput, error) {
	timeout := viper.GetInt(TimeoutConfigPath)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	input := &dynamodb.ScanInput{
		TableName:              aws.String(table),
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}

	items, err := ddbClnt.Scan(ctx, input)

	if err != nil {
		return nil, err
	}

	return items, nil
}

// GetAllObjects returns a slice of object of various types from dynamodb. If the table is empty, it will return an
// empty slice.
func GetAllObjects[T any](ctx context.Context, ddbClnt DDBConnecter, table string) ([]T, error) {
	items, err := getAllItems(ctx, ddbClnt, table)

	if err != nil {
		return nil, utility.WrapError(err, fmt.Sprintf("failed to scan items of %T from table %s", *new(T), table))
	}

	objects := make([]T, 0, len(items.Items))

	for _, item := range items.Items {
		var object T
		if err = attributevalue.UnmarshalMap(item, &object); err != nil {
			return nil, utility.WrapError(err, fmt.Sprintf("failed to unmarshal %T from table %s", *new(T), table))
		}
		objects = append(objects, object)
	}

	return objects, nil
}
