package db_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/model"
	"github.com/4JesusApps/prayertexter/internal/test/mock"
	"github.com/4JesusApps/prayertexter/internal/test/testutil"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

func TestDynamoDBOperations(t *testing.T) {
	expectedDdbItems := []testutil.GetItemResult{
		testutil.MemberItem(model.Member{
			Intercessor:       true,
			Name:              "Intercessor1",
			Phone:             "+11111111111",
			PrayerCount:       1,
			SetupStage:        model.SignUpStepFinal,
			SetupStatus:       model.SetupComplete,
			WeeklyPrayerDate:  "2025-02-16T23:54:01Z",
			WeeklyPrayerLimit: 5,
		}),
		testutil.BlockedPhonesItem("+13333333333", "+14444444444"),
		testutil.IntercessorPhonesItem("+11111111111", "+12222222222"),
		testutil.PrayerItem(model.Prayer{
			Intercessor: model.Member{
				Intercessor:       true,
				Name:              "Intercessor1",
				Phone:             "+11111111111",
				PrayerCount:       1,
				SetupStage:        model.SignUpStepFinal,
				SetupStatus:       model.SetupComplete,
				WeeklyPrayerDate:  "2025-02-13T23:54:01Z",
				WeeklyPrayerLimit: 5,
			},
			IntercessorPhone: "+11111111111",
			ReminderCount:    3,
			ReminderDate:     "2025-02-13T23:54:01Z",
			Request:          "I need prayer for...",
			Requestor: model.Member{
				Name:        "John Doe",
				Phone:       "+11234567890",
				SetupStage:  model.SignUpStepFinal,
				SetupStatus: model.SetupComplete,
			},
		}),
	}

	expectedObjects := []any{
		&model.Member{
			Administrator:     false,
			Intercessor:       true,
			Name:              "Intercessor1",
			Phone:             "+11111111111",
			PrayerCount:       1,
			SetupStage:        model.SignUpStepFinal,
			SetupStatus:       model.SetupComplete,
			WeeklyPrayerDate:  "2025-02-16T23:54:01Z",
			WeeklyPrayerLimit: 5,
		},
		&model.BlockedPhones{
			Key: model.BlockedPhonesKeyValue,
			Phones: []string{
				"+13333333333",
				"+14444444444",
			},
		},
		&model.IntercessorPhones{
			Key: model.IntercessorPhonesKeyValue,
			Phones: []string{
				"+11111111111",
				"+12222222222",
			},
		},
		&model.Prayer{
			Intercessor: model.Member{
				Administrator:     false,
				Intercessor:       true,
				Name:              "Intercessor1",
				Phone:             "+11111111111",
				PrayerCount:       1,
				SetupStage:        model.SignUpStepFinal,
				SetupStatus:       model.SetupComplete,
				WeeklyPrayerDate:  "2025-02-13T23:54:01Z",
				WeeklyPrayerLimit: 5,
			},
			IntercessorPhone: "+11111111111",
			ReminderCount:    3,
			ReminderDate:     "2025-02-13T23:54:01Z",
			Request:          "I need prayer for...",
			Requestor: model.Member{
				Administrator:     false,
				Intercessor:       false,
				Name:              "John Doe",
				Phone:             "+11234567890",
				PrayerCount:       0,
				SetupStage:        model.SignUpStepFinal,
				SetupStatus:       model.SetupComplete,
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
				case *model.Member:
					testGetObject(t, ddbMock, o)
				case *model.BlockedPhones:
					testGetObject(t, ddbMock, o)
				case *model.IntercessorPhones:
					testGetObject(t, ddbMock, o)
				case *model.Prayer:
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
				case *model.Member:
					testPutObject(t, ddbMock, o, expectedDdbItems[index])
				case *model.BlockedPhones:
					testPutObject(t, ddbMock, o, expectedDdbItems[index])
				case *model.IntercessorPhones:
					testPutObject(t, ddbMock, o, expectedDdbItems[index])
				case *model.Prayer:
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

func testPutObject[T any](t *testing.T, ddbMock *mock.DDBConnecter, expectedObject *T, expectedDdbItem testutil.GetItemResult) {
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
