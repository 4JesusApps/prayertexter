package prayertexter_test

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/mshort55/prayertexter/internal/messaging"
	"github.com/mshort55/prayertexter/internal/mock"
	"github.com/mshort55/prayertexter/internal/object"
	"github.com/mshort55/prayertexter/internal/prayertexter"
	"github.com/mshort55/prayertexter/internal/utility"
)

type TestCase struct {
	description    string
	initialMessage messaging.TextMessage

	expectedGetItemCalls    int
	expectedPutItemCalls    int
	expectedDeleteItemCalls int
	expectedSendTextCalls   int

	expectedMembers     []object.Member
	expectedPrayers     []object.Prayer
	expectedTexts       []messaging.TextMessage
	expectedPhones      object.IntercessorPhones
	expectedError       bool
	expectedPrayerQueue bool

	expectedDeleteItems []struct {
		key   string
		table string
	}

	mockGetItemResults []struct {
		Output *dynamodb.GetItemOutput
		Error  error
	}
	mockPutItemResults []struct {
		Error error
	}
	mockDeleteItemResults []struct {
		Error error
	}
	mockSendTextResults []struct {
		Error error
	}
}

func setMocks(ddbMock *mock.DDBConnecter, txtMock *mock.TextSender, test TestCase) {
	ddbMock.GetItemResults = test.mockGetItemResults
	ddbMock.PutItemResults = test.mockPutItemResults
	ddbMock.DeleteItemResults = test.mockDeleteItemResults
	txtMock.SendTextResults = test.mockSendTextResults
}

func testNumMethodCalls(ddbMock *mock.DDBConnecter, txtMock *mock.TextSender, t *testing.T, test TestCase) {
	if ddbMock.GetItemCalls != test.expectedGetItemCalls {
		t.Errorf("expected GetItem to be called %v, got %v",
			test.expectedGetItemCalls, ddbMock.GetItemCalls)
	}

	if ddbMock.PutItemCalls != test.expectedPutItemCalls {
		t.Errorf("expected PutItem to be called %v, got %v",
			test.expectedPutItemCalls, ddbMock.PutItemCalls)
	}

	if ddbMock.DeleteItemCalls != test.expectedDeleteItemCalls {
		t.Errorf("expected DeleteItem to be called %v, got %v",
			test.expectedDeleteItemCalls, ddbMock.DeleteItemCalls)
	}

	if txtMock.SendTextCalls != test.expectedSendTextCalls {
		t.Errorf("expected SendText to be called %v, got %v",
			test.expectedSendTextCalls, txtMock.SendTextCalls)
	}
}

func testMembers(inputs []dynamodb.PutItemInput, t *testing.T, test TestCase) {
	index := 0

	for _, input := range inputs {
		if *input.TableName != object.MemberTable {
			continue
		}

		if index >= len(test.expectedMembers) {
			t.Errorf("there are more Members in put inputs than in expected Members")
		}

		var actualMem object.Member
		if err := attributevalue.UnmarshalMap(input.Item, &actualMem); err != nil {
			t.Errorf("failed to unmarshal PutItemInput into Member: %v", err)
		}

		// Replaces date to make mocking easier.
		if actualMem.WeeklyPrayerDate != "" {
			actualMem.WeeklyPrayerDate = "dummy date/time"
		}

		expectedMem := test.expectedMembers[index]
		if actualMem != expectedMem {
			t.Errorf("expected Member %v, got %v", expectedMem, actualMem)
		}

		index++
	}

	if index < len(test.expectedMembers) {
		t.Errorf("there are more Members in expected Members than in put inputs")
	}
}

