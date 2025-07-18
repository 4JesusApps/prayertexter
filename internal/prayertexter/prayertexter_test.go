package prayertexter_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/prayertexter"
	"github.com/4JesusApps/prayertexter/internal/test"
	"github.com/4JesusApps/prayertexter/internal/test/mock"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestMainFlowBlockUser(t *testing.T) {
	testCases := []test.Case{
		{
			Description: "Someone tries to block a user but is not an admin",

			InitialMessage: messaging.TextMessage{
				Body:  "#BLOCK 777-777-7777",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgUnauthorized,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Blocking a user is attempted, but phone number is invalid",

			InitialMessage: messaging.TextMessage{
				Body:  "#block 123",
				Phone: "+17777777777",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Administrator": &types.AttributeValueMemberBOOL{Value: true},
							"Name":          &types.AttributeValueMemberS{Value: "Admin User"},
							"Phone":         &types.AttributeValueMemberS{Value: "+17777777777"},
							"SetupStage":    &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":   &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgInvalidPhone,
					Phone: "+17777777777",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Blocking a user is attempted, but user is already blocked",

			InitialMessage: messaging.TextMessage{
				Body:  "#block 123-456-7890",
				Phone: "+17777777777",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Administrator": &types.AttributeValueMemberBOOL{Value: true},
							"Name":          &types.AttributeValueMemberS{Value: "Admin User"},
							"Phone":         &types.AttributeValueMemberS{Value: "+17777777777"},
							"SetupStage":    &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":   &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.BlockedPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11234567890"},
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
							}},
						},
					},
					Error: nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgUserAlreadyBlocked,
					Phone: "+17777777777",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Admin successfully blocks a user",

			InitialMessage: messaging.TextMessage{
				Body:  "#block 123-456-7890",
				Phone: "+17777777777",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Administrator": &types.AttributeValueMemberBOOL{Value: true},
							"Name":          &types.AttributeValueMemberS{Value: "Admin User"},
							"Phone":         &types.AttributeValueMemberS{Value: "+17777777777"},
							"SetupStage":    &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":   &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.BlockedPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
							}},
						},
					},
					Error: nil,
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   "+11234567890",
					Table: object.DefaultMemberTable,
				},
			},

			ExpectedBlockPhones: object.BlockedPhones{
				Key: object.BlockedPhonesKeyValue,
				Phones: []string{
					"+12222222222",
					"+13333333333",
					"+11234567890",
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: "+11234567890",
				},
				{
					Body:  messaging.MsgBlockedNotification + messaging.MsgHelp,
					Phone: "+11234567890",
				},
				{
					Body:  messaging.MsgSuccessfullyBlocked,
					Phone: "+17777777777",
				},
			},

			ExpectedGetItemCalls:    3,
			ExpectedPutItemCalls:    1,
			ExpectedDeleteItemCalls: 1,
			ExpectedSendTextCalls:   3,
		},
		{
			Description: "Admin successfully blocks a user that is an intercessor with an active prayer",

			InitialMessage: messaging.TextMessage{
				Body:  "#block 123-456-7890",
				Phone: "+17777777777",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{

				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Administrator": &types.AttributeValueMemberBOOL{Value: true},
							"Name":          &types.AttributeValueMemberS{Value: "Admin User"},
							"Phone":         &types.AttributeValueMemberS{Value: "+17777777777"},
							"SetupStage":    &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":   &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.BlockedPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
							}},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Bad User"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11234567890"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11234567890"},
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+14444444444"},
								&types.AttributeValueMemberS{Value: "+17777777777"},
							}},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
									"Name":              &types.AttributeValueMemberS{Value: "Bad User"},
									"Phone":             &types.AttributeValueMemberS{Value: "+11234567890"},
									"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
									"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
									"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
								},
							},
							"IntercessorPhone": &types.AttributeValueMemberS{Value: "+11234567890"},
							"Request":          &types.AttributeValueMemberS{Value: "Please pray me.."},
							"Requestor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
									"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
									"Phone":       &types.AttributeValueMemberS{Value: "+18888888888"},
									"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
								},
							},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
									"Name":              &types.AttributeValueMemberS{Value: "Bad User"},
									"Phone":             &types.AttributeValueMemberS{Value: "+11234567890"},
									"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
									"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
									"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
								},
							},
							"IntercessorPhone": &types.AttributeValueMemberS{Value: "+11234567890"},
							"Request":          &types.AttributeValueMemberS{Value: "Please pray me.."},
							"Requestor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
									"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
									"Phone":       &types.AttributeValueMemberS{Value: "+18888888888"},
									"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
								},
							},
						},
					},
					Error: nil,
				},
			},

			ExpectedIntPhones: object.IntercessorPhones{
				Key: object.IntercessorPhonesKeyValue,
				Phones: []string{
					"+11111111111",
					"+14444444444",
					"+17777777777",
				},
			},

			ExpectedPrayers: []object.Prayer{
				{
					Intercessor:      object.Member{},
					IntercessorPhone: "dummy ID",
					Request:          "Please pray me..",
					Requestor: object.Member{
						Intercessor: false,
						Name:        "John Doe",
						Phone:       "+18888888888",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   "+11234567890",
					Table: object.DefaultMemberTable,
				},
				{
					Key:   "+11234567890",
					Table: object.DefaultActivePrayersTable,
				},
			},

			ExpectedBlockPhones: object.BlockedPhones{
				Key: object.BlockedPhonesKeyValue,
				Phones: []string{
					"+12222222222",
					"+13333333333",
					"+11234567890",
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: "+11234567890",
				},
				{
					Body:  messaging.MsgBlockedNotification + messaging.MsgHelp,
					Phone: "+11234567890",
				},
				{
					Body:  messaging.MsgSuccessfullyBlocked,
					Phone: "+17777777777",
				},
			},

			ExpectedGetItemCalls:    6,
			ExpectedPutItemCalls:    3,
			ExpectedDeleteItemCalls: 2,
			ExpectedSendTextCalls:   3,
			ExpectedPrayerQueue:     true,
		},
		{
			Description: "Blocked phone number sends in a message and message gets dropped",

			InitialMessage: messaging.TextMessage{
				Body:  "random text",
				Phone: "+19999999999",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// Member empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.BlockedPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
								&types.AttributeValueMemberS{Value: "+14444444444"},
								&types.AttributeValueMemberS{Value: "+19999999999"},
							}},
						},
					},
					Error: nil,
				},
			},

			ExpectedGetItemCalls: 2,
		},
	}

	for _, tc := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}
		ctx := context.Background()

		t.Run(tc.Description, func(t *testing.T) {
			test.SetMocks(ddbMock, txtMock, tc)

			if tc.ExpectedError {
				// Handles failures for error mocks.
				if err := prayertexter.MainFlow(ctx, ddbMock, txtMock, tc.InitialMessage); err == nil {
					t.Fatalf("expected error, got nil")
				}

				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases.
				if err := prayertexter.MainFlow(ctx, ddbMock, txtMock, tc.InitialMessage); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				test.RunAllCommonTests(ddbMock, txtMock, t, tc)
			}
		})
	}
}

