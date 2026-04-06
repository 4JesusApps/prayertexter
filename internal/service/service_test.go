package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/4JesusApps/prayertexter/internal/apperrors"
	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/model"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/4JesusApps/prayertexter/internal/test"
	"github.com/4JesusApps/prayertexter/internal/test/mock"
	"github.com/4JesusApps/prayertexter/internal/test/testutil"
)

func TestMainFlowBlockUser(t *testing.T) {
	testCases := []test.Case{
		{
			Description: "Someone tries to block a user but is not an admin",

			InitialMessage: messaging.TextMessage{
				Body:  "#BLOCK 777-777-7777",
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgUnauthorized,
					Phone: testutil.PhoneMember,
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Blocking a user is attempted, but phone number is invalid",

			InitialMessage: messaging.TextMessage{
				Body:  "#block 123",
				Phone: testutil.PhoneAdmin,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Administrator: true,
					Name:          "Admin User",
					Phone:         testutil.PhoneAdmin,
					SetupStage:    model.SignUpStepFinal,
					SetupStatus:   model.SetupComplete,
				}),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgInvalidPhone,
					Phone: testutil.PhoneAdmin,
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Blocking a user is attempted, but user is already blocked",

			InitialMessage: messaging.TextMessage{
				Body:  "#block 123-456-7890",
				Phone: testutil.PhoneAdmin,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Administrator: true,
					Name:          "Admin User",
					Phone:         testutil.PhoneAdmin,
					SetupStage:    model.SignUpStepFinal,
					SetupStatus:   model.SetupComplete,
				}),
				testutil.BlockedPhonesItem(testutil.PhoneMember, testutil.PhoneAlt1, testutil.PhoneAlt2),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgUserAlreadyBlocked,
					Phone: testutil.PhoneAdmin,
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Admin successfully blocks a user",

			InitialMessage: messaging.TextMessage{
				Body:  "#block 123-456-7890",
				Phone: testutil.PhoneAdmin,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Administrator: true,
					Name:          "Admin User",
					Phone:         testutil.PhoneAdmin,
					SetupStage:    model.SignUpStepFinal,
					SetupStatus:   model.SetupComplete,
				}),
				testutil.BlockedPhonesItem(testutil.PhoneAlt1, testutil.PhoneAlt2),
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   testutil.PhoneMember,
					Table: "Member",
				},
			},

			ExpectedBlockPhones: model.BlockedPhones{
				Key: model.BlockedPhonesKeyValue,
				Phones: []string{
					testutil.PhoneAlt1,
					testutil.PhoneAlt2,
					testutil.PhoneMember,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: testutil.PhoneMember,
				},
				{
					Body:  messaging.MsgBlockedNotification + messaging.MsgHelp,
					Phone: testutil.PhoneMember,
				},
				{
					Body:  messaging.MsgSuccessfullyBlocked,
					Phone: testutil.PhoneAdmin,
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
				Phone: testutil.PhoneAdmin,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Administrator: true,
					Name:          "Admin User",
					Phone:         testutil.PhoneAdmin,
					SetupStage:    model.SignUpStepFinal,
					SetupStatus:   model.SetupComplete,
				}),
				testutil.BlockedPhonesItem(testutil.PhoneAlt1, testutil.PhoneAlt2),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Bad User",
					Phone:             testutil.PhoneMember,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date",
					WeeklyPrayerLimit: 5,
				}),
				testutil.IntercessorPhonesItem(
					testutil.PhoneMember, testutil.PhoneIntercessor, testutil.PhoneAlt3, testutil.PhoneAdmin,
				),
				testutil.PrayerItem(model.Prayer{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Bad User",
						Phone:             testutil.PhoneMember,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneMember,
					Request:          "Please pray me..",
					Requestor: model.Member{
						Name:        "John Doe",
						Phone:       testutil.PhoneAlt4,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				}),
				testutil.PrayerItem(model.Prayer{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Bad User",
						Phone:             testutil.PhoneMember,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneMember,
					Request:          "Please pray me..",
					Requestor: model.Member{
						Name:        "John Doe",
						Phone:       testutil.PhoneAlt4,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				}),
			},

			ExpectedIntPhones: model.IntercessorPhones{
				Key: model.IntercessorPhonesKeyValue,
				Phones: []string{
					testutil.PhoneIntercessor,
					testutil.PhoneAlt3,
					testutil.PhoneAdmin,
				},
			},

			ExpectedPrayers: []model.Prayer{
				{
					Intercessor:      model.Member{},
					IntercessorPhone: "dummy ID",
					Request:          "Please pray me..",
					Requestor: model.Member{
						Intercessor: false,
						Name:        "John Doe",
						Phone:       testutil.PhoneAlt4,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   testutil.PhoneMember,
					Table: "Member",
				},
				{
					Key:   testutil.PhoneMember,
					Table: "ActivePrayer",
				},
			},

			ExpectedBlockPhones: model.BlockedPhones{
				Key: model.BlockedPhonesKeyValue,
				Phones: []string{
					testutil.PhoneAlt1,
					testutil.PhoneAlt2,
					testutil.PhoneMember,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: testutil.PhoneMember,
				},
				{
					Body:  messaging.MsgBlockedNotification + messaging.MsgHelp,
					Phone: testutil.PhoneMember,
				},
				{
					Body:  messaging.MsgSuccessfullyBlocked,
					Phone: testutil.PhoneAdmin,
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

			MockGetItemResults: []testutil.GetItemResult{
				// Member empty get response.
				testutil.EmptyGetResult(),
				testutil.BlockedPhonesItem(
					testutil.PhoneIntercessor,
					testutil.PhoneAlt1,
					testutil.PhoneAlt2,
					testutil.PhoneAlt3,
					"+19999999999",
				),
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
			cfg := config.Load()
			svc := service.NewService(cfg, ddbMock, txtMock)

			if tc.ExpectedError {
				// Handles failures for error mocks.
				if err := svc.MainFlow(ctx, tc.InitialMessage); err == nil {
					t.Fatalf("expected error, got nil")
				}

				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases.
				if err := svc.MainFlow(ctx, tc.InitialMessage); err != nil {
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
				Phone: testutil.PhoneMember,
			},

			ExpectedMembers: []model.Member{
				{
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepOne,
					SetupStatus: model.SetupInProgress,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgNameRequest,
					Phone: testutil.PhoneMember,
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
				Phone: testutil.PhoneMember,
			},

			ExpectedMembers: []model.Member{
				{
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepOne,
					SetupStatus: model.SetupInProgress,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgNameRequest,
					Phone: testutil.PhoneMember,
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepOne,
					SetupStatus: model.SetupInProgress,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepTwo,
					SetupStatus: model.SetupInProgress,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgMemberTypeRequest,
					Phone: testutil.PhoneMember,
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepOne,
					SetupStatus: model.SetupInProgress,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgProfanityDetected,
					Phone: testutil.PhoneMember,
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up stage TWO-A: user texts invalid name, less than 2 letters",

			InitialMessage: messaging.TextMessage{
				Body:  "A",
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepOne,
					SetupStatus: model.SetupInProgress,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgInvalidName,
					Phone: testutil.PhoneMember,
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up stage TWO-A: user texts invalid name, contains non letters",

			InitialMessage: messaging.TextMessage{
				Body:  "Dude!",
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepOne,
					SetupStatus: model.SetupInProgress,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgInvalidName,
					Phone: testutil.PhoneMember,
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up stage TWO-B: user texts 2 to remain anonymous",

			InitialMessage: messaging.TextMessage{
				Body:  "2",
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepOne,
					SetupStatus: model.SetupInProgress,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Name:        "Anonymous",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepTwo,
					SetupStatus: model.SetupInProgress,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgMemberTypeRequest,
					Phone: testutil.PhoneMember,
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepTwo,
					SetupStatus: model.SetupInProgress,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor: false,
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgSignUpConfirmation,
					Phone: testutil.PhoneMember,
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepTwo,
					SetupStatus: model.SetupInProgress,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor: true,
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepThree,
					SetupStatus: model.SetupInProgress,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerNumRequest,
					Phone: testutil.PhoneMember,
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Intercessor: true,
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepThree,
					SetupStatus: model.SetupInProgress,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor, testutil.PhoneAlt1, testutil.PhoneAlt2),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor:       true,
					Name:              "John Doe",
					Phone:             testutil.PhoneMember,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 9999,
				},
			},

			ExpectedIntPhones: model.IntercessorPhones{
				Key: model.IntercessorPhonesKeyValue,
				Phones: []string{
					testutil.PhoneIntercessor,
					testutil.PhoneAlt1,
					testutil.PhoneAlt2,
					testutil.PhoneMember,
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body: messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgIntercessorInstructions + "\n\n" +
						messaging.MsgSignUpConfirmation,
					Phone: testutil.PhoneMember,
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Intercessor: true,
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepThree,
					SetupStatus: model.SetupInProgress,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor, testutil.PhoneAlt1, testutil.PhoneAlt2),
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
			cfg := config.Load()
			svc := service.NewService(cfg, ddbMock, txtMock)

			if tc.ExpectedError {
				// Handles failures for error mocks.
				if err := svc.MainFlow(ctx, tc.InitialMessage); err == nil {
					t.Fatalf("expected error, got nil")
				}

				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases.
				if err := svc.MainFlow(ctx, tc.InitialMessage); err != nil {
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
				Phone: testutil.PhoneMember,
			},

			ExpectedGetItemCalls: 2,
		},
		{
			Description: "Sign up stage THREE: did not send 1 or 2 as expected to answer MsgMemberTypeRequest",

			InitialMessage: messaging.TextMessage{
				Body:  "wrong response to question",
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepTwo,
					SetupStatus: model.SetupInProgress,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgWrongInput,
					Phone: testutil.PhoneMember,
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Sign up final intercessor message: did not send number as expected",

			InitialMessage: messaging.TextMessage{
				Body:  "wrong response to question",
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Intercessor: true,
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepThree,
					SetupStatus: model.SetupInProgress,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgWrongInput,
					Phone: testutil.PhoneMember,
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
			cfg := config.Load()
			svc := service.NewService(cfg, ddbMock, txtMock)

			if err := svc.MainFlow(ctx, tc.InitialMessage); err != nil {
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   testutil.PhoneMember,
					Table: "Member",
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: testutil.PhoneMember,
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
				Phone: testutil.PhoneAlt3,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor4",
					Phone:             testutil.PhoneAlt3,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date",
					WeeklyPrayerLimit: 5,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
				testutil.IntercessorPhonesItem(
					testutil.PhoneIntercessor,
					testutil.PhoneAlt1,
					testutil.PhoneAlt2,
					testutil.PhoneAlt3,
				),
			},

			ExpectedIntPhones: model.IntercessorPhones{
				Key: model.IntercessorPhonesKeyValue,
				Phones: []string{
					testutil.PhoneIntercessor,
					testutil.PhoneAlt1,
					testutil.PhoneAlt2,
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   testutil.PhoneAlt3,
					Table: "Member",
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: testutil.PhoneAlt3,
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
				Phone: testutil.PhoneAlt3,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor4",
					Phone:             testutil.PhoneAlt3,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date",
					WeeklyPrayerLimit: 5,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
				testutil.IntercessorPhonesItem(
					testutil.PhoneIntercessor,
					testutil.PhoneAlt1,
					testutil.PhoneAlt2,
					testutil.PhoneAlt3,
				),
				testutil.PrayerItem(model.Prayer{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor4",
						Phone:             testutil.PhoneAlt3,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneAlt3,
					Request:          "Please pray me..",
					Requestor: model.Member{
						Name:        "John Doe",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				}),
				testutil.PrayerItem(model.Prayer{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor4",
						Phone:             testutil.PhoneAlt3,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneAlt3,
					Request:          "Please pray me..",
					Requestor: model.Member{
						Name:        "John Doe",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				}),
			},

			ExpectedIntPhones: model.IntercessorPhones{
				Key: model.IntercessorPhonesKeyValue,
				Phones: []string{
					testutil.PhoneIntercessor,
					testutil.PhoneAlt1,
					testutil.PhoneAlt2,
				},
			},

			ExpectedPrayers: []model.Prayer{
				{
					Intercessor:      model.Member{},
					IntercessorPhone: "dummy ID",
					Request:          "Please pray me..",
					Requestor: model.Member{
						Intercessor: false,
						Name:        "John Doe",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   testutil.PhoneAlt3,
					Table: "Member",
				},
				{
					Key:   testutil.PhoneAlt3,
					Table: "ActivePrayer",
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgRemoveUser,
					Phone: testutil.PhoneAlt3,
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Intercessor: true,
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
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
			cfg := config.Load()
			svc := service.NewService(cfg, ddbMock, txtMock)

			if tc.ExpectedError {
				// Handles failures for error mocks.
				if err := svc.MainFlow(ctx, tc.InitialMessage); err == nil {
					t.Fatalf("expected error, got nil")
				}

				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases.
				if err := svc.MainFlow(ctx, tc.InitialMessage); err != nil {
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgHelp,
					Phone: testutil.PhoneMember,
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Setup stage 1 user texts help and receives the help message",

			InitialMessage: messaging.TextMessage{
				Body:  "help",
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepOne,
					SetupStatus: model.SetupInProgress,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgHelp,
					Phone: testutil.PhoneMember,
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
			cfg := config.Load()
			svc := service.NewService(cfg, ddbMock, txtMock)

			if err := svc.MainFlow(ctx, tc.InitialMessage); err != nil {
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor, testutil.PhoneAlt1),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       0,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "2024-12-01T01:00:00Z",
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       0,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "2024-12-01T01:00:00Z",
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedPrayers: []model.Prayer{
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             testutil.PhoneIntercessor,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneIntercessor,
					Request:          "I need prayer for these things...",
					Requestor: model.Member{
						Name:        "John Doe",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor2",
						Phone:             testutil.PhoneAlt1,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneAlt1,
					Request:          "I need prayer for these things...",
					Requestor: model.Member{
						Name:        "John Doe",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: testutil.PhoneIntercessor,
				},
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: testutil.PhoneAlt1,
				},
				{
					Body:  messaging.MsgPrayerAssigned,
					Phone: testutil.PhoneMember,
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgProfanityDetected,
					Phone: testutil.PhoneMember,
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Invalid prayer - not enough words",

			InitialMessage: messaging.TextMessage{
				Body:  "this is four words",
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgInvalidRequest,
					Phone: testutil.PhoneMember,
				},
			},

			ExpectedGetItemCalls:  2,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Error with first put Prayer in FindIntercessors",

			InitialMessage: messaging.TextMessage{
				Body:  "I need prayer for these things...",
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor, testutil.PhoneAlt1),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       0,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "2024-12-01T01:00:00Z",
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       0,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "2024-12-01T01:00:00Z",
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor, testutil.PhoneAlt1),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       5,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       5,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
			},

			ExpectedPrayers: []model.Prayer{
				{
					IntercessorPhone: "dummy ID",
					Request:          "I need prayer for these things...",
					Requestor: model.Member{
						Name:        "John Doe",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerQueued,
					Phone: testutil.PhoneMember,
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor, testutil.PhoneAlt1),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       0,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "2024-12-01T01:00:00Z",
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       0,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "2024-12-01T01:00:00Z",
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedPrayers: []model.Prayer{
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             testutil.PhoneIntercessor,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneIntercessor,
					Request:          "I need prayer for these things...",
					Requestor: model.Member{
						Name:        "Anonymous",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor2",
						Phone:             testutil.PhoneAlt1,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneAlt1,
					Request:          "I need prayer for these things...",
					Requestor: model.Member{
						Name:        "Anonymous",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: testutil.PhoneIntercessor,
				},
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: testutil.PhoneAlt1,
				},
				{
					Body:  messaging.MsgPrayerAssigned,
					Phone: testutil.PhoneMember,
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
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor, testutil.PhoneAlt1),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       0,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "2024-12-01T01:00:00Z",
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       0,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "2024-12-01T01:00:00Z",
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedPrayers: []model.Prayer{
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             testutil.PhoneIntercessor,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneIntercessor,
					Request:          "I need prayer for these things...  please keep this private",
					Requestor: model.Member{
						Name:        "Anonymous",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor2",
						Phone:             testutil.PhoneAlt1,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneAlt1,
					Request:          "I need prayer for these things...  please keep this private",
					Requestor: model.Member{
						Name:        "Anonymous",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: testutil.PhoneIntercessor,
				},
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: testutil.PhoneAlt1,
				},
				{
					Body:  messaging.MsgPrayerAssigned,
					Phone: testutil.PhoneMember,
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
			cfg := config.Load()
			svc := service.NewService(cfg, ddbMock, txtMock)

			if tc.ExpectedError {
				// handles failures for error mocks
				if err := svc.MainFlow(ctx, tc.InitialMessage); err == nil {
					t.Fatalf("expected error, got nil")
				}

				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// handles success test cases
				if err := svc.MainFlow(ctx, tc.InitialMessage); err != nil {
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

			MockGetItemResults: []testutil.GetItemResult{
				testutil.IntercessorPhonesItem(
					testutil.PhoneIntercessor, testutil.PhoneAlt1, testutil.PhoneAlt2, testutil.PhoneAlt3, "+15555555555",
				),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       5,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       100,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().AddDate(0, 0, -2).Format(time.RFC3339),
					WeeklyPrayerLimit: 100,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             testutil.PhoneAlt2,
					PrayerCount:       15,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().AddDate(0, 0, -8).Format(time.RFC3339),
					WeeklyPrayerLimit: 15,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor4",
					Phone:             testutil.PhoneAlt3,
					PrayerCount:       9,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().AddDate(0, 0, -6).Format(time.RFC3339),
					WeeklyPrayerLimit: 9,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor5",
					Phone:             "+15555555555",
					PrayerCount:       4,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             testutil.PhoneAlt2,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 15,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor5",
					Phone:             "+15555555555",
					PrayerCount:       5,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedGetItemCalls: 11,
			ExpectedPutItemCalls: 2,
		},
		{
			Description: "This should return a single intercessor because only one does not have maxed out prayers",

			MockGetItemResults: []testutil.GetItemResult{
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor, testutil.PhoneAlt1, testutil.PhoneAlt2),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       5,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       5,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             testutil.PhoneAlt2,
					PrayerCount:       4,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             testutil.PhoneAlt2,
					PrayerCount:       5,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
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

			MockGetItemResults: []testutil.GetItemResult{
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor, testutil.PhoneAlt4),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       2,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
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

			MockGetItemResults: []testutil.GetItemResult{
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor, testutil.PhoneAlt1),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       5,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       5,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: nil,

			ExpectedGetItemCalls: 5,
		},
		{
			Description: "This should return a single intercessor because, while they all are not maxed out on" +
				"prayers, 2 of them already have active prayers",

			MockGetItemResults: []testutil.GetItemResult{
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor, testutil.PhoneAlt1, testutil.PhoneAlt2),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				testutil.PrayerItem(model.Prayer{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             testutil.PhoneIntercessor,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneIntercessor,
					Request:          "Please pray me..",
					Requestor: model.Member{
						Name:        "John Doe",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				}),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             testutil.PhoneAlt2,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				testutil.PrayerItem(model.Prayer{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor3",
						Phone:             testutil.PhoneAlt2,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneAlt2,
					Request:          "Please pray me..",
					Requestor: model.Member{
						Name:        "John Doe",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				}),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       2,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
			},

			ExpectedGetItemCalls: 7,
			ExpectedPutItemCalls: 1,
		},
		{
			Description: "This should return 2 intercessors even when GenRandPhones returns 3 available intercessors",

			MockGetItemResults: []testutil.GetItemResult{
				testutil.IntercessorPhonesItem(
					testutil.PhoneIntercessor,
					testutil.PhoneAlt1,
					testutil.PhoneAlt2,
					testutil.PhoneAlt3,
				),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             testutil.PhoneAlt1,
					PrayerCount:       5,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             testutil.PhoneAlt2,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor4",
					Phone:             testutil.PhoneAlt3,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       2,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             testutil.PhoneAlt2,
					PrayerCount:       2,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
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
			cfg := config.Load()
			svc := service.NewService(cfg, ddbMock, txtMock)

			if tc.ExpectedError {
				// Handles failures for error mocks.
				if _, err := svc.FindIntercessors(ctx, testutil.PhoneAlt4); err == nil {
					t.Fatalf("expected error, got nil")
				}

				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases.
				_, err := svc.FindIntercessors(ctx, testutil.PhoneAlt4)
				if err != nil && !errors.Is(err, apperrors.ErrNoAvailableIntercessors) {
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
				Phone: testutil.PhoneIntercessor,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date",
					WeeklyPrayerLimit: 5,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
				testutil.PrayerItem(model.Prayer{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             testutil.PhoneIntercessor,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneIntercessor,
					Request:          "Please pray me..",
					Requestor: model.Member{
						Name:        "John Doe",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				}),
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   testutil.PhoneIntercessor,
					Table: "ActivePrayer",
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerThankYou,
					Phone: testutil.PhoneIntercessor,
				},
				{
					Body:  messaging.MsgPrayerConfirmation,
					Phone: testutil.PhoneMember,
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
				Phone: testutil.PhoneIntercessor,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date",
					WeeklyPrayerLimit: 5,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
				testutil.PrayerItem(model.Prayer{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             testutil.PhoneIntercessor,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneIntercessor,
					Request:          "Please pray me..",
					Requestor: model.Member{
						Name:        "John Doe",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				}),
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   testutil.PhoneIntercessor,
					Table: "ActivePrayer",
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerThankYou,
					Phone: testutil.PhoneIntercessor,
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
				Phone: testutil.PhoneIntercessor,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date",
					WeeklyPrayerLimit: 5,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgNoActivePrayer,
					Phone: testutil.PhoneIntercessor,
				},
			},

			ExpectedGetItemCalls:  3,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "Error with delete Prayer",

			InitialMessage: messaging.TextMessage{
				Body:  "prayed",
				Phone: testutil.PhoneMember,
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date",
					WeeklyPrayerLimit: 5,
				}),
				// BlockedPhones empty get response.
				testutil.EmptyGetResult(),
				testutil.PrayerItem(model.Prayer{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             testutil.PhoneIntercessor,
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneIntercessor,
					Request:          "Please pray me..",
					Requestor: model.Member{
						Name:        "John Doe",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				}),
				testutil.MemberItem(model.Member{
					Name:        "John Doe",
					Phone:       testutil.PhoneMember,
					SetupStage:  model.SignUpStepFinal,
					SetupStatus: model.SetupComplete,
				}),
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
			cfg := config.Load()
			svc := service.NewService(cfg, ddbMock, txtMock)

			if tc.ExpectedError {
				// Handles failures for error mocks
				if err := svc.MainFlow(ctx, tc.InitialMessage); err == nil {
					t.Fatalf("expected error, got nil")
				}

				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases
				if err := svc.MainFlow(ctx, tc.InitialMessage); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				test.RunAllCommonTests(ddbMock, txtMock, t, tc)
			}
		})
	}
}