func testPrayers(inputs []dynamodb.PutItemInput, t *testing.T, test TestCase, queue bool) {
	// We need to be careful here that inputs (Prayers) are not mixed with active and queued prayers, because this test
	// function cannot handle that.
	index := 0
	expectedTable := object.GetPrayerTable(queue)

	for _, input := range inputs {
		if *input.TableName != expectedTable {
			continue
		}

		if index >= len(test.expectedPrayers) {
			t.Errorf("there are more Prayers in put inputs than in expected Prayers of table type: %v", expectedTable)
		}

		var actualPryr object.Prayer
		if err := attributevalue.UnmarshalMap(input.Item, &actualPryr); err != nil {
			t.Errorf("failed to unmarshal PutItemInput into Prayer: %v", err)
		}

		// Replaces date and random ID to make mocking easier.
		if !queue {
			actualPryr.Intercessor.WeeklyPrayerDate = "dummy date/time"
		} else if queue {
			actualPryr.IntercessorPhone = "dummy ID"
		}

		expectedPryr := test.expectedPrayers[index]
		if actualPryr != expectedPryr {
			t.Errorf("expected Prayer %v, got %v", expectedPryr, actualPryr)
		}

		index++
	}

	if index < len(test.expectedPrayers) {
		t.Errorf("there are more Prayers in expected Prayers than in put inputs of table type: %v", expectedTable)
	}
}

func testPhones(inputs []dynamodb.PutItemInput, t *testing.T, test TestCase) {
	index := 0

	for _, input := range inputs {
		if *input.TableName != object.IntercessorPhonesTable {
			continue
		} else if val, ok := input.Item[object.IntercessorPhonesAttribute]; !ok {
			continue
		} else if stringVal, isString := val.(*types.AttributeValueMemberS); !isString {
			continue
		} else if stringVal.Value != object.IntercessorPhonesKey {
			continue
		}

		if index > 1 {
			t.Errorf("there are more IntercessorPhones in expected IntercessorPhones than 1 which is not expected")
		}

		var actualPhones object.IntercessorPhones
		if err := attributevalue.UnmarshalMap(input.Item, &actualPhones); err != nil {
			t.Errorf("failed to unmarshal PutItemInput into IntercessorPhones: %v", err)
		}

		if !reflect.DeepEqual(actualPhones, test.expectedPhones) {
			t.Errorf("expected IntercessorPhones %v, got %v", test.expectedPhones, actualPhones)
		}

		index++
	}
}

func testDeleteItem(inputs []dynamodb.DeleteItemInput, t *testing.T, test TestCase) {
	index := 0

	for _, input := range inputs {
		if index >= len(test.expectedDeleteItems) {
			t.Errorf("there are more delete item inputs than expected delete items")
		}

		switch *input.TableName {
		case object.MemberTable:
			testDeleteMember(input, &index, t, test)
		case object.ActivePrayersTable:
			testDeletePrayer(input, &index, t, test)
		default:
			t.Errorf("unexpected table name, got %v", *input.TableName)
		}
	}

	if index < len(test.expectedDeleteItems) {
		t.Errorf("there are more expected delete items than delete item inputs")
	}
}

func testDeleteMember(input dynamodb.DeleteItemInput, index *int, t *testing.T, test TestCase) {
	if *input.TableName != test.expectedDeleteItems[*index].table {
		t.Errorf("expected Member table %v, got %v",
			test.expectedDeleteItems[*index].table, *input.TableName)
	}

	mem := object.Member{}
	if err := attributevalue.UnmarshalMap(input.Key, &mem); err != nil {
		t.Fatalf("failed to unmarshal to Member: %v", err)
	}

	if mem.Phone != test.expectedDeleteItems[*index].key {
		t.Errorf("expected Member phone %v for delete key, got %v",
			test.expectedDeleteItems[*index].key, mem.Phone)
	}

	*index++
}

func testDeletePrayer(input dynamodb.DeleteItemInput, index *int, t *testing.T, test TestCase) {
	if *input.TableName != test.expectedDeleteItems[*index].table {
		t.Errorf("expected Prayer table %v, got %v",
			test.expectedDeleteItems[*index].table, *input.TableName)
	}

	pryr := object.Prayer{}
	if err := attributevalue.UnmarshalMap(input.Key, &pryr); err != nil {
		t.Fatalf("failed to unmarshal to Prayer: %v", err)
	}

	if pryr.IntercessorPhone != test.expectedDeleteItems[*index].key {
		t.Errorf("expected Prayer phone %v for delete key, got %v",
			test.expectedDeleteItems[*index].key, pryr.IntercessorPhone)
	}

	*index++
}

