package object_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/test/mock"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestUpdate(t *testing.T) {
	mockGetItemResults := []struct {
		Output *dynamodb.GetItemOutput
		Error  error
	}{
		{
			Output: &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"Key": &types.AttributeValueMemberS{Value: object.StateTrackerKeyValue},
					"States": &types.AttributeValueMemberL{
						Value: []types.AttributeValue{
							&types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Error": &types.AttributeValueMemberS{Value: "sample error text"},
									"Message": &types.AttributeValueMemberM{
										Value: map[string]types.AttributeValue{
											"Body":  &types.AttributeValueMemberS{Value: "sample text message 1"},
											"Phone": &types.AttributeValueMemberS{Value: "+11234567890"},
										},
									},
									"ID":        &types.AttributeValueMemberS{Value: "67f8ce776cc147c2b8700af909639ba2"},
									"Stage":     &types.AttributeValueMemberS{Value: "HELP"},
									"Status":    &types.AttributeValueMemberS{Value: "FAILED"},
									"TimeStart": &types.AttributeValueMemberS{Value: "2025-02-16T23:54:01Z"},
								},
							},
						},
					},
				},
			},
			Error: nil,
		},
	}

	expectedStateTracker := object.StateTracker{
		Key: object.StateTrackerKeyValue,
		States: []object.State{
			{
				Error: "sample error text",
				Message: messaging.TextMessage{
					Body:  "sample text message 1",
					Phone: "+11234567890",
				},
				ID:        "67f8ce776cc147c2b8700af909639ba2",
				Stage:     "HELP",
				Status:    "FAILED",
				TimeStart: "2025-02-16T23:54:01Z",
			},
			{
				Error: "",
				Message: messaging.TextMessage{
					Body:  "sample text message 2",
					Phone: "+19987654321",
				},
				ID:        "19ee2955d41d08325e1a97cbba1e544b",
				Stage:     "MEMBER DELETE",
				Status:    "IN PROGRESS",
				TimeStart: "2025-02-16T23:57:01Z",
			},
		},
	}

	newState := object.State{
		Error: "",
		Message: messaging.TextMessage{
			Body:  "sample text message 2",
			Phone: "+19987654321",
		},
		ID:        "19ee2955d41d08325e1a97cbba1e544b",
		Stage:     "MEMBER DELETE",
		Status:    "IN PROGRESS",
		TimeStart: "2025-02-16T23:57:01Z",
	}

	ddbMock := &mock.DDBConnecter{}
	ddbMock.GetItemResults = mockGetItemResults
	ctx := context.Background()

	t.Run("add new State to StateTracker", func(t *testing.T) {
		if err := newState.Update(ctx, ddbMock, false); err != nil {
			t.Errorf("unexpected error %v", err)
		}
		input := ddbMock.PutItemInputs[0]
		testStateTracker(input, t, expectedStateTracker)
	})

	t.Run("removes new State to StateTracker", func(t *testing.T) {
		// this resets the GetItem mock so that it can re-use mockGetItemResults
		ddbMock.GetItemCalls = 0
		if err := newState.Update(ctx, ddbMock, true); err != nil {
			t.Errorf("unexpected error %v", err)
		}

		// this removes the last State because remove is set to true, so it is expected to not be there
		expectedStateTracker.States = expectedStateTracker.States[:1]

		input := ddbMock.PutItemInputs[1]
		testStateTracker(input, t, expectedStateTracker)
	})
}

func testStateTracker(input dynamodb.PutItemInput, t *testing.T, expectedStateTracker object.StateTracker) {
	if *input.TableName != object.DefaultStateTrackerTable {
		t.Errorf("expected table %v, got %v", object.DefaultStateTrackerTable, *input.TableName)
	}

	actualStateTracker := object.StateTracker{}
	if err := attributevalue.UnmarshalMap(input.Item, &actualStateTracker); err != nil {
		t.Errorf("failed to unmarshal PutItemInput into StateTracker: %v", err)
	}

	if !reflect.DeepEqual(actualStateTracker, expectedStateTracker) {
		t.Errorf("expected StateTracker %v, got %v", expectedStateTracker, actualStateTracker)
	}
}
