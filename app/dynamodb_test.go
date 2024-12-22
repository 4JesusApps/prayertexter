package prayertexter

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type MockDDBConnecter struct {
	GetItemCalls   int
	GetItemInputs  []dynamodb.GetItemInput
	GetItemOutputs []*dynamodb.GetItemOutput
	GetItemErrors  []error

	PutItemCalls  int
	PutItemInputs []dynamodb.PutItemInput
	PutItemErrors []error

	DeleteItemCalls  int
	DeleteItemInputs []dynamodb.DeleteItemInput
	DeleteItemErrors []error
}

func (mock *MockDDBConnecter) GetItem(ctx context.Context,
	input *dynamodb.GetItemInput,
	opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {

	mock.GetItemCalls++
	mock.GetItemInputs = append(mock.GetItemInputs, *input)

	output := &dynamodb.GetItemOutput{}

	var err error

	if len(mock.GetItemOutputs) > 0 {
		output = mock.GetItemOutputs[0]
		// This removes the first get output from the slice once it is used in a mock
		// this allows for supplying multiple get outputs for different get methods
		mock.GetItemOutputs = mock.GetItemOutputs[1:]
	}
	if len(mock.GetItemErrors) > 0 {
		err = mock.GetItemErrors[0]
		// This removes the first error from the slice once it is used in a mock
		// this allows for supplying multiple errors for different get methods
		mock.GetItemErrors = mock.GetItemErrors[1:]
	}

	return output, err
}

func (mock *MockDDBConnecter) PutItem(ctx context.Context,
	input *dynamodb.PutItemInput,
	opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {

	mock.PutItemCalls++
	mock.PutItemInputs = append(mock.PutItemInputs, *input)

	var err error

	if len(mock.PutItemErrors) > 0 {
		err = mock.PutItemErrors[0]
		// This removes the first error from the slice once it is used in a mock
		// this allows for supplying multiple errors for different put methods
		mock.PutItemErrors = mock.PutItemErrors[1:]
	}

	return nil, err
}

func (mock *MockDDBConnecter) DeleteItem(ctx context.Context,
	input *dynamodb.DeleteItemInput,
	opts ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {

	mock.DeleteItemCalls++
	mock.DeleteItemInputs = append(mock.DeleteItemInputs, *input)

	var err error
	if len(mock.DeleteItemErrors) > 0 {
		err = mock.DeleteItemErrors[0]
		// This removes the first error from the slice once it is used in a mock
		// this allows for supplying multiple errors for different delete methods
		mock.DeleteItemErrors = mock.DeleteItemErrors[1:]
	}

	return nil, err
}
