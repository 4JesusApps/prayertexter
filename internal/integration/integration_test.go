//go:build integration

// Package integration_test runs end-to-end flow tests against a real DynamoDB Local
// instance launched via testcontainers-go. These tests target the Router.Handle entry
// point with real repositories, real services, and a recording in-memory MessageSender
// so we can assert both persisted state and outbound SMS at every step.
//
// Run with:
//
//	go test -tags integration ./internal/integration/...
package integration_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	tcdynamodb "github.com/testcontainers/testcontainers-go/modules/dynamodb"
)

const (
	memberTable       = "Member"
	activePrayerTable = "ActivePrayer"
	queuedPrayerTable = "QueuedPrayer"
	generalTable      = "General"
)

// ddbClient is shared by all tests in the suite. The underlying container is
// started once by TestMain and torn down on suite exit. Per-test isolation is
// handled by resetTables, not by recreating the container.
var ddbClient *dynamodb.Client

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcdynamodb.Run(ctx, "amazon/dynamodb-local:latest")
	if err != nil {
		log.Fatalf("start dynamodb container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			log.Printf("terminate dynamodb container: %v", err)
		}
	}()

	endpoint, err := container.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("get container endpoint: %v", err)
	}

	// Pin deterministic credentials so every test process sees the same DynamoDB
	// Local namespace. Without this, DynamoDB Local partitions data by
	// access-key+region pair and tests intermittently fail to find rows.
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-west-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		),
	)
	if err != nil {
		log.Fatalf("load aws config: %v", err)
	}

	ddbClient = dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String("http://" + endpoint)
	})

	if err := createTables(ctx, ddbClient); err != nil {
		log.Fatalf("create tables: %v", err)
	}

	os.Exit(m.Run())
}

// createTables creates the four DynamoDB tables used by the app, with key schemas
// that mirror dev/dynamodb/*.json and deploy/db/template.yaml.
func createTables(ctx context.Context, c *dynamodb.Client) error {
	tables := []struct {
		name string
		key  string
	}{
		{memberTable, "Phone"},
		{activePrayerTable, "IntercessorPhone"},
		{queuedPrayerTable, "IntercessorPhone"},
		{generalTable, "Key"},
	}

	for _, t := range tables {
		_, err := c.CreateTable(ctx, &dynamodb.CreateTableInput{
			TableName:   aws.String(t.name),
			BillingMode: types.BillingModePayPerRequest,
			AttributeDefinitions: []types.AttributeDefinition{
				{AttributeName: aws.String(t.key), AttributeType: types.ScalarAttributeTypeS},
			},
			KeySchema: []types.KeySchemaElement{
				{AttributeName: aws.String(t.key), KeyType: types.KeyTypeHash},
			},
		})
		if err != nil {
			return fmt.Errorf("create %s: %w", t.name, err)
		}
	}
	return nil
}

// resetTables clears all items from every table without dropping the tables
// themselves. Called from newTestEnv to give each test a clean DB.
func resetTables(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	tables := []struct {
		name string
		key  string
	}{
		{memberTable, "Phone"},
		{activePrayerTable, "IntercessorPhone"},
		{queuedPrayerTable, "IntercessorPhone"},
		{generalTable, "Key"},
	}

	for _, tbl := range tables {
		out, err := ddbClient.Scan(ctx, &dynamodb.ScanInput{TableName: aws.String(tbl.name)})
		if err != nil {
			t.Fatalf("scan %s: %v", tbl.name, err)
		}
		for _, item := range out.Items {
			_, err := ddbClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
				TableName: aws.String(tbl.name),
				Key: map[string]types.AttributeValue{
					tbl.key: item[tbl.key],
				},
			})
			if err != nil {
				t.Fatalf("delete from %s: %v", tbl.name, err)
			}
		}
	}
}