func TestMainFlowSignUp(t *testing.T) {
	testCases := []test.Case{
		{
			Description: "Sign up stage ONE: user texts the word pray to start sign up process - this also tests that" +
				"invalid characters are removed",

			InitialMessage: messaging.TextMessage{
				Body:  "pray./,, ",
				Phone: "+11234567890",
			},

			ExpectedMembers: []object.Member{
				{
					Phone:       "+11234567890",
					SetupStage:  object.MemberSignUpStepOne,
					SetupStatus: object.MemberSetupInProgress,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgNameRequest,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedPutItemCalls:  1,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up stage ONE: user texts the word Pray (capitol P) to start sign up process",

			InitialMessage: messaging.TextMessage{
				Body:  "Pray",
				Phone: "+11234567890",
			},

			ExpectedMembers: []object.Member{
				{
					Phone:       "+11234567890",
					SetupStage:  object.MemberSignUpStepOne,
					SetupStatus: object.MemberSetupInProgress,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgNameRequest,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedPutItemCalls:  1,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up stage ONE: get Member error",

			InitialMessage: messaging.TextMessage{
				Body:  "pray",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: nil,
					Error:  errors.New("first get item failure"),
				},
			},

			ExpectedError:        true,
			ExpectedGetItemCalls: 1,
		},
		{
			Description: "Sign up stage TWO-A: user texts name",

			InitialMessage: messaging.TextMessage{
				Body:  "John Doe",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepOne)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupInProgress},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Name:        "John Doe",
					Phone:       "+11234567890",
					SetupStage:  object.MemberSignUpStepTwo,
					SetupStatus: object.MemberSetupInProgress,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgMemberTypeRequest,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedPutItemCalls:  1,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up stage TWO-A: user texts name with profanity which should get blocked",

			InitialMessage: messaging.TextMessage{
				Body:  "Bastard",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepOne)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupInProgress},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgProfanityDetected,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up stage TWO-A: user texts invalid name, less than 2 letters",

			InitialMessage: messaging.TextMessage{
				Body:  "A",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepOne)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupInProgress},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgInvalidName,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up stage TWO-A: user texts invalid name, contains non letters",

			InitialMessage: messaging.TextMessage{
				Body:  "Dude!",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepOne)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupInProgress},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgInvalidName,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up stage TWO-B: user texts 2 to remain anonymous",

			InitialMessage: messaging.TextMessage{
				Body:  "2",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepOne)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupInProgress},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Name:        "Anonymous",
					Phone:       "+11234567890",
					SetupStage:  object.MemberSignUpStepTwo,
					SetupStatus: object.MemberSetupInProgress,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgMemberTypeRequest,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedPutItemCalls:  1,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up final prayer message: user texts 1 which means they do not want to be an " +
				"intercessor - this also tests that invalid characters are removed",

			InitialMessage: messaging.TextMessage{
				Body:  "1 !@#$%^&*() \n \n ",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepTwo)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupInProgress},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Intercessor: false,
					Name:        "John Doe",
					Phone:       "+11234567890",
					SetupStage:  object.MemberSignUpStepFinal,
					SetupStatus: object.MemberSetupComplete,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgSignUpConfirmation,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedPutItemCalls:  1,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up stage THREE: user texts 2 which means they want to be an intercessor",

			InitialMessage: messaging.TextMessage{
				Body:  "2",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepTwo)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupInProgress},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Intercessor: true,
					Name:        "John Doe",
					Phone:       "+11234567890",
					SetupStage:  object.MemberSignUpStepThree,
					SetupStatus: object.MemberSetupInProgress,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerNumRequest,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedPutItemCalls:  1,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up final intercessor message: user texts the number of prayers they are willing to " +
				"receive per week - this also tests that commas are removed from msg body",

			InitialMessage: messaging.TextMessage{
				Body:  "9,999",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepThree)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupInProgress},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
							}},
						},
					},
					Error: nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Intercessor:       true,
					Name:              "John Doe",
					Phone:             "+11234567890",
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 9999,
				},
			},

			ExpectedIntPhones: object.IntercessorPhones{
				Key: object.IntercessorPhonesKeyValue,
				Phones: []string{
					"+11111111111",
					"+12222222222",
					"+13333333333",
					"+11234567890",
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body: messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgIntercessorInstructions + "\n\n" +
						messaging.MsgSignUpConfirmation,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  3,
			ExpectedPutItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up final intercessor message: put IntercessorPhones error",

			InitialMessage: messaging.TextMessage{
				Body:  "10",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepThree)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupInProgress},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
							}},
						},
					},
					Error: nil,
				},
			},

			MockPutItemResults: []struct {
				Error error
			}{
				{
					Error: errors.New("third put item failure"),
				},
			},

			ExpectedError:        true,
			ExpectedGetItemCalls: 3,
			ExpectedPutItemCalls: 1,
		},
	}

	for _, tc := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}
		ctx := context.Background()

		t.Run(tc.Description, func(t *testing.T) {
			test.SetMocks(ddbMock, txtMock, tc)

			if tc.ExpectedError {
				// Handles failures for error mocks.
				if err := prayertexter.MainFlow(ctx, ddbMock, txtMock, tc.InitialMessage); err == nil {
					t.Fatalf("expected error, got nil")
				}

				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases.
				if err := prayertexter.MainFlow(ctx, ddbMock, txtMock, tc.InitialMessage); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				test.RunAllCommonTests(ddbMock, txtMock, t, tc)
			}
		})
	}
}

