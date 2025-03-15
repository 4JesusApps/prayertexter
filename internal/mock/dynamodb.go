package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type DDBConnecter struct {
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

func (m *DDBConnecter) GetItem(_ context.Context, input *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (
	*dynamodb.GetItemOutput, error) { //nolint:whitespace // line too long, needed new line

	m.GetItemCalls++
	m.GetItemInputs = append(m.GetItemInputs, *input)

	if len(m.GetItemResults) <= m.GetItemCalls-1 {
		return &dynamodb.GetItemOutput{}, nil
	}

	result := m.GetItemResults[m.GetItemCalls-1]
	return result.Output, result.Error
}

func (m *DDBConnecter) PutItem(_ context.Context, input *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (
	*dynamodb.PutItemOutput, error) { //nolint:whitespace // line too long, needed new line

	m.PutItemCalls++
	m.PutItemInputs = append(m.PutItemInputs, *input)

	if len(m.PutItemResults) <= m.PutItemCalls-1 {
		return &dynamodb.PutItemOutput{}, nil
	}

	result := m.PutItemResults[m.PutItemCalls-1]
	return nil, result.Error
}

func (m *DDBConnecter) DeleteItem(_ context.Context, input *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (
	*dynamodb.DeleteItemOutput, error) { //nolint:whitespace // line too long, needed new line

	m.DeleteItemCalls++
	m.DeleteItemInputs = append(m.DeleteItemInputs, *input)

	if len(m.DeleteItemResults) <= m.DeleteItemCalls-1 {
		return &dynamodb.DeleteItemOutput{}, nil
	}

	result := m.DeleteItemResults[m.DeleteItemCalls-1]
	return nil, result.Error
}
