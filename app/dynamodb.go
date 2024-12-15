package prayertexter

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		slog.Error("unable to load aws-sdk-go-v2 config")
		return nil, err
	}

	local, err := strconv.ParseBool(os.Getenv("AWS_SAM_LOCAL"))
	if err != nil {
		slog.Error("unable to convert AWS_SAM_LOCAL value to boolean")
		return nil, err
	}

	var clnt *dynamodb.Client

	if local {
		clnt = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String("http://dynamodb:8000")
		})
	} else {
		clnt = dynamodb.NewFromConfig(cfg)
	}

	return clnt, nil
}

func getItem(clnt DDBConnecter, attr, key, table string) (*dynamodb.GetItemOutput, error) {
	out, err := clnt.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			attr: &types.AttributeValueMemberS{Value: key},
		},
	})

	return out, err
}

func putItem(clnt DDBConnecter, table string, data map[string]types.AttributeValue) error {
	_, err := clnt.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &table,
		Item:      data,
	})

	return err
}

func delItem(clnt DDBConnecter, attr, key, table string) error {
	_, err := clnt.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: &table,
		Key: map[string]types.AttributeValue{
			attr: &types.AttributeValueMemberS{Value: key},
		},
	})

	return err
}