func TestMainFlowSignUpWrongInputs(t *testing.T) {
	testCases := []test.Case{
		{
			Description: "pray misspelled - returns non registered user and exits",

			InitialMessage: messaging.TextMessage{
				Body:  "prayyy",
				Phone: "+11234567890",
			},

			ExpectedGetItemCalls: 2,
		},
		{
			Description: "Sign up stage THREE: did not send 1 or 2 as expected to answer MsgMemberTypeRequest",

			InitialMessage: messaging.TextMessage{
				Body:  "wrong response to question",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepTwo)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupInProgress},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgWrongInput,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up final intercessor message: did not send number as expected",

			InitialMessage: messaging.TextMessage{
				Body:  "wrong response to question",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepThree)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupInProgress},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgWrongInput,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
	}

	for _, tc := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}
		ctx := context.Background()

		t.Run(tc.Description, func(t *testing.T) {
			test.SetMocks(ddbMock, txtMock, tc)

			if err := prayertexter.MainFlow(ctx, ddbMock, txtMock, tc.InitialMessage); err != nil {
				t.Fatalf("unexpected error starting MainFlow: %v", err)
			}

			test.RunAllCommonTests(ddbMock, txtMock, t, tc)
		})
	}
}

func TestMainFlowMemberDelete(t *testing.T) {
	testCases := []test.Case{
		{
			Description: "Delete non intercessor member with cancel txt - phone list stays the same",

			InitialMessage: messaging.TextMessage{
				Body:  "cancel",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   "+11234567890",
					Table: object.DefaultMemberTable,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:    2,
			ExpectedDeleteItemCalls: 1,
			ExpectedSendTextCalls:   1,
		},
		{
			Description: "Delete intercessor member with STOP txt - phone list changes",

			InitialMessage: messaging.TextMessage{
				Body:  "STOP",
				Phone: "+14444444444",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor4"},
							"Phone":             &types.AttributeValueMemberS{Value: "+14444444444"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
								&types.AttributeValueMemberS{Value: "+14444444444"},
							}},
						},
					},
					Error: nil,
				},
			},

			ExpectedIntPhones: object.IntercessorPhones{
				Key: object.IntercessorPhonesKeyValue,
				Phones: []string{
					"+11111111111",
					"+12222222222",
					"+13333333333",
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   "+14444444444",
					Table: object.DefaultMemberTable,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: "+14444444444",
				},
			},

			ExpectedGetItemCalls:    4,
			ExpectedPutItemCalls:    1,
			ExpectedDeleteItemCalls: 1,
			ExpectedSendTextCalls:   1,
		},
		{
			Description: "Delete intercessor member with STOP txt - phone list changes, active prayer gets moved to" +
				"prayer queue",

			InitialMessage: messaging.TextMessage{
				Body:  "STOP",
				Phone: "+14444444444",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor4"},
							"Phone":             &types.AttributeValueMemberS{Value: "+14444444444"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
								&types.AttributeValueMemberS{Value: "+14444444444"},
							}},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
									"Name":              &types.AttributeValueMemberS{Value: "Intercessor4"},
									"Phone":             &types.AttributeValueMemberS{Value: "+14444444444"},
									"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
									"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
									"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
								},
							},
							"IntercessorPhone": &types.AttributeValueMemberS{Value: "+14444444444"},
							"Request":          &types.AttributeValueMemberS{Value: "Please pray me.."},
							"Requestor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
									"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
									"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
									"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
								},
							},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
									"Name":              &types.AttributeValueMemberS{Value: "Intercessor4"},
									"Phone":             &types.AttributeValueMemberS{Value: "+14444444444"},
									"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
									"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
									"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
								},
							},
							"IntercessorPhone": &types.AttributeValueMemberS{Value: "+14444444444"},
							"Request":          &types.AttributeValueMemberS{Value: "Please pray me.."},
							"Requestor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
									"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
									"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
									"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
								},
							},
						},
					},
					Error: nil,
				},
			},

			ExpectedIntPhones: object.IntercessorPhones{
				Key: object.IntercessorPhonesKeyValue,
				Phones: []string{
					"+11111111111",
					"+12222222222",
					"+13333333333",
				},
			},

			ExpectedPrayers: []object.Prayer{
				{
					Intercessor:      object.Member{},
					IntercessorPhone: "dummy ID",
					Request:          "Please pray me..",
					Requestor: object.Member{
						Intercessor: false,
						Name:        "John Doe",
						Phone:       "+11234567890",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   "+14444444444",
					Table: object.DefaultMemberTable,
				},
				{
					Key:   "+14444444444",
					Table: object.DefaultActivePrayersTable,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: "+14444444444",
				},
			},

			ExpectedGetItemCalls:    5,
			ExpectedPutItemCalls:    2,
			ExpectedDeleteItemCalls: 2,
			ExpectedSendTextCalls:   1,
			ExpectedPrayerQueue:     true,
		},
		{
			Description: "Delete member - expected error on DelItem",

			InitialMessage: messaging.TextMessage{
				Body:  "cancel",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			MockDeleteItemResults: []struct {
				Error error
			}{
				{
					Error: errors.New("delete item failure"),
				},
			},

			ExpectedError:           true,
			ExpectedGetItemCalls:    2,
			ExpectedDeleteItemCalls: 1,
		},
	}

	for _, tc := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}
		ctx := context.Background()

		t.Run(tc.Description, func(t *testing.T) {
			test.SetMocks(ddbMock, txtMock, tc)

			if tc.ExpectedError {
				// Handles failures for error mocks.
				if err := prayertexter.MainFlow(ctx, ddbMock, txtMock, tc.InitialMessage); err == nil {
					t.Fatalf("expected error, got nil")
				}

				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases.
				if err := prayertexter.MainFlow(ctx, ddbMock, txtMock, tc.InitialMessage); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				test.RunAllCommonTests(ddbMock, txtMock, t, tc)
			}
		})
	}
}

