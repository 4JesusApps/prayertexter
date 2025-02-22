package prayertexter

import (
	"context"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
		return &dynamodb.DeleteItemOutput{}, nil
	}

	result := mock.DeleteItemResults[mock.DeleteItemCalls-1]
	return nil, result.Error
}

var expectedDdbItems = []struct {
	Output *dynamodb.GetItemOutput
	Error  error
}{
	// member test
	{
		Output: &dynamodb.GetItemOutput{
			Item: map[string]types.AttributeValue{
				"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
				"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
				"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
				"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
				"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
				"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
				"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2025-02-16T23:54:01Z"},
				"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
			},
		},
		Error: nil,
	},
	// intercessorphones test
	{
		Output: &dynamodb.GetItemOutput{
			Item: map[string]types.AttributeValue{
				"Key": &types.AttributeValueMemberS{Value: intercessorPhonesKey},
				"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
					&types.AttributeValueMemberS{Value: "111-111-1111"},
					&types.AttributeValueMemberS{Value: "222-222-2222"},
				}},
			},
		},
		Error: nil,
	},
	// prayer test
	{
		Output: &dynamodb.GetItemOutput{
			Item: map[string]types.AttributeValue{
				"Intercessor": &types.AttributeValueMemberM{
					Value: map[string]types.AttributeValue{
						"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
						"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
						"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
						"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
						"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
						"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
						"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2025-02-13T23:54:01Z"},
						"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
					},
				},
				"IntercessorPhone": &types.AttributeValueMemberS{Value: "111-111-1111"},
				"Request":          &types.AttributeValueMemberS{Value: "I need prayer for..."},
				"Requestor": &types.AttributeValueMemberM{
					Value: map[string]types.AttributeValue{
						"Intercessor":       &types.AttributeValueMemberBOOL{Value: false},
						"Name":              &types.AttributeValueMemberS{Value: "John Doe"},
						"Phone":             &types.AttributeValueMemberS{Value: "123-456-7890"},
						"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
						"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
						"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
						"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: ""},
						"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "0"},
					},
				},
			},
		},
		Error: nil,
	},
	// statetracker test
	{
		Output: &dynamodb.GetItemOutput{
			Item: map[string]types.AttributeValue{
				"Key": &types.AttributeValueMemberS{Value: stateTrackerKey},
				"States": &types.AttributeValueMemberL{
					Value: []types.AttributeValue{
						&types.AttributeValueMemberM{
							Value: map[string]types.AttributeValue{
								"Error": &types.AttributeValueMemberS{Value: "sample error text"},
								"Message": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Body":  &types.AttributeValueMemberS{Value: "sample text message 1"},
										"Phone": &types.AttributeValueMemberS{Value: "123-456-7890"},
									},
								},
								"RequestID":  &types.AttributeValueMemberS{Value: "f88f9757-cecb-4b7f-a3ab-e27c07915b70"},
								"Stage":      &types.AttributeValueMemberS{Value: "HELP"},
								"Status":     &types.AttributeValueMemberS{Value: "FAILED"},
								"TimeStart":  &types.AttributeValueMemberS{Value: "2025-02-16T23:54:01Z"},
								"TimeFinish": &types.AttributeValueMemberS{Value: "2025-02-16T23:54:02Z"},
							},
						},
						&types.AttributeValueMemberM{
							Value: map[string]types.AttributeValue{
								"Error": &types.AttributeValueMemberS{Value: ""},
								"Message": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Body":  &types.AttributeValueMemberS{Value: "sample text message 2"},
										"Phone": &types.AttributeValueMemberS{Value: "998-765-4321"},
									},
								},
								"RequestID":  &types.AttributeValueMemberS{Value: "5bb663c9-c95d-4ee7-b895-a542f01fa23b"},
								"Stage":      &types.AttributeValueMemberS{Value: "MEMBER DELETE"},
								"Status":     &types.AttributeValueMemberS{Value: "COMPLETED"},
								"TimeStart":  &types.AttributeValueMemberS{Value: "2025-02-16T23:57:01Z"},
								"TimeFinish": &types.AttributeValueMemberS{Value: "2025-02-16T23:57:02Z"},
							},
						},
					},
				},
			},
		},
		Error: nil,
	},
}

