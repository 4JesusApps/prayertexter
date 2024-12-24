package prayertexter

import (
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type TestCase struct {
	description           string
	txt                   TextMessage
	expectedMember        Member
	expectedPhones        IntercessorPhones
	expectedDeleteItemKey string

	mockGetItemOutputs   []*dynamodb.GetItemOutput
	mockGetItemErrors    []error
	mockPutItemErrors    []error
	mockDeleteItemErrors []error

	expectedGetItemCalls    int
	expectedPutItemCalls    int
	expectedDeleteItemCalls int
}

func TestMainFlowSignUp(t *testing.T) {
	testCases := []TestCase{
		{
			description: "Sign up stage ONE: user texts the word pray to start sign up process",

			txt: TextMessage{
				Body:  "pray",
				Phone: "123-456-7890",
			},

			expectedMember: Member{
				Phone:       "123-456-7890",
				SetupStage:  1,
				SetupStatus: "in-progress",
			},

			expectedGetItemCalls: 1,
			expectedPutItemCalls: 1,
		},
		{
			description: "Sign up stage ONE: user texts the word Pray (capitol P) to start sign up process",

			txt: TextMessage{
				Body:  "Pray",
				Phone: "123-456-7890",
			},

			expectedMember: Member{
				Phone:       "123-456-7890",
				SetupStage:  1,
				SetupStatus: "in-progress",
			},

			expectedGetItemCalls: 1,
			expectedPutItemCalls: 1,
		},
		{
			description: "Sign up stage ONE: get Member error",

			txt: TextMessage{
				Body:  "pray",
				Phone: "123-456-7890",
			},

			mockGetItemErrors: []error{errors.New("first get item failure")},
		},
		{
			description: "Sign up stage TWO-A: user texts name",

			txt: TextMessage{
				Body:  "John Doe",
				Phone: "123-456-7890",
			},

			mockGetItemOutputs: []*dynamodb.GetItemOutput{
				{
					Item: map[string]types.AttributeValue{
						"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
						"SetupStage":  &types.AttributeValueMemberN{Value: "1"},
						"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
					},
				},
			},

			expectedMember: Member{
				Name:        "John Doe",
				Phone:       "123-456-7890",
				SetupStage:  2,
				SetupStatus: "in-progress",
			},

			expectedGetItemCalls: 1,
			expectedPutItemCalls: 1,
		},
		{
			description: "Sign up stage TWO-B: user texts 2 to remain anonymous",

			txt: TextMessage{
				Body:  "2",
				Phone: "123-456-7890",
			},

			mockGetItemOutputs: []*dynamodb.GetItemOutput{
				{
					Item: map[string]types.AttributeValue{
						"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
						"SetupStage":  &types.AttributeValueMemberN{Value: "1"},
						"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
					},
				},
			},

			expectedMember: Member{
				Name:        "Anonymous",
				Phone:       "123-456-7890",
				SetupStage:  2,
				SetupStatus: "in-progress",
			},

			expectedGetItemCalls: 1,
			expectedPutItemCalls: 1,
		},
		{
			description: "Sign up final prayer message: user texts 1 which means they do not want to be an intercessor",

			txt: TextMessage{
				Body:  "1",
				Phone: "123-456-7890",
			},

			mockGetItemOutputs: []*dynamodb.GetItemOutput{
				{
					Item: map[string]types.AttributeValue{
						"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
						"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
						"SetupStage":  &types.AttributeValueMemberN{Value: "2"},
						"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
					},
				},
			},

			expectedMember: Member{
				Intercessor: false,
				Name:        "John Doe",
				Phone:       "123-456-7890",
				SetupStage:  99,
				SetupStatus: "completed",
			},

			expectedGetItemCalls: 1,
			expectedPutItemCalls: 1,
		},
		{
			description: "Sign up stage THREE: user texts 2 which means they want to be an intercessor",

			txt: TextMessage{
				Body:  "2",
				Phone: "123-456-7890",
			},

			mockGetItemOutputs: []*dynamodb.GetItemOutput{
				{
					Item: map[string]types.AttributeValue{
						"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
						"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
						"SetupStage":  &types.AttributeValueMemberN{Value: "2"},
						"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
					},
				},
			},

			expectedMember: Member{
				Intercessor: true,
				Name:        "John Doe",
				Phone:       "123-456-7890",
				SetupStage:  3,
				SetupStatus: "in-progress",
			},

			expectedGetItemCalls: 1,
			expectedPutItemCalls: 1,
		},
		{
			description: "Sign up final intercessor message: user texts the number of prayers they are willing to receive per week",

			txt: TextMessage{
				Body:  "10",
				Phone: "123-456-7890",
			},

			mockGetItemOutputs: []*dynamodb.GetItemOutput{
				{
					Item: map[string]types.AttributeValue{
						"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
						"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
						"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
						"SetupStage":  &types.AttributeValueMemberN{Value: "3"},
						"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
					},
				},
				{
					Item: map[string]types.AttributeValue{
						"Name": &types.AttributeValueMemberS{Value: intercessorPhonesKey},
						"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
							&types.AttributeValueMemberS{Value: "111-111-1111"},
							&types.AttributeValueMemberS{Value: "222-222-2222"},
							&types.AttributeValueMemberS{Value: "333-333-3333"},
						}},
					},
				},
			},

			expectedMember: Member{
				Intercessor:       true,
				Name:              "John Doe",
				Phone:             "123-456-7890",
				SetupStage:        99,
				SetupStatus:       "completed",
				WeeklyPrayerDate:  "dummy date/time",
				WeeklyPrayerLimit: 10,
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

			expectedGetItemCalls: 2,
			expectedPutItemCalls: 2,
		},
		{
			description: "Sign up final intercessor message: put IntercessorPhones error",

			txt: TextMessage{
				Body:  "10",
				Phone: "123-456-7890",
			},

			mockGetItemOutputs: []*dynamodb.GetItemOutput{
				{
					Item: map[string]types.AttributeValue{
						"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
						"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
						"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
						"SetupStage":  &types.AttributeValueMemberN{Value: "3"},
						"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
					},
				},
				{
					Item: map[string]types.AttributeValue{
						"Name": &types.AttributeValueMemberS{Value: intercessorPhonesKey},
						"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
							&types.AttributeValueMemberS{Value: "111-111-1111"},
							&types.AttributeValueMemberS{Value: "222-222-2222"},
							&types.AttributeValueMemberS{Value: "333-333-3333"},
						}},
					},
				},
			},

			mockPutItemErrors: []error{errors.New("first put item failure")},
		},
	}

	for _, test := range testCases {
		txtsvc := FakeTextService{}
		ddbMock := &MockDDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			ddbMock.GetItemOutputs = test.mockGetItemOutputs
			ddbMock.GetItemErrors = test.mockGetItemErrors
			ddbMock.PutItemErrors = test.mockPutItemErrors

			if len(ddbMock.GetItemErrors) > 0 || len(ddbMock.PutItemErrors) > 0 {
				// handles MainFlow signUp failures due to  GetItem or PutItem mock errors
				if err := MainFlow(test.txt, ddbMock, txtsvc); err == nil {
					t.Fatalf("expected error %v, got nil", err)
				}
			} else {
				// handles MainFlow signUp success test cases
				if err := MainFlow(test.txt, ddbMock, txtsvc); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				if ddbMock.GetItemCalls != test.expectedGetItemCalls {
					t.Errorf("expected GetItem to be called %v, got %v",
						test.expectedGetItemCalls, ddbMock.GetItemCalls)
				}

				if ddbMock.PutItemCalls != test.expectedPutItemCalls {
					t.Errorf("expected PutItem to be called %v, got %v",
						test.expectedPutItemCalls, ddbMock.PutItemCalls)
				}

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

				if mem != test.expectedMember {
					t.Errorf("expected Member %v, got %v", test.expectedMember, mem)
				}

				// this gets tested for signUpFinalIntercessorMessage only
				// signUpFinalIntercessorMessage has 2 different put methods used
				if len(ddbMock.PutItemInputs) == 2 {
					input := ddbMock.PutItemInputs[0]
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
			expectedPutItemCalls: 0,
		},
		{
			description: "Sign up stage THREE: did not send 1 or 2 as expected to answer msgMemberTypeRequest",

			txt: TextMessage{
				Body:  "wrong response to question",
				Phone: "123-456-7890",
			},

			mockGetItemOutputs: []*dynamodb.GetItemOutput{
				{
					Item: map[string]types.AttributeValue{
						"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
						"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
						"SetupStage":  &types.AttributeValueMemberN{Value: "2"},
						"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
					},
				},
			},

			expectedGetItemCalls: 1,
			expectedPutItemCalls: 0,
		},
		{
			description: "Sign up final intercessor message: did not send number as expected",

			txt: TextMessage{
				Body:  "wrong response to question",
				Phone: "123-456-7890",
			},

			mockGetItemOutputs: []*dynamodb.GetItemOutput{
				{
					Item: map[string]types.AttributeValue{
						"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
						"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
						"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
						"SetupStage":  &types.AttributeValueMemberN{Value: "3"},
						"SetupStatus": &types.AttributeValueMemberS{Value: "in-progress"},
					},
				},
			},

			expectedGetItemCalls: 1,
			expectedPutItemCalls: 0,
		},
	}

	for _, test := range testCases {
		txtsvc := FakeTextService{}
		ddbMock := &MockDDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			ddbMock.GetItemOutputs = test.mockGetItemOutputs

			if err := MainFlow(test.txt, ddbMock, txtsvc); err != nil {
				t.Fatalf("unexpected error starting MainFlow: %v", err)
			}

			if ddbMock.GetItemCalls != test.expectedGetItemCalls {
				t.Errorf("expected GetItem to be called %v, got %v",
					test.expectedGetItemCalls, ddbMock.GetItemCalls)
			}

			if ddbMock.PutItemCalls != test.expectedPutItemCalls {
				t.Errorf("expected PutItem to be called %v, got %v",
					test.expectedPutItemCalls, ddbMock.PutItemCalls)
			}

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

			mockGetItemOutputs: []*dynamodb.GetItemOutput{
				{
					Item: map[string]types.AttributeValue{
						"Name":  &types.AttributeValueMemberS{Value: "John Doe"},
						"Phone": &types.AttributeValueMemberS{Value: "123-456-7890"},
					},
				},
			},

			expectedDeleteItemKey:   "123-456-7890",
			expectedGetItemCalls:    1,
			expectedDeleteItemCalls: 1,
		},
		{
			description: "Delete intercessor member with STOP txt - phone list changes",

			txt: TextMessage{
				Body:  "STOP",
				Phone: "123-456-7890",
			},

			mockGetItemOutputs: []*dynamodb.GetItemOutput{
				{
					Item: map[string]types.AttributeValue{
						"Intercessor": &types.AttributeValueMemberBOOL{Value: true},
						"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
						"Phone":       &types.AttributeValueMemberS{Value: "123-456-7890"},
					},
				},
				{
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
			},

			expectedPhones: IntercessorPhones{
				Name: intercessorPhonesKey,
				Phones: []string{
					"111-111-1111",
					"222-222-2222",
					"333-333-3333",
				},
			},

			expectedDeleteItemKey:   "123-456-7890",
			expectedGetItemCalls:    2,
			expectedPutItemCalls:    1,
			expectedDeleteItemCalls: 1,
		},
		{
			description: "Delete member - expected error on DelItem",

			txt: TextMessage{
				Body:  "cancel",
				Phone: "123-456-7890",
			},

			mockDeleteItemErrors: []error{errors.New("delete item failure")},
		},
	}

	for _, test := range testCases {
		txtsvc := FakeTextService{}
		ddbMock := &MockDDBConnecter{}

		t.Run(test.description, func(t *testing.T) {
			ddbMock.GetItemOutputs = test.mockGetItemOutputs
			ddbMock.DeleteItemErrors = test.mockDeleteItemErrors

			if len(ddbMock.DeleteItemErrors) > 0 {
				// handles MainFlow memberDelete failures due to DeleteItem mock errors
				if err := MainFlow(test.txt, ddbMock, txtsvc); err == nil {
					t.Fatalf("expected error %v, got nil", err)
				}
			} else {
				// handles MainFlow memberDelete success test cases
				if err := MainFlow(test.txt, ddbMock, txtsvc); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

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

				delInput := ddbMock.DeleteItemInputs[0]
				if *delInput.TableName != memberTable {
					t.Errorf("expected Member table name %v, got %v", memberTable, *delInput.TableName)
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
					if *putInput.TableName != intercessorPhonesTable {
						t.Errorf("expected IntercessorPhones table name %v, got %v",
							intercessorPhonesTable, *putInput.TableName)
					}

					phones := IntercessorPhones{}
					if err := attributevalue.UnmarshalMap(putInput.Item, &phones); err != nil {
						t.Fatalf("failed to unmarshal to IntercessorPhones: %v", err)
					}

					if !reflect.DeepEqual(phones, test.expectedPhones) {
						t.Errorf("expected IntercessorPhones %v, got %v",
							test.expectedPhones, phones)
					}
				}
			}
		})
	}
}
