package db_test

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/test/mock"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestDynamoDBOperations(t *testing.T) {
	expectedDdbItems := []struct {
		Output *dynamodb.GetItemOutput
		Error  error
	}{
		// Member
		{
			Output: &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"Administrator":     &types.AttributeValueMemberBOOL{Value: false},
					"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
					"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
					"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
					"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
					"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
					"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
					"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2025-02-16T23:54:01Z"},
					"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
				},
			},
			Error: nil,
		},
		// BlockedPhones
		{
			Output: &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"Key": &types.AttributeValueMemberS{Value: object.BlockedPhonesKeyValue},
					"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
						&types.AttributeValueMemberS{Value: "+13333333333"},
						&types.AttributeValueMemberS{Value: "+14444444444"},
					}},
				},
			},
			Error: nil,
		},
		// IntercessorPhones
		{
			Output: &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
					"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
						&types.AttributeValueMemberS{Value: "+11111111111"},
						&types.AttributeValueMemberS{Value: "+12222222222"},
					}},
				},
			},
			Error: nil,
		},
		// Prayer
		{
			Output: &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"Intercessor": &types.AttributeValueMemberM{
						Value: map[string]types.AttributeValue{
							"Administrator":     &types.AttributeValueMemberBOOL{Value: false},
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2025-02-13T23:54:01Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					"IntercessorPhone": &types.AttributeValueMemberS{Value: "+11111111111"},
					"ReminderCount":    &types.AttributeValueMemberN{Value: "3"},
					"ReminderDate":     &types.AttributeValueMemberS{Value: "2025-02-13T23:54:01Z"},
					"Request":          &types.AttributeValueMemberS{Value: "I need prayer for..."},
					"Requestor": &types.AttributeValueMemberM{
						Value: map[string]types.AttributeValue{
							"Administrator":     &types.AttributeValueMemberBOOL{Value: false},
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: false},
							"Name":              &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11234567890"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: ""},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "0"},
						},
					},
				},
			},
			Error: nil,
		},
	}

	expectedObjects := []any{
		&object.Member{
			Administrator:     false,
			Intercessor:       true,
			Name:              "Intercessor1",
			Phone:             "+11111111111",
			PrayerCount:       1,
			SetupStage:        object.MemberSignUpStepFinal,
			SetupStatus:       object.MemberSetupComplete,
			WeeklyPrayerDate:  "2025-02-16T23:54:01Z",
			WeeklyPrayerLimit: 5,
		},
		&object.BlockedPhones{
			Key: object.BlockedPhonesKeyValue,
			Phones: []string{
				"+13333333333",
				"+14444444444",
			},
		},
		&object.IntercessorPhones{
			Key: object.IntercessorPhonesKeyValue,
			Phones: []string{
				"+11111111111",
				"+12222222222",
			},
		},
		&object.Prayer{
			Intercessor: object.Member{
				Administrator:     false,
				Intercessor:       true,
				Name:              "Intercessor1",
				Phone:             "+11111111111",
				PrayerCount:       1,
				SetupStage:        object.MemberSignUpStepFinal,
				SetupStatus:       object.MemberSetupComplete,
				WeeklyPrayerDate:  "2025-02-13T23:54:01Z",
				WeeklyPrayerLimit: 5,
			},
			IntercessorPhone: "+11111111111",
			ReminderCount:    3,
			ReminderDate:     "2025-02-13T23:54:01Z",
			Request:          "I need prayer for...",
			Requestor: object.Member{
				Administrator:     false,
				Intercessor:       false,
				Name:              "John Doe",
				Phone:             "+11234567890",
				PrayerCount:       0,
				SetupStage:        object.MemberSignUpStepFinal,
				SetupStatus:       object.MemberSetupComplete,
				WeeklyPrayerDate:  "",
				WeeklyPrayerLimit: 0,
			},
		},
	}

	t.Run("GetDdbObject", func(t *testing.T) {
		ddbMock := &mock.DDBConnecter{}
		ddbMock.GetItemResults = expectedDdbItems

		for _, obj := range expectedObjects {
			t.Run(fmt.Sprintf("Get %T", obj), func(t *testing.T) {
				switch o := obj.(type) {
				case *object.Member:
					testGetObject(t, ddbMock, o)
				case *object.BlockedPhones:
					testGetObject(t, ddbMock, o)
				case *object.IntercessorPhones:
					testGetObject(t, ddbMock, o)
				case *object.Prayer:
					testGetObject(t, ddbMock, o)
				default:
					t.Errorf("unexpected object type %T", obj)
				}
			})
		}
	})

	t.Run("PutDdbObject", func(t *testing.T) {
		ddbMock := &mock.DDBConnecter{}

		for index, obj := range expectedObjects {
			t.Run(fmt.Sprintf("Put %T", obj), func(t *testing.T) {
				switch o := obj.(type) {
				case *object.Member:
					testPutObject(t, ddbMock, o, expectedDdbItems[index])
				case *object.BlockedPhones:
					testPutObject(t, ddbMock, o, expectedDdbItems[index])
				case *object.IntercessorPhones:
					testPutObject(t, ddbMock, o, expectedDdbItems[index])
				case *object.Prayer:
					testPutObject(t, ddbMock, o, expectedDdbItems[index])
				default:
					t.Errorf("unexpected object type %T", obj)
				}
			})
		}
	})
}

func testGetObject[T any](t *testing.T, ddbMock db.DDBConnecter, expectedObject *T) {
	ctx := context.Background()
	// The parameters test test test are used here because mocking makes using real parameters unnecessary.
	testedObject, err := db.GetDdbObject[T](ctx, ddbMock, "test", "test", "test")
	if err != nil {
		t.Errorf("getDdbObject failed for type %T: %v", expectedObject, err)
	}

	if !reflect.DeepEqual(testedObject, expectedObject) {
		t.Errorf("expected object %v of type %T, got %v of type %T",
			expectedObject, expectedObject, testedObject, testedObject)
	}
}

func testPutObject[T any](t *testing.T, ddbMock *mock.DDBConnecter, expectedObject *T, expectedDdbItem struct {
	Output *dynamodb.GetItemOutput
	Error  error
}) {
	ctx := context.Background()
	// The parameter test is used here because mocking makes using real parameters unnecessary.
	err := db.PutDdbObject(ctx, ddbMock, "test", expectedObject)
	if err != nil {
		t.Errorf("putDdbObject failed for type %T: %v", expectedObject, err)
	}

	lastPutItem := ddbMock.PutItemInputs[len(ddbMock.PutItemInputs)-1].Item

	expectedMap := make(map[string]any)
	lastPutMap := make(map[string]any)

	if err = attributevalue.UnmarshalMap(expectedDdbItem.Output.Item, &expectedMap); err != nil {
		t.Errorf("failed to unmarshal expectedDdbItem: %v", err)
	}

	if err = attributevalue.UnmarshalMap(lastPutItem, &lastPutMap); err != nil {
		t.Errorf("failed to unmarshal lastPutItem: %v", err)
	}

	if !reflect.DeepEqual(expectedMap, lastPutMap) {
		t.Errorf("expected map %v, got %v", expectedMap, lastPutMap)
	}
}
