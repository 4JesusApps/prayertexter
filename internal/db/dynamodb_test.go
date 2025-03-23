package db_test

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/mshort55/prayertexter/internal/db"
	"github.com/mshort55/prayertexter/internal/messaging"
	"github.com/mshort55/prayertexter/internal/mock"
	"github.com/mshort55/prayertexter/internal/object"
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
		// IntercessorPhones
		{
			Output: &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKey},
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
					"Request":          &types.AttributeValueMemberS{Value: "I need prayer for..."},
					"Requestor": &types.AttributeValueMemberM{
						Value: map[string]types.AttributeValue{
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
		// StateTracker
		{
			Output: &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"Key": &types.AttributeValueMemberS{Value: object.StateTrackerKey},
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
							&types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Error": &types.AttributeValueMemberS{Value: ""},
									"Message": &types.AttributeValueMemberM{
										Value: map[string]types.AttributeValue{
											"Body":  &types.AttributeValueMemberS{Value: "sample text message 2"},
											"Phone": &types.AttributeValueMemberS{Value: "+19987654321"},
										},
									},
									"ID":        &types.AttributeValueMemberS{Value: "19ee2955d41d08325e1a97cbba1e544b"},
									"Stage":     &types.AttributeValueMemberS{Value: "MEMBER DELETE"},
									"Status":    &types.AttributeValueMemberS{Value: "IN PROGRESS"},
									"TimeStart": &types.AttributeValueMemberS{Value: "2025-02-16T23:57:01Z"},
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
		&object.Member{
			Intercessor:       true,
			Name:              "Intercessor1",
			Phone:             "+11111111111",
			PrayerCount:       1,
			SetupStage:        object.MemberSignUpStepFinal,
			SetupStatus:       object.MemberSetupComplete,
			WeeklyPrayerDate:  "2025-02-16T23:54:01Z",
			WeeklyPrayerLimit: 5,
		},
		&object.IntercessorPhones{
			Key: object.IntercessorPhonesKey,
			Phones: []string{
				"+11111111111",
				"+12222222222",
			},
		},
		&object.Prayer{
			Intercessor: object.Member{
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
			Request:          "I need prayer for...",
			Requestor: object.Member{
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
		&object.StateTracker{
			Key: object.StateTrackerKey,
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
		},
	}

	t.Run("Get dynamodb object tests", func(t *testing.T) {
		ddbMock := &mock.DDBConnecter{}
		ddbMock.GetItemResults = expectedDdbItems

		t.Run("Get Member", func(t *testing.T) {
			obj := expectedObjects[0].(*object.Member)
			testGetObject(t, ddbMock, obj)
		})

		t.Run("Get IntercessorPhones", func(t *testing.T) {
			obj := expectedObjects[1].(*object.IntercessorPhones)
			testGetObject(t, ddbMock, obj)
		})

		t.Run("Get Prayer", func(t *testing.T) {
			obj := expectedObjects[2].(*object.Prayer)
			testGetObject(t, ddbMock, obj)
		})

		t.Run("Get StateTracker", func(t *testing.T) {
			obj := expectedObjects[3].(*object.StateTracker)
			testGetObject(t, ddbMock, obj)
		})
	})

	t.Run("Put dynamodb object tests", func(t *testing.T) {
		ddbMock := &mock.DDBConnecter{}

		t.Run("Put Member", func(t *testing.T) {
			obj := expectedObjects[0].(*object.Member)
			testPutObject(t, ddbMock, obj, expectedDdbItems[0])
		})

		t.Run("Put IntercessorPhones", func(t *testing.T) {
			obj := expectedObjects[1].(*object.IntercessorPhones)
			testPutObject(t, ddbMock, obj, expectedDdbItems[1])
		})

		t.Run("Put Prayer", func(t *testing.T) {
			obj := expectedObjects[2].(*object.Prayer)
			testPutObject(t, ddbMock, obj, expectedDdbItems[2])
		})

		t.Run("Put StateTracker", func(t *testing.T) {
			obj := expectedObjects[3].(*object.StateTracker)
			testPutObject(t, ddbMock, obj, expectedDdbItems[3])
		})
	})
}

func testGetObject[T any](t *testing.T, ddbMock db.DDBConnecter, expectedObject *T) {
	// using test test test here because get ddb function is mocked so parameters are irrelevant
	testedObject, err := db.GetDdbObject[T](ddbMock, "test", "test", "test")
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
	// using test here because put ddb function is mocked so this parameter is irrelevant
	err := db.PutDdbObject(ddbMock, "test", expectedObject)
	if err != nil {
		t.Errorf("putDdbObject failed for type %T: %v", expectedObject, err)
	}

	lastPutItem := ddbMock.PutItemInputs[len(ddbMock.PutItemInputs)-1].Item

	expectedMap := make(map[string]interface{})
	lastPutMap := make(map[string]interface{})

	if err := attributevalue.UnmarshalMap(expectedDdbItem.Output.Item, &expectedMap); err != nil {
		t.Errorf("failed to unmarshal expectedDdbItem: %v", err)
	}

	if err := attributevalue.UnmarshalMap(lastPutItem, &lastPutMap); err != nil {
		t.Errorf("failed to unmarshal lastPutItem: %v", err)
	}

	if !reflect.DeepEqual(expectedMap, lastPutMap) {
		t.Errorf("expected map %v, got %v", expectedMap, lastPutMap)
	}
}