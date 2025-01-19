package prayertexter

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type TestCase struct {
	description string
	txt         TextMessage

	expectedGetItemCalls    int
	expectedPutItemCalls    int
	expectedDeleteItemCalls int
	expectedSendTextCalls   int

	expectedMembers       []Member
	expectedPrayers       []Prayer
	expectedTexts         []TextMessage
	expectedPhones        IntercessorPhones
	expectedDeleteItemKey string
	expectedError         bool

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

func testNumMethodCalls(ddbMock *MockDDBConnecter, txtMock *MockTextService, t *testing.T, test TestCase) {
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

func testMember(input dynamodb.PutItemInput, index int, t *testing.T, test TestCase) {
	if *input.TableName != memberTable {
		t.Errorf("expected Member table name %v, got %v", memberTable, *input.TableName)
	}

	mem := Member{}
	if err := attributevalue.UnmarshalMap(input.Item, &mem); err != nil {
		t.Fatalf("failed to unmarshal to Member: %v", err)
	}

	// change date/time to dummy time to avoid mocking time.Now()
	if mem.WeeklyPrayerDate != "" {
		mem.WeeklyPrayerDate = "dummy date/time"
	}

	if mem != test.expectedMembers[index] {
		t.Errorf("expected Member %v, got %v", test.expectedMembers[0], mem)
	}
}

func testPrayer(input dynamodb.PutItemInput, index int, t *testing.T, test TestCase, queue bool) {
	table := getPrayerTable(queue)
	if *input.TableName != table {
		t.Errorf("expected Prayer table name %v, got %v",
			table, *input.TableName)
	}

	pryr := Prayer{}
	if err := attributevalue.UnmarshalMap(input.Item, &pryr); err != nil {
		t.Fatalf("failed to unmarshal to Prayer: %v", err)
	}

	if !queue {
		pryr.Intercessor.WeeklyPrayerDate = "dummy date/time"
	} else if queue {
		pryr.IntercessorPhone = "1234567890"
	}

	if pryr != test.expectedPrayers[index] {
		t.Errorf("expected Prayer %v, got %v", test.expectedPrayers[index], pryr)
	}
}

func testPhones(input dynamodb.PutItemInput, t *testing.T, test TestCase) {
	if *input.TableName != intercessorPhonesTable {
		t.Errorf("expected IntercessorPhones table name %v, got %v",
			intercessorPhonesTable, *input.TableName)
	}

	phones := IntercessorPhones{}
	if err := attributevalue.UnmarshalMap(input.Item, &phones); err != nil {
		t.Fatalf("failed to unmarshal to IntercessorPhones: %v", err)
	}

	if !reflect.DeepEqual(phones, test.expectedPhones) {
		t.Errorf("expected IntercessorPhones %v, got %v",
			test.expectedPhones, phones)
	}
}

func testTxtMessage(txtMock *MockTextService, t *testing.T, test TestCase) {
	for i, txt := range txtMock.SendTextInputs {
		// Some text messages use PLACEHOLDER and replace that with the txt recipients name
		// Therefor to make testing easier, the message body is replaced by the msg constant
		if strings.Contains(txt.Body, "Hello! Please pray for") {
			txt.Body = msgPrayerIntro
		} else if strings.Contains(txt.Body, "There was profanity found in your prayer request:") {
			txt.Body = msgProfanityFound
		} else if strings.Contains(txt.Body, "You're prayer request has been prayed for by") {
			txt.Body = msgPrayerConfirmation
		}

		if txt != test.expectedTexts[i] {
			t.Errorf("expected txt %v, got %v",
				test.expectedTexts[i], txt)
		}
	}
}

func TestMainFlowSignUp(t *testing.T) {
	testCases := []TestCase{
		{
			description: "Sign up stage ONE: user texts the word pray to start sign up process",

			txt: TextMessage{
				Body:  "pray",
				Phone: "123-456-7890",
			},

			expectedMembers: []Member{
				{
					Phone:       "123-456-7890",
					SetupStage:  1,
					SetupStatus: "in-progress",
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgNameRequest,
					Phone: "123-456-7890",
				},
			},

			expectedGetItemCalls:  1,
			expectedPutItemCalls:  1,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up stage ONE: user texts the word Pray (capitol P) to start sign up process",

			txt: TextMessage{
				Body:  "Pray",
				Phone: "123-456-7890",
			},

			expectedMembers: []Member{
				{
					Phone:       "123-456-7890",
					SetupStage:  1,
					SetupStatus: "in-progress",
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgNameRequest,
					Phone: "123-456-7890",
				},
			},

			expectedGetItemCalls:  1,
			expectedPutItemCalls:  1,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up stage ONE: get Member error",

			txt: TextMessage{
				Body:  "pray",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: nil,
					Error:  errors.New("first get item failure"),
				},
			},

			expectedError:        true,
			expectedGetItemCalls: 1,
		},
		{
			description: "Sign up stage TWO-A: user texts name",

			txt: TextMessage{
				Body:  "John Doe",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "1"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
						},
					},
					Error: nil,
				},
			},

			expectedMembers: []Member{
				{
					Name:        "John Doe",
					Phone:       "123-456-7890",
					SetupStage:  2,
					SetupStatus: "in-progress",
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgMemberTypeRequest,
					Phone: "123-456-7890",
				},
			},

			expectedGetItemCalls:  1,
			expectedPutItemCalls:  1,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up stage TWO-B: user texts 2 to remain anonymous",

			txt: TextMessage{
				Body:  "2",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "1"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
						},
					},
					Error: nil,
				},
			},

			expectedMembers: []Member{
				{
					Name:        "Anonymous",
					Phone:       "123-456-7890",
					SetupStage:  2,
					SetupStatus: "in-progress",
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgMemberTypeRequest,
					Phone: "123-456-7890",
				},
			},

			expectedGetItemCalls:  1,
			expectedPutItemCalls:  1,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up final prayer message: user texts 1 which means they do not want to be an intercessor",

			txt: TextMessage{
				Body:  "1",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "2"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
						},
					},
					Error: nil,
				},
			},

			expectedMembers: []Member{
				{
					Intercessor: false,
					Name:        "John Doe",
					Phone:       "123-456-7890",
					SetupStage:  99,
					SetupStatus: "completed",
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgPrayerInstructions,
					Phone: "123-456-7890",
				},
			},

			expectedGetItemCalls:  1,
			expectedPutItemCalls:  1,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up stage THREE: user texts 2 which means they want to be an intercessor",

			txt: TextMessage{
				Body:  "2",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "2"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
						},
					},
					Error: nil,
				},
			},

			expectedMembers: []Member{
				{
					Intercessor: true,
					Name:        "John Doe",
					Phone:       "123-456-7890",
					SetupStage:  3,
					SetupStatus: "in-progress",
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgPrayerNumRequest,
					Phone: "123-456-7890",
				},
			},

			expectedGetItemCalls:  1,
			expectedPutItemCalls:  1,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up final intercessor message: user texts the number of prayers they are willing to receive per week",

			txt: TextMessage{
				Body:  "10",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "3"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name": &types.AttributeValueMemberS{Value: intercessorPhonesKey},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "111-111-1111"},
								&types.AttributeValueMemberS{Value: "222-222-2222"},
								&types.AttributeValueMemberS{Value: "333-333-3333"},
							}},
						},
					},
					Error: nil,
				},
			},

			expectedMembers: []Member{
				{
					Intercessor:       true,
					Name:              "John Doe",
					Phone:             "123-456-7890",
					SetupStage:        99,
					SetupStatus:       "completed",
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 10,
				},
			},

			expectedPhones: IntercessorPhones{
				Name: intercessorPhonesKey,
				Phones: []string{
					"111-111-1111",
					"222-222-2222",
					"333-333-3333",
					"123-456-7890",
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgIntercessorInstructions,
					Phone: "123-456-7890",
				},
			},

			expectedGetItemCalls:  2,
			expectedPutItemCalls:  2,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up final intercessor message: put IntercessorPhones error",

			txt: TextMessage{
				Body:  "10",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "3"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name": &types.AttributeValueMemberS{Value: intercessorPhonesKey},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "111-111-1111"},
								&types.AttributeValueMemberS{Value: "222-222-2222"},
								&types.AttributeValueMemberS{Value: "333-333-3333"},
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
					Error: errors.New("first put item failure"),
				},
			},

			expectedError:        true,
			expectedGetItemCalls: 2,
			expectedPutItemCalls: 1,
		},
	}

	for _, test := range testCases {
		txtMock := &MockTextService{}
		ddbMock := &MockDDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			ddbMock.GetItemResults = test.mockGetItemResults
			ddbMock.PutItemResults = test.mockPutItemResults
			txtMock.SendTextResults = test.mockSendTextResults

			if test.expectedError {
				// handles failures for error mocks
				if err := MainFlow(test.txt, ddbMock, txtMock); err == nil {
					t.Fatalf("expected error, got nil")
				}
				testNumMethodCalls(ddbMock, txtMock, t, test)
			} else {
				// handles success test cases
				if err := MainFlow(test.txt, ddbMock, txtMock); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				testNumMethodCalls(ddbMock, txtMock, t, test)
				testTxtMessage(txtMock, t, test)

				var input dynamodb.PutItemInput
				if len(ddbMock.PutItemInputs) == 1 {
					// put Member is the only put method in all signUpFlow stages accept for the
					// signUpFinalIntercessorMessage stage
					input = ddbMock.PutItemInputs[0]
				} else if len(ddbMock.PutItemInputs) == 2 {
					// since signUpFinalIntercessorMessage uses 2 put methods, the put Member input needs
					// to be changed accordingly here
					input = ddbMock.PutItemInputs[1]
				}
				testMember(input, 0, t, test)

				// this gets tested for signUpFinalIntercessorMessage only
				// signUpFinalIntercessorMessage has 2 different put methods used
				if len(ddbMock.PutItemInputs) == 2 {
					input := ddbMock.PutItemInputs[0]
					testPhones(input, t, test)
				}
			}
		})
	}
}