func TestMainFlowHelp(t *testing.T) {
	testCases := []test.Case{
		{
			Description: "Setup stage 99 user texts help and receives the help message",

			InitialMessage: messaging.TextMessage{
				Body:  "help",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgHelp,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Setup stage 1 user texts help and receives the help message",

			InitialMessage: messaging.TextMessage{
				Body:  "help",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepOne)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupInProgress},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgHelp,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
	}

	for _, tc := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}
		ctx := context.Background()

		t.Run(tc.Description, func(t *testing.T) {
			test.SetMocks(ddbMock, txtMock, tc)

			if err := prayertexter.MainFlow(ctx, ddbMock, txtMock, tc.InitialMessage); err != nil {
				t.Fatalf("unexpected error starting MainFlow: %v", err)
			}

			test.RunAllCommonTests(ddbMock, txtMock, t, tc)
		})
	}
}

func TestMainFlowPrayerRequest(t *testing.T) {
	testCases := []test.Case{
		{
			Description: "Successful simple prayer request flow",

			InitialMessage: messaging.TextMessage{
				Body:  "I need prayer for these things...",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "+12222222222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             "+11111111111",
					PrayerCount:       1,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             "+12222222222",
					PrayerCount:       1,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedPrayers: []object.Prayer{
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             "+11111111111",
						PrayerCount:       1,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+11111111111",
					Request:          "I need prayer for these things...",
					Requestor: object.Member{
						Name:        "John Doe",
						Phone:       "+11234567890",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor2",
						Phone:             "+12222222222",
						PrayerCount:       1,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+12222222222",
					Request:          "I need prayer for these things...",
					Requestor: object.Member{
						Name:        "John Doe",
						Phone:       "+11234567890",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+11111111111",
				},
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+12222222222",
				},
				{
					Body:  messaging.MsgPrayerAssigned,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  7,
			ExpectedPutItemCalls:  4,
			ExpectedSendTextCalls: 3,
		},
		{
			Description: "Profanity detected in prayer request which should get blocked",

			InitialMessage: messaging.TextMessage{
				Body:  "sh!t",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgProfanityDetected,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Invalid prayer - not enough words",

			InitialMessage: messaging.TextMessage{
				Body:  "this is four words",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgInvalidRequest,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Error with first put Prayer in FindIntercessors",

			InitialMessage: messaging.TextMessage{
				Body:  "I need prayer for these things...",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "+12222222222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			MockPutItemResults: []struct {
				Error error
			}{
				{
					Error: nil,
				},
				{
					Error: nil,
				},
				{
					Error: errors.New("put item failure"),
				},
			},

			ExpectedGetItemCalls: 7,
			ExpectedPutItemCalls: 3,
			ExpectedError:        true,
		},
		{
			Description: "No available intercessors because of maxed out prayer counters",

			InitialMessage: messaging.TextMessage{
				Body:  "I need prayer for these things...",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "+12222222222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedPrayers: []object.Prayer{
				{
					IntercessorPhone: "dummy ID",
					Request:          "I need prayer for these things...",
					Requestor: object.Member{
						Name:        "John Doe",
						Phone:       "+11234567890",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerQueued,
					Phone: "+11234567890",
				},
			},

			ExpectedPrayerQueue:   true,
			ExpectedGetItemCalls:  7,
			ExpectedPutItemCalls:  1,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Anonymous prayer request using #anon prefix",

			InitialMessage: messaging.TextMessage{
				Body:  "#anon I need prayer for these things...",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "+12222222222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             "+11111111111",
					PrayerCount:       1,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             "+12222222222",
					PrayerCount:       1,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedPrayers: []object.Prayer{
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             "+11111111111",
						PrayerCount:       1,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+11111111111",
					Request:          "I need prayer for these things...",
					Requestor: object.Member{
						Name:        "Anonymous",
						Phone:       "+11234567890",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor2",
						Phone:             "+12222222222",
						PrayerCount:       1,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+12222222222",
					Request:          "I need prayer for these things...",
					Requestor: object.Member{
						Name:        "Anonymous",
						Phone:       "+11234567890",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+11111111111",
				},
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+12222222222",
				},
				{
					Body:  messaging.MsgPrayerAssigned,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:  7,
			ExpectedPutItemCalls:  4,
			ExpectedSendTextCalls: 3,
		},
		{
			Description: "Anonymous prayer request using #ANON anywhere in message",

			InitialMessage: messaging.TextMessage{
				Body:  "I need prayer for these things... #ANON please keep this private",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "+12222222222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             "+11111111111",
					PrayerCount:       1,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             "+12222222222",
					PrayerCount:       1,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedPrayers: []object.Prayer{
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             "+11111111111",
						PrayerCount:       1,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+11111111111",
					Request:          "I need prayer for these things...  please keep this private",
					Requestor: object.Member{
						Name:        "Anonymous",
						Phone:       "+11234567890",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor2",
						Phone:             "+12222222222",
						PrayerCount:       1,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+12222222222",
					Request:          "I need prayer for these things...  please keep this private",
					Requestor: object.Member{
						Name:        "Anonymous",
						Phone:       "+11234567890",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+11111111111",
				},
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+12222222222",
				},
				{
					Body:  messaging.MsgPrayerAssigned,
					Phone: "+11234567890",
				},
			},

			ExpectedExactMessageMatch: []struct {
				Index   int
				Message string
			}{
				{
					Index: 0,
					Message: `PrayerTexter: Hello! Please pray for Anonymous:

I need prayer for these things...  please keep this private

Once you have prayed, reply with the word prayed so that the prayer can be confirmed.

Reply HELP for help or STOP to cancel.`,
				},
			},

			ExpectedGetItemCalls:  7,
			ExpectedPutItemCalls:  4,
			ExpectedSendTextCalls: 3,
		},
	}

	for _, tc := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}
		ctx := context.Background()

		t.Run(tc.Description, func(t *testing.T) {
			test.SetMocks(ddbMock, txtMock, tc)

			if tc.ExpectedError {
				// handles failures for error mocks
				if err := prayertexter.MainFlow(ctx, ddbMock, txtMock, tc.InitialMessage); err == nil {
					t.Fatalf("expected error, got nil")
				}

				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// handles success test cases
				if err := prayertexter.MainFlow(ctx, ddbMock, txtMock, tc.InitialMessage); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				test.RunAllCommonTests(ddbMock, txtMock, t, tc)
			}
		})
	}
}

func TestFindIntercessors(t *testing.T) {
	testCases := []test.Case{
		{
			Description: "This should pick #3 and #5 intercessors based on prayer counts/dates",

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
								&types.AttributeValueMemberS{Value: "+14444444444"},
								&types.AttributeValueMemberS{Value: "+15555555555"},
							}},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
							"Name":        &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":       &types.AttributeValueMemberS{Value: "+12222222222"},
							"PrayerCount": &types.AttributeValueMemberN{Value: "100"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate": &types.AttributeValueMemberS{Value: time.Now().AddDate(
								0, 0, -2).Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "100"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
							"Name":        &types.AttributeValueMemberS{Value: "Intercessor3"},
							"Phone":       &types.AttributeValueMemberS{Value: "+13333333333"},
							"PrayerCount": &types.AttributeValueMemberN{Value: "15"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate": &types.AttributeValueMemberS{Value: time.Now().AddDate(
								0, 0, -8).Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "15"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
							"Name":        &types.AttributeValueMemberS{Value: "Intercessor4"},
							"Phone":       &types.AttributeValueMemberS{Value: "+14444444444"},
							"PrayerCount": &types.AttributeValueMemberN{Value: "9"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate": &types.AttributeValueMemberS{Value: time.Now().AddDate(
								0, 0, -6).Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "9"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor5"},
							"Phone":             &types.AttributeValueMemberS{Value: "+15555555555"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "4"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             "+13333333333",
					PrayerCount:       1,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 15,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor5",
					Phone:             "+15555555555",
					PrayerCount:       5,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedGetItemCalls: 11,
			ExpectedPutItemCalls: 2,
		},
		{
			Description: "This should return a single intercessor because only one does not have maxed out prayers",

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
							}},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "+12222222222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor3"},
							"Phone":             &types.AttributeValueMemberS{Value: "+13333333333"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "4"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             "+13333333333",
					PrayerCount:       5,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedGetItemCalls: 7,
			ExpectedPutItemCalls: 1,
		},
		{
			Description: "This should return a single intercessor because the other intercessor (888-888-8888) gets" +
				"removed. In a real situation, this would be because they are the ones who sent in the prayer request.",
			// FindIntercessors has a parameter for skipping a phone number. We are using 888-888-8888 for this, which
			// is set permanently in the main testing logic for this section.

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+18888888888"},
							}},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             "+11111111111",
					PrayerCount:       2,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedGetItemCalls: 3,
			ExpectedPutItemCalls: 1,
		},
		{
			Description: "This should return the error NoAvailableIntercessors because all intercessors are maxed " +
				"out on prayer requests",

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
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
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "+12222222222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedMembers: nil,

			ExpectedGetItemCalls: 5,
		},
		{
			Description: "This should return a single intercessor because, while they all are not maxed out on" +
				"prayers, 2 of them already have active prayers",

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
							}},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
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
									"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
									"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
								},
							},
							"IntercessorPhone": &types.AttributeValueMemberS{Value: "+11111111111"},
							"Request":          &types.AttributeValueMemberS{Value: "Please pray me.."},
							"Requestor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
									"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
									"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
									"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
								},
							},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "+12222222222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor3"},
							"Phone":             &types.AttributeValueMemberS{Value: "+13333333333"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
									"Name":              &types.AttributeValueMemberS{Value: "Intercessor3"},
									"Phone":             &types.AttributeValueMemberS{Value: "+13333333333"},
									"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
									"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
									"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
								},
							},
							"IntercessorPhone": &types.AttributeValueMemberS{Value: "+13333333333"},
							"Request":          &types.AttributeValueMemberS{Value: "Please pray me.."},
							"Requestor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
									"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
									"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
									"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
								},
							},
						},
					},
					Error: nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             "+12222222222",
					PrayerCount:       2,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedGetItemCalls: 7,
			ExpectedPutItemCalls: 1,
		},
		{
			Description: "This should return 2 intercessors even when GenRandPhones returns 3 available intercessors",

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
								&types.AttributeValueMemberS{Value: "+14444444444"},
							}},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "+12222222222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor3"},
							"Phone":             &types.AttributeValueMemberS{Value: "+13333333333"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor4"},
							"Phone":             &types.AttributeValueMemberS{Value: "+14444444444"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// Prayer empty get response because there are no active prayers for this intercessor.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedMembers: []object.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             "+11111111111",
					PrayerCount:       2,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             "+13333333333",
					PrayerCount:       2,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedGetItemCalls: 7,
			ExpectedPutItemCalls: 2,
		},
	}

	for _, tc := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}
		ctx := context.Background()

		t.Run(tc.Description, func(t *testing.T) {
			test.SetMocks(ddbMock, txtMock, tc)

			if tc.ExpectedError {
				// Handles failures for error mocks.
				if _, err := prayertexter.FindIntercessors(ctx, ddbMock, "+18888888888"); err == nil {
					t.Fatalf("expected error, got nil")
				}

				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases.
				_, err := prayertexter.FindIntercessors(ctx, ddbMock, "+18888888888")
				if err != nil && !errors.Is(err, utility.ErrNoAvailableIntercessors) {
					// NoAvailableIntercessors is an expected errors that can occur with FindIntercessors. This
					// error should be handled accordingly by the caller. Since this is expected, it is included
					// here in the success test cases instead of the error cases.
					t.Fatalf("unexpected error starting FindIntercessors: %v", err)
				}

				test.RunAllCommonTests(ddbMock, txtMock, t, tc)
			}
		})
	}
}