var expectedObjects = []any{
	&Member{
		Intercessor:       true,
		Name:              "Intercessor1",
		Phone:             "111-111-1111",
		PrayerCount:       1,
		SetupStage:        99,
		SetupStatus:       "completed",
		WeeklyPrayerDate:  "2025-02-16T23:54:01Z",
		WeeklyPrayerLimit: 5,
	},
	&IntercessorPhones{
		Key: intercessorPhonesKey,
		Phones: []string{
			"111-111-1111",
			"222-222-2222",
		},
	},
	&Prayer{
		Intercessor: Member{
			Intercessor:       true,
			Name:              "Intercessor1",
			Phone:             "111-111-1111",
			PrayerCount:       1,
			SetupStage:        99,
			SetupStatus:       "completed",
			WeeklyPrayerDate:  "2025-02-13T23:54:01Z",
			WeeklyPrayerLimit: 5,
		},
		IntercessorPhone: "111-111-1111",
		Request:          "I need prayer for...",
		Requestor: Member{
			Intercessor:       false,
			Name:              "John Doe",
			Phone:             "123-456-7890",
			PrayerCount:       0,
			SetupStage:        99,
			SetupStatus:       "completed",
			WeeklyPrayerDate:  "",
			WeeklyPrayerLimit: 0,
		},
	},
	&StateTracker{
		Key: stateTrackerKey,
		States: []State{
			{
				Error: "sample error text",
				Message: TextMessage{
					Body:  "sample text message 1",
					Phone: "123-456-7890",
				},
				RequestID:  "f88f9757-cecb-4b7f-a3ab-e27c07915b70",
				Stage:      "HELP",
				Status:     "FAILED",
				TimeStart:  "2025-02-16T23:54:01Z",
				TimeFinish: "2025-02-16T23:54:02Z",
			},
			{
				Error: "",
				Message: TextMessage{
					Body:  "sample text message 2",
					Phone: "998-765-4321",
				},
				RequestID:  "5bb663c9-c95d-4ee7-b895-a542f01fa23b",
				Stage:      "MEMBER DELETE",
				Status:     "COMPLETED",
				TimeStart:  "2025-02-16T23:57:01Z",
				TimeFinish: "2025-02-16T23:57:02Z",
			},
		},
	},
}

func TestGetDdbObject(t *testing.T) {
	ddbMock := &MockDDBConnecter{}
	ddbMock.GetItemResults = expectedDdbItems

	for _, expectedObject := range expectedObjects {
		switch obj := expectedObject.(type) {
		case *Member:
			testObject(t, ddbMock, obj)
		case *IntercessorPhones:
			testObject(t, ddbMock, obj)
		case *Prayer:
			testObject(t, ddbMock, obj)
		case *StateTracker:
			testObject(t, ddbMock, obj)
		default:
			t.Errorf("unexpected type %T", expectedObject)
		}
	}
}

func testObject[T any](t *testing.T, ddbMock DDBConnecter, expectedObject *T) {
	testedObject, err := getDdbObject[T](ddbMock, "test", "test", "test")
	if err != nil {
		t.Errorf("getDdbObject failed for type %T: %v", expectedObject, err)
	}

	if !reflect.DeepEqual(testedObject, expectedObject) {
		t.Errorf("expected object %v of type %T, got %v of type %T", expectedObject, expectedObject, testedObject, testedObject)
	}
}

func TestPutDdbObject(t *testing.T) {
	ddbMock := &MockDDBConnecter{}

	for index, expectedObject := range expectedObjects {
		switch obj := expectedObject.(type) {
		case *Member:
			testPutObject(t, ddbMock, obj, index)
		case *IntercessorPhones:
			testPutObject(t, ddbMock, obj, index)
		case *Prayer:
			testPutObject(t, ddbMock, obj, index)
		case *StateTracker:
			testPutObject(t, ddbMock, obj, index)
		default:
			t.Errorf("unexpected type %T", expectedObject)
		}
	}
}

func testPutObject[T any](t *testing.T, ddbMock *MockDDBConnecter, expectedObject *T, index int) {
	err := putDdbObject(ddbMock, "test", expectedObject)
	if err != nil {
		t.Errorf("putDdbObject failed for type %T: %v", expectedObject, err)
	}

	expectedDdbItem := expectedDdbItems[index].Output.Item
	lastPutItem := ddbMock.PutItemInputs[len(ddbMock.PutItemInputs)-1].Item

	expectedMap := make(map[string]interface{})
	lastPutMap := make(map[string]interface{})

	if err := attributevalue.UnmarshalMap(expectedDdbItem, &expectedMap); err != nil {
		t.Errorf("failed to unmarshal expectedDdbItem: %v", err)
	}

	if err := attributevalue.UnmarshalMap(lastPutItem, &lastPutMap); err != nil {
		t.Errorf("failed to unmarshal lastPutItem: %v", err)
	}

	if !reflect.DeepEqual(expectedMap, lastPutMap) {
		t.Errorf("expected map %v, got %v", expectedMap, lastPutMap)
	}
}