func TestMainFlowSignUpWrongInputs(t *testing.T) {
	testCases := []TestCase{
		// these test cases should do 1 get Member only because return nil on signUpWrongInput
		// 1 get Member call only shows that they took the correct flow
		{
			description: "pray misspelled - returns non registered user and exits",

			txt: TextMessage{
				Body:  "prayyy",
				Phone: "123-456-7890",
			},

			expectedGetItemCalls: 1,
		},
		{
			description: "Sign up stage THREE: did not send 1 or 2 as expected to answer msgMemberTypeRequest",

			txt: TextMessage{
				Body:  "wrong response to question",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "2"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
						},
					},
					Error: nil,
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgWrongInput,
					Phone: "123-456-7890",
				},
			},

			expectedGetItemCalls:  1,
			expectedSendTextCalls: 1,
		},
		{
			description: "Sign up final intercessor message: did not send number as expected",

			txt: TextMessage{
				Body:  "wrong response to question",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "3"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
						},
					},
					Error: nil,
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgWrongInput,
					Phone: "123-456-7890",
				},
			},

			expectedGetItemCalls:  1,
			expectedSendTextCalls: 1,
		},
	}

	for _, test := range testCases {
		txtMock := &MockTextService{}
		ddbMock := &MockDDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			ddbMock.GetItemResults = test.mockGetItemResults
			txtMock.SendTextResults = test.mockSendTextResults

			if err := MainFlow(test.txt, ddbMock, txtMock); err != nil {
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

			txt: TextMessage{
				Body:  "cancel",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "completed"},
						},
					},
					Error: nil,
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgRemoveUser,
					Phone: "123-456-7890",
				},
			},

			expectedDeleteItemKey:   "123-456-7890",
			expectedGetItemCalls:    1,
			expectedDeleteItemCalls: 1,
			expectedSendTextCalls:   1,
		},
		{
			description: "Delete intercessor member with STOP txt - phone list changes",

			txt: TextMessage{
				Body:  "STOP",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "completed"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name": &types.AttributeValueMemberS{Value: intercessorPhonesKey},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "111-111-1111"},
								&types.AttributeValueMemberS{Value: "222-222-2222"},
								&types.AttributeValueMemberS{Value: "333-333-3333"},
								&types.AttributeValueMemberS{Value: "123-456-7890"},
							}},
						},
					},
					Error: nil,
				},
			},

			expectedPhones: IntercessorPhones{
				Name: intercessorPhonesKey,
				Phones: []string{
					"111-111-1111",
					"222-222-2222",
					"333-333-3333",
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgRemoveUser,
					Phone: "123-456-7890",
				},
			},

			expectedDeleteItemKey:   "123-456-7890",
			expectedGetItemCalls:    2,
			expectedPutItemCalls:    1,
			expectedDeleteItemCalls: 1,
			expectedSendTextCalls:   1,
		},
		{
			description: "Delete member - expected error on DelItem",

			txt: TextMessage{
				Body:  "cancel",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "completed"},
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
			expectedGetItemCalls:    1,
			expectedDeleteItemCalls: 1,
		},
	}

	for _, test := range testCases {
		txtMock := &MockTextService{}
		ddbMock := &MockDDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			ddbMock.GetItemResults = test.mockGetItemResults
			ddbMock.DeleteItemResults = test.mockDeleteItemResults
			txtMock.SendTextResults = test.mockSendTextResults

			if test.expectedError {
				// handles failures for error mocks
				if err := MainFlow(test.txt, ddbMock, txtMock); err == nil {
					t.Fatalf("expected error, got nil")
				}
				testNumMethodCalls(ddbMock, txtMock, t, test)
			} else {
				// handles success test cases
				if err := MainFlow(test.txt, ddbMock, txtMock); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				testNumMethodCalls(ddbMock, txtMock, t, test)
				testTxtMessage(txtMock, t, test)

				delInput := ddbMock.DeleteItemInputs[0]
				if *delInput.TableName != memberTable {
					t.Errorf("expected Member table name %v, got %v",
						memberTable, *delInput.TableName)
				}

				mem := Member{}
				if err := attributevalue.UnmarshalMap(delInput.Key, &mem); err != nil {
					t.Fatalf("failed to unmarshal to Member: %v", err)
				}

				if mem.Phone != test.expectedDeleteItemKey {
					t.Errorf("expected Member phone %v for delete key, got %v",
						test.expectedDeleteItemKey, mem.Phone)
				}

				if len(ddbMock.PutItemInputs) > 0 {
					putInput := ddbMock.PutItemInputs[0]
					testPhones(putInput, t, test)
				}
			}
		})
	}
}

