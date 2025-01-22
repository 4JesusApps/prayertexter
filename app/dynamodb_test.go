package prayertexter

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type MockDDBConnecter struct {
	GetItemCalls    int
	PutItemCalls    int
	DeleteItemCalls int

	GetItemInputs    []dynamodb.GetItemInput
	PutItemInputs    []dynamodb.PutItemInput
	DeleteItemInputs []dynamodb.DeleteItemInput

	GetItemResults []struct {
		Output *dynamodb.GetItemOutput
		Error  error
	}
	PutItemResults []struct {
		Error error
	}
	DeleteItemResults []struct {
		Error error
	}
}

func (mock *MockDDBConnecter) GetItem(ctx context.Context, input *dynamodb.GetItemInput,
	opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {

	mock.GetItemCalls++
	mock.GetItemInputs = append(mock.GetItemInputs, *input)

	if len(mock.GetItemResults) <= mock.GetItemCalls-1 {
		slog.Info("no GetItem mock loaded; returning empty output and nil error")
		return &dynamodb.GetItemOutput{}, nil
	}

	result := mock.GetItemResults[mock.GetItemCalls-1]
	return result.Output, result.Error
}

func (mock *MockDDBConnecter) PutItem(ctx context.Context, input *dynamodb.PutItemInput,
	opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {

	mock.PutItemCalls++
	mock.PutItemInputs = append(mock.PutItemInputs, *input)

	if len(mock.PutItemResults) <= mock.PutItemCalls-1 {
		slog.Info("no PutItem mock loaded; returning empty output and nil error")
		return &dynamodb.PutItemOutput{}, nil
	}

	result := mock.PutItemResults[mock.PutItemCalls-1]
	return nil, result.Error
}

func (mock *MockDDBConnecter) DeleteItem(ctx context.Context, input *dynamodb.DeleteItemInput,
	opts ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {

	mock.DeleteItemCalls++
	mock.DeleteItemInputs = append(mock.DeleteItemInputs, *input)

	if len(mock.DeleteItemResults) <= mock.DeleteItemCalls-1 {
		slog.Info("no DeleteItem mock loaded; returning empty output and nil error")
		return &dynamodb.DeleteItemOutput{}, nil
	}

	result := mock.DeleteItemResults[mock.DeleteItemCalls-1]
	return nil, result.Error
}
