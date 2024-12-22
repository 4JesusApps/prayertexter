package prayertexter

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestMainFlowSignUpFlow(t *testing.T) {
	type TestCase struct {
		description    string
		txt            TextMessage
		expectedMember Member
		expectedPhones IntercessorPhones

		mockGetItemOutputs []*dynamodb.GetItemOutput

		expectedGetItemCalls  int
		expectedGetItemErrors []error

		expectedPutItemCalls  int
		expectedPutItemErrors []error
	}

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
				WeeklyPrayerDate:  "???",
				WeeklyPrayerLimit: 10,
			},

			expectedPhones: IntercessorPhones{
				Name: intercessorPhonesKey,
				Phones: []string{
					"111-111-1111",
					"222-222-2222",
					"333-333-3333",
				},
			},

			expectedGetItemCalls: 2,
			expectedPutItemCalls: 2,
		},
	}

	for _, testCase := range testCases {
		txtsvc := FakeTextService{}
		ddbMock := &MockDDBConnecter{}

		t.Run(testCase.description, func(t *testing.T) {
			ddbMock.GetItemOutputs = testCase.mockGetItemOutputs

			err := MainFlow(testCase.txt, ddbMock, txtsvc)
			if err != nil {
				t.Fatalf("unexpected error starting MainFlow: %v", err)
			}

			if ddbMock.GetItemCalls != testCase.expectedGetItemCalls {
				t.Errorf("expected GetItem to be called %v, got %v",
					testCase.expectedGetItemCalls, ddbMock.GetItemCalls)
			}

			if ddbMock.PutItemCalls != testCase.expectedPutItemCalls {
				t.Errorf("expected PutItem to be called %v, got %v",
					testCase.expectedPutItemCalls, ddbMock.PutItemCalls)
			}

			if len(ddbMock.PutItemInputs) == 1 {
				// this path gets taken on all signUpFlow stages accept for last stage because
				// this they only use 1 put method
				input := ddbMock.PutItemInputs[0]
				if *input.TableName != memberTable {
					t.Errorf("expected Member table name %v, got %v", memberTable, *input.TableName)
				}

				mem := Member{}
				if err := attributevalue.UnmarshalMap(input.Item, &mem); err != nil {
					t.Fatalf("failed to unmarshal to Member: %v", err)
				}

				if mem != testCase.expectedMember {
					t.Errorf("expected Member %v, got %v", testCase.expectedMember, mem)
				}
			} else if len(ddbMock.PutItemInputs) == 2 {
				// this path gets taken for signUpFinalIntercessorMessage only
				input := ddbMock.PutItemInputs[0]
				if *input.TableName != intercessorPhonesTable {
					t.Errorf("expected IntercessorPhones table name %v, got %v",
						intercessorPhonesTable, *input.TableName)
				}

				phones := IntercessorPhones{}
				if err := attributevalue.UnmarshalMap(input.Item, &phones); err != nil {
					t.Fatalf("failed to unmarshal to IntercessorPhones: %v", err)
				}

				if reflect.DeepEqual(phones, testCase.expectedPhones) {
					t.Errorf("expected IntercessorPhones %v, got %v",
						testCase.expectedPhones, phones)
				}

				input = ddbMock.PutItemInputs[1]
				if *input.TableName != memberTable {
					t.Errorf("expected Member table name %v, got %v", memberTable, *input.TableName)
				}

				mem := Member{}
				if err := attributevalue.UnmarshalMap(input.Item, &mem); err != nil {
					t.Fatalf("failed to unmarshal to Member: %v", err)
				}

				if mem != testCase.expectedMember {
					t.Errorf("expected Member %v, got %v", testCase.expectedMember, mem)
				}

			}
		})
	}
}