func testTxtMessage(txtMock *mock.TextSender, t *testing.T, test TestCase) {
	index := 0

	for _, input := range txtMock.SendTextInputs {
		if index >= len(test.expectedTexts) {
			t.Errorf("there are more text message inputs than expected texts")
		}

		// Some text messages use PLACEHOLDER and replace that with the txt recipients name. Therefor to make testing
		// easier, the message body is replaced by the msg constant.
		switch {
		case strings.Contains(*input.MessageBody, "Hello! Please pray for"):
			input.MessageBody = aws.String(messaging.MsgPrayerIntro)
		case strings.Contains(*input.MessageBody, "There was profanity found in your prayer request:"):
			input.MessageBody = aws.String(messaging.MsgProfanityFound)
		case strings.Contains(*input.MessageBody, "You're prayer request has been prayed for by"):
			input.MessageBody = aws.String(messaging.MsgPrayerConfirmation)
		}

		receivedText := messaging.TextMessage{
			Body:  *input.MessageBody,
			Phone: *input.DestinationPhoneNumber,
		}

		// This part makes mocking messages less painful. We do not need to worry about new lines, pre, or post
		// messages. They are removed when messages are tested.
		for _, t := range []*messaging.TextMessage{&receivedText, &test.expectedTexts[index]} {
			for _, str := range []string{"\n", messaging.MsgPre, messaging.MsgPost} {
				t.Body = strings.ReplaceAll(t.Body, str, "")
			}
		}

		if receivedText != test.expectedTexts[index] {
			t.Errorf("expected txt %v, got %v", test.expectedTexts[index], receivedText)
		}

		index++
	}

	if index < len(test.expectedTexts) {
		t.Errorf("there are more expected texts than text message inputs")
	}
}