func TestMainFlowPrayerRequest(t *testing.T) {

	// getMember (initial in MainFlow)
	// getIntPhones (inside findIntercessors)
	// getMember (inside findIntercessors) (2 times)
	// putMember (inside findIntercessors) (2 times)
	// putPrayer (end of prayerRequest) (2 times)

	testCases := []TestCase{
		{
			description: "Successful simple prayer request flow",

			txt: TextMessage{
				Body:  "I need prayer for...",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "completed"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name": &types.AttributeValueMemberS{Value: intercessorPhonesKey},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "111-111-1111"},
								&types.AttributeValueMemberS{Value: "222-222-2222"},
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
							"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "222-222-2222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
			},

			expectedMembers: []Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             "111-111-1111",
					PrayerCount:       1,
					SetupStage:        99,
					SetupStatus:       "completed",
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             "222-222-2222",
					PrayerCount:       1,
					SetupStage:        99,
					SetupStatus:       "completed",
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			expectedPrayers: []Prayer{
				{
					Intercessor: Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             "111-111-1111",
						PrayerCount:       1,
						SetupStage:        99,
						SetupStatus:       "completed",
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "111-111-1111",
					Request:          "I need prayer for...",
					Requestor: Member{
						Name:        "John Doe",
						Phone:       "123-456-7890",
						SetupStage:  99,
						SetupStatus: "completed",
					},
				},
				{
					Intercessor: Member{
						Intercessor:       true,
						Name:              "Intercessor2",
						Phone:             "222-222-2222",
						PrayerCount:       1,
						SetupStage:        99,
						SetupStatus:       "completed",
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "222-222-2222",
					Request:          "I need prayer for...",
					Requestor: Member{
						Name:        "John Doe",
						Phone:       "123-456-7890",
						SetupStage:  99,
						SetupStatus: "completed",
					},
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgPrayerIntro,
					Phone: "111-111-1111",
				},
				{
					Body:  msgPrayerIntro,
					Phone: "222-222-2222",
				},
				{
					Body:  msgPrayerSentOut,
					Phone: "123-456-7890",
				},
			},

			expectedGetItemCalls:  4,
			expectedPutItemCalls:  4,
			expectedSendTextCalls: 3,
		},
		{
			description: "Profanity detected",

			txt: TextMessage{
				Body:  "fuckkk you",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "completed"},
						},
					},
					Error: nil,
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgProfanityFound,
					Phone: "123-456-7890",
				},
			},

			expectedGetItemCalls:  1,
			expectedSendTextCalls: 1,
		},
		{
			description: "Error with first put Member in findIntercessors",

			txt: TextMessage{
				Body:  "I need prayer for...",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "completed"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name": &types.AttributeValueMemberS{Value: intercessorPhonesKey},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "111-111-1111"},
								&types.AttributeValueMemberS{Value: "222-222-2222"},
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
							"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "222-222-2222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
			},

			mockPutItemResults: []struct {
				Error error
			}{
				{
					Error: errors.New("put item failure"),
				},
			},

			expectedGetItemCalls: 3,
			expectedPutItemCalls: 1,
			expectedError:        true,
		},
		{
			description: "No available intercessors because of maxed out prayer counters",

			txt: TextMessage{
				Body:  "I need prayer for...",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
							"SetupStage":  &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus": &types.AttributeValueMemberS{Value: "completed"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name": &types.AttributeValueMemberS{Value: intercessorPhonesKey},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "111-111-1111"},
								&types.AttributeValueMemberS{Value: "222-222-2222"},
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
							"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "222-222-2222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
			},

			expectedPrayers: []Prayer{
				{
					IntercessorPhone: "1234567890",
					Request:          "I need prayer for...",
					Requestor: Member{
						Name:        "John Doe",
						Phone:       "123-456-7890",
						SetupStage:  99,
						SetupStatus: "completed",
					},
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgPrayerQueued,
					Phone: "123-456-7890",
				},
			},

			expectedGetItemCalls:  5,
			expectedPutItemCalls:  1,
			expectedSendTextCalls: 1,
		},
	}

	for _, test := range testCases {
		txtMock := &MockTextService{}
		ddbMock := &MockDDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			ddbMock.GetItemResults = test.mockGetItemResults
			ddbMock.PutItemResults = test.mockPutItemResults
			txtMock.SendTextResults = test.mockSendTextResults

			if test.expectedError {
				// handles failures for error mocks
				if err := MainFlow(test.txt, ddbMock, txtMock); err == nil {
					t.Fatalf("expected error, got nil")
				}
				testNumMethodCalls(ddbMock, txtMock, t, test)
			} else {
				// handles success test cases
				if err := MainFlow(test.txt, ddbMock, txtMock); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				testNumMethodCalls(ddbMock, txtMock, t, test)
				testTxtMessage(txtMock, t, test)

				if len(ddbMock.PutItemInputs) > 1 {
					// this tests the first two items (Members) from PutItemInputs
					for i := 0; i < 2; i++ {
						input := ddbMock.PutItemInputs[i]
						testMember(input, i, t, test)
					}

					// this tests the last two items (Prayers) from PutItemInputs
					for i := 2; i < 4; i++ {
						input := ddbMock.PutItemInputs[i]
						testPrayer(input, i-2, t, test, false)
					}
				} else if len(ddbMock.PutItemInputs) == 1 {
					// this tests when there are no intercessors, so there is only 1 put prayer to queued prayers table
					input := ddbMock.PutItemInputs[0]
					testPrayer(input, 0, t, test, true)
				}
			}
		})
	}
}

