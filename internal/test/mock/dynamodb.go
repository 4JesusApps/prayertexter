package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type DDBConnecter struct {
	GetItemCalls    int
	PutItemCalls    int
	DeleteItemCalls int
	ScanCalls       int

	GetItemInputs    []dynamodb.GetItemInput
	PutItemInputs    []dynamodb.PutItemInput
	DeleteItemInputs []dynamodb.DeleteItemInput
	ScanInputs       []dynamodb.ScanInput

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

	ScanResults []struct {
		Output *dynamodb.ScanOutput
		Error  error
	}
}

func (d *DDBConnecter) GetItem(_ context.Context, input *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	d.GetItemCalls++
	d.GetItemInputs = append(d.GetItemInputs, *input)

	// This helps if no mock result is provided, it will at least return empty results
	if len(d.GetItemResults) <= d.GetItemCalls-1 {
		return &dynamodb.GetItemOutput{}, nil
	}

	result := d.GetItemResults[d.GetItemCalls-1]
	return result.Output, result.Error
}

func (d *DDBConnecter) PutItem(_ context.Context, input *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	d.PutItemCalls++
	d.PutItemInputs = append(d.PutItemInputs, *input)

	// This helps if no mock result is provided, it will at least return empty results
	if len(d.PutItemResults) <= d.PutItemCalls-1 {
		return &dynamodb.PutItemOutput{}, nil
	}

	result := d.PutItemResults[d.PutItemCalls-1]
	return nil, result.Error
}

func (d *DDBConnecter) DeleteItem(_ context.Context, input *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	d.DeleteItemCalls++
	d.DeleteItemInputs = append(d.DeleteItemInputs, *input)

	// This helps if no mock result is provided, it will at least return empty results
	if len(d.DeleteItemResults) <= d.DeleteItemCalls-1 {
		return &dynamodb.DeleteItemOutput{}, nil
	}

	result := d.DeleteItemResults[d.DeleteItemCalls-1]
	return nil, result.Error
}

func (d *DDBConnecter) Scan(_ context.Context, input *dynamodb.ScanInput, _ ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	d.ScanCalls++
	d.ScanInputs = append(d.ScanInputs, *input)

	// This helps if no mock result is provided, it will at least return empty results
	if len(d.ScanResults) <= d.ScanCalls-1 {
		return &dynamodb.ScanOutput{}, nil
	}

	result := d.ScanResults[d.ScanCalls-1]
	return result.Output, result.Error
}