func TestMainFlowSignUp(t *testing.T) {
	testCases := []TestCase{
		{
			description: "Sign up stage ONE: user texts the word pray to start sign up process",

			initialMessage: messaging.TextMessage{
				Body:  "pray",
				Phone: "+11234567890",
			},

			expectedMembers: []object.Member{
				{
					Phone:       "+11234567890",
					SetupStage:  object.MemberSignUpStepOne,
					SetupStatus: object.MemberSetupInProgress,
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgNameRequest,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  4,
			expectedPutItemCalls:  4,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up stage ONE: user texts the word Pray (capitol P) to start sign up process",

			initialMessage: messaging.TextMessage{
				Body:  "Pray",
				Phone: "+11234567890",
			},

			expectedMembers: []object.Member{
				{
					Phone:       "+11234567890",
					SetupStage:  object.MemberSignUpStepOne,
					SetupStatus: object.MemberSetupInProgress,
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgNameRequest,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  4,
			expectedPutItemCalls:  4,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up stage ONE: get Member error",

			initialMessage: messaging.TextMessage{
				Body:  "pray",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: nil,
					Error:  errors.New("first get item failure"),
				},
			},

			expectedError:        true,
			expectedGetItemCalls: 2,
			expectedPutItemCalls: 1,
		},
		{
			description: "Sign up stage TWO-A: user texts name",

			initialMessage: messaging.TextMessage{
				Body:  "John Doe",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
			},

			expectedMembers: []object.Member{
				{
					Name:        "John Doe",
					Phone:       "+11234567890",
					SetupStage:  object.MemberSignUpStepTwo,
					SetupStatus: object.MemberSetupInProgress,
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgMemberTypeRequest,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  4,
			expectedPutItemCalls:  4,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up stage TWO-B: user texts 2 to remain anonymous",

			initialMessage: messaging.TextMessage{
				Body:  "2",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
			},

			expectedMembers: []object.Member{
				{
					Name:        "Anonymous",
					Phone:       "+11234567890",
					SetupStage:  object.MemberSignUpStepTwo,
					SetupStatus: object.MemberSetupInProgress,
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgMemberTypeRequest,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  4,
			expectedPutItemCalls:  4,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up final prayer message: user texts 1 which means they do not want to be an intercessor",

			initialMessage: messaging.TextMessage{
				Body:  "1",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
			},

			expectedMembers: []object.Member{
				{
					Intercessor: false,
					Name:        "John Doe",
					Phone:       "+11234567890",
					SetupStage:  object.MemberSignUpStepFinal,
					SetupStatus: object.MemberSetupComplete,
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgSignUpConfirmation,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  4,
			expectedPutItemCalls:  4,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up stage THREE: user texts 2 which means they want to be an intercessor",

			initialMessage: messaging.TextMessage{
				Body:  "2",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
			},

			expectedMembers: []object.Member{
				{
					Intercessor: true,
					Name:        "John Doe",
					Phone:       "+11234567890",
					SetupStage:  object.MemberSignUpStepThree,
					SetupStatus: object.MemberSetupInProgress,
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerNumRequest,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  4,
			expectedPutItemCalls:  4,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up final intercessor message: user texts the number of prayers they are willing to" +
				"receive per week",

			initialMessage: messaging.TextMessage{
				Body:  "10",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKey},
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

			expectedMembers: []object.Member{
				{
					Intercessor:       true,
					Name:              "John Doe",
					Phone:             "+11234567890",
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 10,
				},
			},

			expectedPhones: object.IntercessorPhones{
				Key: object.IntercessorPhonesKey,
				Phones: []string{
					"+11111111111",
					"+12222222222",
					"+13333333333",
					"+11234567890",
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body: messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgIntercessorInstructions + "\n\n" +
						messaging.MsgSignUpConfirmation,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  5,
			expectedPutItemCalls:  5,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up final intercessor message: put IntercessorPhones error",

			initialMessage: messaging.TextMessage{
				Body:  "10",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKey},
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

			mockPutItemResults: []struct {
				Error error
			}{
				{
					Error: nil,
				},
				{
					Error: nil,
				},
				{
					Error: errors.New("third put item failure"),
				},
			},

			expectedError:        true,
			expectedGetItemCalls: 5,
			expectedPutItemCalls: 4,
		},
	}

	for _, test := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			setMocks(ddbMock, txtMock, test)

			if test.expectedError {
				// Handles failures for error mocks.
				if err := prayertexter.MainFlow(test.initialMessage, ddbMock, txtMock); err == nil {
					t.Fatalf("expected error, got nil")
				}
				testNumMethodCalls(ddbMock, txtMock, t, test)
			} else {
				// Handles success test cases.
				if err := prayertexter.MainFlow(test.initialMessage, ddbMock, txtMock); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				testNumMethodCalls(ddbMock, txtMock, t, test)
				testTxtMessage(txtMock, t, test)
				testMembers(ddbMock.PutItemInputs, t, test)
				testPhones(ddbMock.PutItemInputs, t, test)
			}
		})
	}
}

func TestMainFlowSignUpWrongInputs(t *testing.T) {
	testCases := []TestCase{
		{
			description: "pray misspelled - returns non registered user and exits",

			initialMessage: messaging.TextMessage{
				Body:  "prayyy",
				Phone: "+11234567890",
			},

			expectedGetItemCalls: 4,
			expectedPutItemCalls: 3,
		},
		{
			description: "Sign up stage THREE: did not send 1 or 2 as expected to answer MsgMemberTypeRequest",

			initialMessage: messaging.TextMessage{
				Body:  "wrong response to question",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgWrongInput,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  4,
			expectedPutItemCalls:  3,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up final intercessor message: did not send number as expected",

			initialMessage: messaging.TextMessage{
				Body:  "wrong response to question",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgWrongInput,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  4,
			expectedPutItemCalls:  3,
			expectedSendTextCalls: 1,
		},
	}

	for _, test := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			setMocks(ddbMock, txtMock, test)

			if err := prayertexter.MainFlow(test.initialMessage, ddbMock, txtMock); err != nil {
				t.Fatalf("unexpected error starting MainFlow: %v", err)
			}

			testNumMethodCalls(ddbMock, txtMock, t, test)
			testTxtMessage(txtMock, t, test)
		})
	}
}

func TestMainFlowMemberDelete(t *testing.T) {
	testCases := []TestCase{
		{
			description: "Delete non intercessor member with cancel txt - phone list stays the same",

			initialMessage: messaging.TextMessage{
				Body:  "cancel",
				Phone: "1234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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

			expectedDeleteItems: []struct {
				key   string
				table string
			}{
				{
					key:   "+11234567890",
					table: object.MemberTable,
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:    4,
			expectedPutItemCalls:    3,
			expectedDeleteItemCalls: 1,
			expectedSendTextCalls:   1,
		},
		{
			description: "Delete intercessor member with STOP txt - phone list changes",

			initialMessage: messaging.TextMessage{
				Body:  "STOP",
				Phone: "+14444444444",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKey},
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

			expectedPhones: object.IntercessorPhones{
				Key: object.IntercessorPhonesKey,
				Phones: []string{
					"+11111111111",
					"+12222222222",
					"+13333333333",
				},
			},

			expectedDeleteItems: []struct {
				key   string
				table string
			}{
				{
					key:   "+14444444444",
					table: object.MemberTable,
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: "+14444444444",
				},
			},

			expectedGetItemCalls:    6,
			expectedPutItemCalls:    4,
			expectedDeleteItemCalls: 1,
			expectedSendTextCalls:   1,
		},
		{
			description: "Delete intercessor member with STOP txt - phone list changes, active prayer gets moved to" +
				"prayer queue",

			initialMessage: messaging.TextMessage{
				Body:  "STOP",
				Phone: "+14444444444",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
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
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKey},
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

			expectedPhones: object.IntercessorPhones{
				Key: object.IntercessorPhonesKey,
				Phones: []string{
					"+11111111111",
					"+12222222222",
					"+13333333333",
				},
			},

			expectedPrayers: []object.Prayer{
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

			expectedDeleteItems: []struct {
				key   string
				table string
			}{
				{
					key:   "+14444444444",
					table: object.MemberTable,
				},
				{
					key:   "+14444444444",
					table: object.ActivePrayersTable,
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: "+14444444444",
				},
			},

			expectedGetItemCalls:    7,
			expectedPutItemCalls:    5,
			expectedDeleteItemCalls: 2,
			expectedSendTextCalls:   1,
		},
		{
			description: "Delete member - expected error on DelItem",

			initialMessage: messaging.TextMessage{
				Body:  "cancel",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
			},

			mockDeleteItemResults: []struct {
				Error error
			}{
				{
					Error: errors.New("delete item failure"),
				},
			},

			expectedError:           true,
			expectedGetItemCalls:    4,
			expectedPutItemCalls:    3,
			expectedDeleteItemCalls: 1,
		},
	}

	for _, test := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			setMocks(ddbMock, txtMock, test)

			if test.expectedError {
				// Handles failures for error mocks.
				if err := prayertexter.MainFlow(test.initialMessage, ddbMock, txtMock); err == nil {
					t.Fatalf("expected error, got nil")
				}
				testNumMethodCalls(ddbMock, txtMock, t, test)
			} else {
				// Handles success test cases.
				if err := prayertexter.MainFlow(test.initialMessage, ddbMock, txtMock); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				testNumMethodCalls(ddbMock, txtMock, t, test)
				testTxtMessage(txtMock, t, test)
				testPhones(ddbMock.PutItemInputs, t, test)
				testPrayers(ddbMock.PutItemInputs, t, test, true)
				testDeleteItem(ddbMock.DeleteItemInputs, t, test)
			}
		})
	}
}

func TestMainFlowHelp(t *testing.T) {
	testCases := []TestCase{
		{
			description: "Setup stage 99 user texts help and receives the help message",

			initialMessage: messaging.TextMessage{
				Body:  "help",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgHelp,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  4,
			expectedPutItemCalls:  3,
			expectedSendTextCalls: 1,
		},
		{
			description: "Setup stage 1 user texts help and receives the help message",

			initialMessage: messaging.TextMessage{
				Body:  "help",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgHelp,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  4,
			expectedPutItemCalls:  3,
			expectedSendTextCalls: 1,
		},
	}

	for _, test := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			setMocks(ddbMock, txtMock, test)

			if err := prayertexter.MainFlow(test.initialMessage, ddbMock, txtMock); err != nil {
				t.Fatalf("unexpected error starting MainFlow: %v", err)
			}

			testNumMethodCalls(ddbMock, txtMock, t, test)
			testTxtMessage(txtMock, t, test)
			testMembers(ddbMock.PutItemInputs, t, test)
			testPrayers(ddbMock.PutItemInputs, t, test, test.expectedPrayerQueue)
		})
	}
}

func TestMainFlowPrayerRequest(t *testing.T) {
	testCases := []TestCase{
		{
			description: "Successful simple prayer request flow",

			initialMessage: messaging.TextMessage{
				Body:  "I need prayer for...",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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

			expectedMembers: []object.Member{
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

			expectedPrayers: []object.Prayer{
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
					Request:          "I need prayer for...",
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
					Request:          "I need prayer for...",
					Requestor: object.Member{
						Name:        "John Doe",
						Phone:       "+11234567890",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+11111111111",
				},
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+12222222222",
				},
				{
					Body:  messaging.MsgPrayerSentOut,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  9,
			expectedPutItemCalls:  7,
			expectedSendTextCalls: 3,
		},
		{
			description: "Profanity detected",

			initialMessage: messaging.TextMessage{
				Body:  "sh!t",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgProfanityFound,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:  4,
			expectedPutItemCalls:  3,
			expectedSendTextCalls: 1,
		},
		{
			description: "Error with first put Prayer in FindIntercessors",

			initialMessage: messaging.TextMessage{
				Body:  "I need prayer for...",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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

			mockPutItemResults: []struct {
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

			expectedGetItemCalls: 7,
			expectedPutItemCalls: 4,
			expectedError:        true,
		},
		{
			description: "No available intercessors because of maxed out prayer counters",

			initialMessage: messaging.TextMessage{
				Body:  "I need prayer for...",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
				},
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

			expectedPrayers: []object.Prayer{
				{
					IntercessorPhone: "dummy ID",
					Request:          "I need prayer for...",
					Requestor: object.Member{
						Name:        "John Doe",
						Phone:       "+11234567890",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerQueued,
					Phone: "+11234567890",
				},
			},

			expectedPrayerQueue:   true,
			expectedGetItemCalls:  9,
			expectedPutItemCalls:  4,
			expectedSendTextCalls: 1,
		},
	}

	for _, test := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			setMocks(ddbMock, txtMock, test)

			if test.expectedError {
				// handles failures for error mocks
				if err := prayertexter.MainFlow(test.initialMessage, ddbMock, txtMock); err == nil {
					t.Fatalf("expected error, got nil")
				}
				testNumMethodCalls(ddbMock, txtMock, t, test)
			} else {
				// handles success test cases
				if err := prayertexter.MainFlow(test.initialMessage, ddbMock, txtMock); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				testNumMethodCalls(ddbMock, txtMock, t, test)
				testTxtMessage(txtMock, t, test)
				testMembers(ddbMock.PutItemInputs, t, test)
				testPrayers(ddbMock.PutItemInputs, t, test, test.expectedPrayerQueue)
			}
		})
	}
}

func TestFindIntercessors(t *testing.T) {
	testCases := []TestCase{
		{
			description: "This should pick #3 and #5 intercessors based on prayer counts/dates",

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKey},
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

			expectedMembers: []object.Member{
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

			expectedGetItemCalls: 11,
			expectedPutItemCalls: 2,
		},
		{
			description: "This should return a single intercessor because only one does not have maxed out prayers",

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKey},
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

			expectedMembers: []object.Member{
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

			expectedGetItemCalls: 7,
			expectedPutItemCalls: 1,
		},
		{
			description: "This should return a single intercessor because the other intercessor (888-888-8888) gets" +
				"removed. In a real situation, this would be because they are the ones who sent in the prayer request.",
			// FindIntercessors has a parameter for skipping a phone number. We are using 888-888-8888 for this, which
			// is set permanently in the main testing logic for this section.

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKey},
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

			expectedMembers: []object.Member{
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

			expectedGetItemCalls: 3,
			expectedPutItemCalls: 1,
		},
		{
			description: "This should return the error NoAvailableIntercessors because all intercessors are maxed " +
				"out on prayer requests",

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
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

			expectedMembers: nil,

			expectedGetItemCalls: 5,
		},
		{
			description: "This should return a single intercessor because, while they all are not maxed out on" +
				"prayers, 2 of them already have active prayers",

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKey},
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

			expectedMembers: []object.Member{
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

			expectedGetItemCalls: 7,
			expectedPutItemCalls: 1,
		},
	}

	for _, test := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			setMocks(ddbMock, txtMock, test)

			if test.expectedError {
				// Handles failures for error mocks.
				if _, err := prayertexter.FindIntercessors(ddbMock, "+18888888888"); err == nil {
					t.Fatalf("expected error, got nil")
				}
				testNumMethodCalls(ddbMock, txtMock, t, test)
			} else {
				// Handles success test cases.
				_, err := prayertexter.FindIntercessors(ddbMock, "+18888888888")
				if err != nil && !errors.Is(err, utility.ErrNoAvailableIntercessors) {
					// NoAvailableIntercessors is an expected errors that can occur with FindIntercessors. This
					// error should be handled accordingly by the caller. Since this is expected, it is included
					// here in the success test cases instead of the error cases.
					t.Fatalf("unexpected error starting FindIntercessors: %v", err)
				}

				testNumMethodCalls(ddbMock, txtMock, t, test)
				testMembers(ddbMock.PutItemInputs, t, test)
			}
		})
	}
}

func TestMainFlowCompletePrayer(t *testing.T) {
	testCases := []TestCase{
		{
			description: "Successful prayer request completion",

			initialMessage: messaging.TextMessage{
				Body:  "prayed",
				Phone: "+11111111111",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
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
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// StateTracker empty get response. It would over complicate to test this here.
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

			expectedDeleteItems: []struct {
				key   string
				table string
			}{
				{
					key:   "+11111111111",
					table: object.ActivePrayersTable,
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerThankYou,
					Phone: "+11111111111",
				},
				{
					Body:  messaging.MsgPrayerConfirmation,
					Phone: "+11234567890",
				},
			},

			expectedGetItemCalls:    6,
			expectedPutItemCalls:    3,
			expectedDeleteItemCalls: 1,
			expectedSendTextCalls:   2,
		},
		{
			description: "Successful prayer request completion - skip sending prayer confirmation text to prayer" +
				"requestor because they are no longer a member",

			initialMessage: messaging.TextMessage{
				Body:  "prayed",
				Phone: "+11111111111",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
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
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// StateTracker empty get response. It would over complicate to test this here.
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

			expectedDeleteItems: []struct {
				key   string
				table string
			}{
				{
					key:   "+11111111111",
					table: object.ActivePrayersTable,
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerThankYou,
					Phone: "+11111111111",
				},
			},

			expectedGetItemCalls:    6,
			expectedPutItemCalls:    3,
			expectedDeleteItemCalls: 1,
			expectedSendTextCalls:   1,
		},
		{
			description: "No active prayers to mark as prayed",

			initialMessage: messaging.TextMessage{
				Body:  "prayed",
				Phone: "+11111111111",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
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
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
			},

			expectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgNoActivePrayer,
					Phone: "+11111111111",
				},
			},

			expectedGetItemCalls:  5,
			expectedPutItemCalls:  3,
			expectedSendTextCalls: 1,
		},
		{
			description: "Error with delete Prayer",

			initialMessage: messaging.TextMessage{
				Body:  "prayed",
				Phone: "+11234567890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					// StateTracker empty get response. It would over complicate to test this here.
					Output: &dynamodb.GetItemOutput{},
					Error:  nil,
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
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					// StateTracker empty get response. It would over complicate to test this here.
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

			mockDeleteItemResults: []struct {
				Error error
			}{
				{
					Error: errors.New("delete item failure"),
				},
			},

			expectedError:           true,
			expectedGetItemCalls:    6,
			expectedPutItemCalls:    3,
			expectedDeleteItemCalls: 1,
			expectedSendTextCalls:   2,
		},
	}

	for _, test := range testCases {
		txtMock := &mock.TextSender{}
		ddbMock := &mock.DDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			setMocks(ddbMock, txtMock, test)

			if test.expectedError {
				// Handles failures for error mocks
				if err := prayertexter.MainFlow(test.initialMessage, ddbMock, txtMock); err == nil {
					t.Fatalf("expected error, got nil")
				}
				testNumMethodCalls(ddbMock, txtMock, t, test)
			} else {
				// Handles success test cases
				if err := prayertexter.MainFlow(test.initialMessage, ddbMock, txtMock); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				testNumMethodCalls(ddbMock, txtMock, t, test)
				testTxtMessage(txtMock, t, test)
				testDeleteItem(ddbMock.DeleteItemInputs, t, test)
			}
		})
	}
}