func TestFindIntercessors(t *testing.T) {
	testCases := []TestCase{
		{
			// this mocks the get member outputs so we do not need to worry about the math/rand part
			// #3 gets selected because the date is past 7 days; date + counter gets reset
			// #5 gets chosen because it has 1 prayer slot available
			description: "This should pick #3 and #5 intercessors based on prayer counts/dates",

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name": &types.AttributeValueMemberS{Value: intercessorPhonesKey},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "111-111-1111"},
								&types.AttributeValueMemberS{Value: "222-222-2222"},
								&types.AttributeValueMemberS{Value: "333-333-3333"},
								&types.AttributeValueMemberS{Value: "444-444-4444"},
								&types.AttributeValueMemberS{Value: "555-555-5555"},
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
							"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "222-222-2222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "100"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().AddDate(0, 0, -2).Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "100"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor3"},
							"Phone":             &types.AttributeValueMemberS{Value: "333-333-3333"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "15"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().AddDate(0, 0, -7).Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "15"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor4"},
							"Phone":             &types.AttributeValueMemberS{Value: "444-444-4444"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "9"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().AddDate(0, 0, -6).Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "9"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor5"},
							"Phone":             &types.AttributeValueMemberS{Value: "555-555-5555"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "4"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
			},

			expectedMembers: []Member{
				{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             "333-333-3333",
					PrayerCount:       1,
					SetupStage:        99,
					SetupStatus:       "completed",
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 15,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor5",
					Phone:             "555-555-5555",
					PrayerCount:       5,
					SetupStage:        99,
					SetupStatus:       "completed",
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			expectedGetItemCalls: 6,
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
							"Name": &types.AttributeValueMemberS{Value: intercessorPhonesKey},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "111-111-1111"},
								&types.AttributeValueMemberS{Value: "222-222-2222"},
								&types.AttributeValueMemberS{Value: "333-333-3333"},
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
							"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "222-222-2222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor3"},
							"Phone":             &types.AttributeValueMemberS{Value: "333-333-3333"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "4"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
			},

			expectedMembers: []Member{
				{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             "333-333-3333",
					PrayerCount:       5,
					SetupStage:        99,
					SetupStatus:       "completed",
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			expectedGetItemCalls: 4,
			expectedPutItemCalls: 1,
		},
		{
			description: "This should return nil because all intercessors are maxed out on prayer requests",

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Name": &types.AttributeValueMemberS{Value: intercessorPhonesKey},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "111-111-1111"},
								&types.AttributeValueMemberS{Value: "222-222-2222"},
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
							"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
							"Phone":             &types.AttributeValueMemberS{Value: "222-222-2222"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "5"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
			},

			expectedGetItemCalls: 3,
		},
	}

	for _, test := range testCases {
		txtMock := &MockTextService{}
		ddbMock := &MockDDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			ddbMock.GetItemResults = test.mockGetItemResults

			if test.expectedError {
				// handles failures for error mocks
				if _, err := findIntercessors(ddbMock); err == nil {
					t.Fatalf("expected error, got nil")
				}
				testNumMethodCalls(ddbMock, txtMock, t, test)
			} else {
				// handles success test cases
				intercessors, err := findIntercessors(ddbMock)
				if err != nil {
					t.Fatalf("unexpected error starting findIntercessors: %v", err)
				}

				testNumMethodCalls(ddbMock, txtMock, t, test)

				current_date := time.Now().Truncate(time.Minute)
				for indx, intrcsr := range intercessors {
					prayer_date, err := time.Parse(time.RFC3339, intrcsr.WeeklyPrayerDate)
					if err != nil {
						t.Fatalf("date parse failed during find intercessors: %v", err)
					}
					prayer_date = prayer_date.Truncate(time.Minute)
					if current_date.Equal(prayer_date) {
						intercessors[indx].WeeklyPrayerDate = "dummy date/time"
					} else {
						t.Fatalf("expected dates to match: %v %v", current_date, prayer_date)
					}
				}

				if !slices.Equal(intercessors, test.expectedMembers) {
					t.Errorf("expected []Member %v, got %v", test.expectedMembers, intercessors)
				}
			}
		})
	}
}

func TestMainFlowCompletePrayer(t *testing.T) {
	testCases := []TestCase{
		{
			description: "Successful prayer request completion",

			txt: TextMessage{
				Body:  "prayed",
				Phone: "111-111-1111",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
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
									"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
									"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
									"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
									"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
									"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
									"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
								},
							},
							"IntercessorPhone": &types.AttributeValueMemberS{Value: "111-111-1111"},
							"Request":          &types.AttributeValueMemberS{Value: "Please pray me.."},
							"Requestor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
									"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
									"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
									"SetupStage":  &types.AttributeValueMemberN{Value: "99"},
									"SetupStatus": &types.AttributeValueMemberS{Value: "completed"},
								},
							},
						},
					},
					Error: nil,
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgPrayerThankYou,
					Phone: "111-111-1111",
				},
				{
					Body:  msgPrayerConfirmation,
					Phone: "123-456-7890",
				},
			},

			expectedDeleteItemKey:   "111-111-1111",
			expectedGetItemCalls:    2,
			expectedDeleteItemCalls: 1,
			expectedSendTextCalls:   2,
		},
		{
			description: "No active prayers to mark as prayed",

			txt: TextMessage{
				Body:  "prayed",
				Phone: "111-111-1111",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					Error: nil,
				},
			},

			expectedTexts: []TextMessage{
				{
					Body:  msgNoActivePrayer,
					Phone: "111-111-1111",
				},
			},

			expectedGetItemCalls:  2,
			expectedSendTextCalls: 1,
		},
		{
			description: "Error with first Member get in MainFlow",

			txt: TextMessage{
				Body:  "prayed",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{},
					Error:  errors.New("first get item failure"),
				},
			},

			expectedError:        true,
			expectedGetItemCalls: 1,
		},
		{
			description: "Error with delete Prayer",

			txt: TextMessage{
				Body:  "prayed",
				Phone: "123-456-7890",
			},

			mockGetItemResults: []struct {
				Output *dynamodb.GetItemOutput
				Error  error
			}{
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor1"},
							"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
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
									"Phone":             &types.AttributeValueMemberS{Value: "111-111-1111"},
									"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
									"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
									"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
									"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
									"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
								},
							},
							"IntercessorPhone": &types.AttributeValueMemberS{Value: "111-111-1111"},
							"Request":          &types.AttributeValueMemberS{Value: "Please pray me.."},
							"Requestor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
									"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
									"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
									"SetupStage":  &types.AttributeValueMemberN{Value: "99"},
									"SetupStatus": &types.AttributeValueMemberS{Value: "completed"},
								},
							},
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
			expectedGetItemCalls:    2,
			expectedDeleteItemCalls: 1,
			expectedSendTextCalls:   2,
		},
	}

	for _, test := range testCases {
		txtMock := &MockTextService{}
		ddbMock := &MockDDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			ddbMock.GetItemResults = test.mockGetItemResults
			ddbMock.DeleteItemResults = test.mockDeleteItemResults
			txtMock.SendTextResults = test.mockSendTextResults

			if test.expectedError {
				// handles failures for error mocks
				if err := MainFlow(test.txt, ddbMock, txtMock); err == nil {
					t.Fatalf("expected error, got nil")
				}
				testNumMethodCalls(ddbMock, txtMock, t, test)
			} else {
				// handles success test cases
				if err := MainFlow(test.txt, ddbMock, txtMock); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				testNumMethodCalls(ddbMock, txtMock, t, test)
				testTxtMessage(txtMock, t, test)

				if len(ddbMock.DeleteItemInputs) != 0 {
					delInput := ddbMock.DeleteItemInputs[0]
					if *delInput.TableName != activePrayersTable {
						t.Errorf("expected Prayer table name %v, got %v",
							activePrayersTable, *delInput.TableName)
					}

					pryr := Prayer{}
					if err := attributevalue.UnmarshalMap(delInput.Key, &pryr); err != nil {
						t.Fatalf("failed to unmarshal to Prayer: %v", err)
					}

					if pryr.IntercessorPhone != test.expectedDeleteItemKey {
						t.Errorf("expected Prayer phone %v for delete key, got %v",
							test.expectedDeleteItemKey, pryr.IntercessorPhone)
					}
				}
			}
		})
	}
}