func TestMainFlowCompletePrayer(t *testing.T) {
	testCases := []test.Case{
		{
			Description: "Successful prayer request completion with capitol letters and spaces/new lines - this also" +
				"tests that invalid characters are removed",

			InitialMessage: messaging.TextMessage{
				Body:  "prAyEd   \n",
				Phone: "+11111111111",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
									"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
									"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
								},
							},
							"IntercessorPhone": &types.AttributeValueMemberS{Value: "+11111111111"},
							"Request":          &types.AttributeValueMemberS{Value: "Please pray me.."},
							"Requestor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
									"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
									"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
									"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
								},
							},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   "+11111111111",
					Table: object.DefaultActivePrayersTable,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerThankYou,
					Phone: "+11111111111",
				},
				{
					Body:  messaging.MsgPrayerConfirmation,
					Phone: "+11234567890",
				},
			},

			ExpectedGetItemCalls:    4,
			ExpectedDeleteItemCalls: 1,
			ExpectedSendTextCalls:   2,
		},
		{
			Description: "Successful prayer request completion - skip sending prayer confirmation text to prayer" +
				"requestor because they are no longer a member",

			InitialMessage: messaging.TextMessage{
				Body:  "prayed",
				Phone: "+11111111111",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
									"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
									"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
								},
							},
							"IntercessorPhone": &types.AttributeValueMemberS{Value: "+11111111111"},
							"Request":          &types.AttributeValueMemberS{Value: "Please pray me.."},
							"Requestor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
									"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
									"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
									"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
								},
							},
						},
					},
					Error: nil,
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   "+11111111111",
					Table: object.DefaultActivePrayersTable,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerThankYou,
					Phone: "+11111111111",
				},
			},

			ExpectedGetItemCalls:    4,
			ExpectedDeleteItemCalls: 1,
			ExpectedSendTextCalls:   1,
		},
		{
			Description: "No active prayers to mark as prayed",

			InitialMessage: messaging.TextMessage{
				Body:  "prayed",
				Phone: "+11111111111",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgNoActivePrayer,
					Phone: "+11111111111",
				},
			},

			ExpectedGetItemCalls:  3,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Error with delete Prayer",

			InitialMessage: messaging.TextMessage{
				Body:  "prayed",
				Phone: "+11234567890",
			},

			MockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11111111111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// BlockedPhones empty get response.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
									"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
									"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
								},
							},
							"IntercessorPhone": &types.AttributeValueMemberS{Value: "+11111111111"},
							"Request":          &types.AttributeValueMemberS{Value: "Please pray me.."},
							"Requestor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
									"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
									"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
									"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
								},
							},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
						},
					},
					Error: nil,
				},
			},

			MockDeleteItemResults: []struct {
				Error error
			}{
				{
					Error: errors.New("delete item failure"),
				},
			},

			ExpectedError:           true,
			ExpectedGetItemCalls:    4,
			ExpectedDeleteItemCalls: 1,
			ExpectedSendTextCalls:   2,
		},
	}

	for _, tc := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}
		ctx := context.Background()

		t.Run(tc.Description, func(t *testing.T) {
			test.SetMocks(ddbMock, txtMock, tc)

			if tc.ExpectedError {
				// Handles failures for error mocks
				if err := prayertexter.MainFlow(ctx, ddbMock, txtMock, tc.InitialMessage); err == nil {
					t.Fatalf("expected error, got nil")
				}

				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases
				if err := prayertexter.MainFlow(ctx, ddbMock, txtMock, tc.InitialMessage); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				test.RunAllCommonTests(ddbMock, txtMock, t, tc)
			}
		})
	}
}
