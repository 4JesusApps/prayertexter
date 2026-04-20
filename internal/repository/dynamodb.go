package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DDBClient interface {
	GetItem(
		ctx context.Context,
		input *dynamodb.GetItemInput,
		opts ...func(*dynamodb.Options),
	) (*dynamodb.GetItemOutput, error)
	PutItem(
		ctx context.Context,
		input *dynamodb.PutItemInput,
		opts ...func(*dynamodb.Options),
	) (*dynamodb.PutItemOutput, error)
	DeleteItem(
		ctx context.Context,
		input *dynamodb.DeleteItemInput,
		opts ...func(*dynamodb.Options),
	) (*dynamodb.DeleteItemOutput, error)
	Scan(
		ctx context.Context,
		params *dynamodb.ScanInput,
		optFns ...func(*dynamodb.Options),
	) (*dynamodb.ScanOutput, error)
}

type DynamoDBRepository[T any] struct {
	client   DDBClient
	table    string
	keyField string
	timeout  int
}

func NewDynamoDBRepository[T any](client DDBClient, table, keyField string, timeout int) *DynamoDBRepository[T] {
	return &DynamoDBRepository[T]{
		client:   client,
		table:    table,
		keyField: keyField,
		timeout:  timeout,
	}
}

func (r *DynamoDBRepository[T]) Get(ctx context.Context, key string) (*T, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.timeout)*time.Second)
	defer cancel()

	input := &dynamodb.GetItemInput{
		TableName: &r.table,
		Key: map[string]types.AttributeValue{
			r.keyField: &types.AttributeValueMemberS{Value: key},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}

	resp, err := r.client.GetItem(ctx, input)
	if err != nil {
		return nil, utility.WrapError(err, fmt.Sprintf("failed to get item from table %s", r.table))
	}

	var item T
	if err = attributevalue.UnmarshalMap(resp.Item, &item); err != nil {
		return nil, utility.WrapError(err, fmt.Sprintf("failed to unmarshal item from table %s", r.table))
	}

	return &item, nil
}

func (r *DynamoDBRepository[T]) Save(ctx context.Context, item *T) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.timeout)*time.Second)
	defer cancel()

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return utility.WrapError(err, fmt.Sprintf("failed to marshal item for table %s", r.table))
	}

	input := &dynamodb.PutItemInput{
		TableName:              &r.table,
		Item:                   av,
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}

	_, err = r.client.PutItem(ctx, input)
	return utility.WrapError(err, fmt.Sprintf("failed to put item in table %s", r.table))
}

func (r *DynamoDBRepository[T]) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.timeout)*time.Second)
	defer cancel()

	input := &dynamodb.DeleteItemInput{
		TableName: &r.table,
		Key: map[string]types.AttributeValue{
			r.keyField: &types.AttributeValueMemberS{Value: key},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}

	_, err := r.client.DeleteItem(ctx, input)
	return utility.WrapError(err, fmt.Sprintf("failed to delete item from table %s", r.table))
}

func (r *DynamoDBRepository[T]) GetAll(ctx context.Context) ([]T, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.timeout)*time.Second)
	defer cancel()

	input := &dynamodb.ScanInput{
		TableName:              &r.table,
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}

	resp, err := r.client.Scan(ctx, input)
	if err != nil {
		return nil, utility.WrapError(err, fmt.Sprintf("failed to scan table %s", r.table))
	}

	items := make([]T, 0, len(resp.Items))
	for _, item := range resp.Items {
		var obj T
		if err = attributevalue.UnmarshalMap(item, &obj); err != nil {
			return nil, utility.WrapError(err, fmt.Sprintf("failed to unmarshal item from table %s", r.table))
		}
		items = append(items, obj)
	}

	return items, nil
}
