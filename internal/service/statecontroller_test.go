package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/model"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/4JesusApps/prayertexter/internal/test"
	"github.com/4JesusApps/prayertexter/internal/test/mock"
	"github.com/4JesusApps/prayertexter/internal/test/testutil"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestAssignQueuedPrayers(t *testing.T) {
	testCases := []test.Case{
		{
			Description: "3 queued prayers get assigned to the first 6 intercessors (2 intercessors per prayer)",

			MockScanResults: []struct {
				Output *dynamodb.ScanOutput
				Error  error
			}{
				{
					Output: &dynamodb.ScanOutput{
						Items: []map[string]types.AttributeValue{
							testutil.PrayerItem(model.Prayer{
								IntercessorPhone: "848d9497e7d3cf7cc9bd997f44089967",
								Request:          "Please pray for me ...",
								Requestor: model.Member{
									Name:        "John Doe1",
									Phone:       testutil.PhoneMember,
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
							testutil.PrayerItem(model.Prayer{
								IntercessorPhone: "c3a3c79412496c510609c3d5110fbf14",
								Request:          "Please pray for me too ...",
								Requestor: model.Member{
									Name:        "John Doe2",
									Phone:       "+14567890123",
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
							testutil.PrayerItem(model.Prayer{
								IntercessorPhone: "9d7158545d5423200bbad27f88d4950c",
								Request:          "Pray for me also! ...",
								Requestor: model.Member{
									Name:        "John Doe3",
									Phone:       "+18901234567",
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
						},
					},
					Error: nil,
				},
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.IntercessorPhonesItem(
					testutil.PhoneIntercessor, testutil.PhoneAlt1, testutil.PhoneAlt2,
					testutil.PhoneAlt3, "+15555555555", "+16666666666",
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
					PrayerCount:       25,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 50,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.IntercessorPhonesItem(
					testutil.PhoneIntercessor, testutil.PhoneAlt1, testutil.PhoneAlt2,
					testutil.PhoneAlt3, "+15555555555", "+16666666666",
				),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             testutil.PhoneAlt2,
					PrayerCount:       55,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 120,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor4",
					Phone:             testutil.PhoneAlt3,
					PrayerCount:       8,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 1000,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.IntercessorPhonesItem(
					testutil.PhoneIntercessor, testutil.PhoneAlt1, testutil.PhoneAlt2,
					testutil.PhoneAlt3, "+15555555555", "+16666666666",
				),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor5",
					Phone:             "+15555555555",
					PrayerCount:       88,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 89,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor6",
					Phone:             "+16666666666",
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 5555,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             "+11111111111",
					PrayerCount:       2,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             "+12222222222",
					PrayerCount:       26,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 50,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             "+13333333333",
					PrayerCount:       56,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 120,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor4",
					Phone:             "+14444444444",
					PrayerCount:       9,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 1000,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor5",
					Phone:             "+15555555555",
					PrayerCount:       89,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 89,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor6",
					Phone:             "+16666666666",
					PrayerCount:       2,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5555,
				},
			},

			ExpectedPrayers: []model.Prayer{
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             "+11111111111",
						PrayerCount:       2,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+11111111111",
					Request:          "Please pray for me ...",
					Requestor: model.Member{
						Name:        "John Doe1",
						Phone:       "+11234567890",
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor2",
						Phone:             "+12222222222",
						PrayerCount:       26,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 50,
					},
					IntercessorPhone: "+12222222222",
					Request:          "Please pray for me ...",
					Requestor: model.Member{
						Name:        "John Doe1",
						Phone:       "+11234567890",
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor3",
						Phone:             "+13333333333",
						PrayerCount:       56,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 120,
					},
					IntercessorPhone: "+13333333333",
					Request:          "Please pray for me too ...",
					Requestor: model.Member{
						Name:        "John Doe2",
						Phone:       "+14567890123",
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor4",
						Phone:             "+14444444444",
						PrayerCount:       9,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 1000,
					},
					IntercessorPhone: "+14444444444",
					Request:          "Please pray for me too ...",
					Requestor: model.Member{
						Name:        "John Doe2",
						Phone:       "+14567890123",
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor5",
						Phone:             "+15555555555",
						PrayerCount:       89,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 89,
					},
					IntercessorPhone: "+15555555555",
					Request:          "Pray for me also! ...",
					Requestor: model.Member{
						Name:        "John Doe3",
						Phone:       "+18901234567",
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor6",
						Phone:             "+16666666666",
						PrayerCount:       2,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5555,
					},
					IntercessorPhone: "+16666666666",
					Request:          "Pray for me also! ...",
					Requestor: model.Member{
						Name:        "John Doe3",
						Phone:       "+18901234567",
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
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
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+13333333333",
				},
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+14444444444",
				},
				{
					Body:  messaging.MsgPrayerAssigned,
					Phone: "+14567890123",
				},
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+15555555555",
				},
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+16666666666",
				},
				{
					Body:  messaging.MsgPrayerAssigned,
					Phone: "+18901234567",
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   "848d9497e7d3cf7cc9bd997f44089967",
					Table: "QueuedPrayer",
				},
				{
					Key:   "c3a3c79412496c510609c3d5110fbf14",
					Table: "QueuedPrayer",
				},
				{
					Key:   "9d7158545d5423200bbad27f88d4950c",
					Table: "QueuedPrayer",
				},
			},

			ExpectedGetItemCalls:    15,
			ExpectedPutItemCalls:    12,
			ExpectedDeleteItemCalls: 3,
			ExpectedScanCalls:       1,
			ExpectedSendTextCalls:   9,
		},
		{
			Description: "2 queued prayers but only 1 intercessor",

			MockScanResults: []struct {
				Output *dynamodb.ScanOutput
				Error  error
			}{
				{
					Output: &dynamodb.ScanOutput{
						Items: []map[string]types.AttributeValue{
							testutil.PrayerItem(model.Prayer{
								IntercessorPhone: "848d9497e7d3cf7cc9bd997f44089967",
								Request:          "Please pray me ...",
								Requestor: model.Member{
									Name:        "John Doe1",
									Phone:       testutil.PhoneMember,
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
							testutil.PrayerItem(model.Prayer{
								IntercessorPhone: "c3a3c79412496c510609c3d5110fbf14",
								Request:          "Please pray me too ...",
								Requestor: model.Member{
									Name:        "John Doe2",
									Phone:       "+14567890123",
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
						},
					},
					Error: nil,
				},
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor),
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
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor),
				testutil.MemberItem(model.Member{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             testutil.PhoneIntercessor,
					PrayerCount:       2,
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
						PrayerCount:       2,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: testutil.PhoneIntercessor,
					Request:          "Please pray me ..",
					Requestor: model.Member{
						Name:        "John Doe1",
						Phone:       testutil.PhoneMember,
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				}),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor1",
					Phone:             "+11111111111",
					PrayerCount:       2,
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
						Phone:             "+11111111111",
						PrayerCount:       2,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+11111111111",
					Request:          "Please pray me ...",
					Requestor: model.Member{
						Name:        "John Doe1",
						Phone:       "+11234567890",
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+11111111111",
				},
				{
					Body:  messaging.MsgPrayerAssigned,
					Phone: "+11234567890",
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   "848d9497e7d3cf7cc9bd997f44089967",
					Table: "QueuedPrayer",
				},
			},

			ExpectedGetItemCalls:    6,
			ExpectedPutItemCalls:    2,
			ExpectedDeleteItemCalls: 1,
			ExpectedScanCalls:       1,
			ExpectedSendTextCalls:   2,
		},
		{
			Description: "1 queued prayer and 2 intercessors, but none of them are available due to maxed out prayer" +
				"counters",

			MockScanResults: []struct {
				Output *dynamodb.ScanOutput
				Error  error
			}{
				{
					Output: &dynamodb.ScanOutput{
						Items: []map[string]types.AttributeValue{
							testutil.PrayerItem(model.Prayer{
								IntercessorPhone: "848d9497e7d3cf7cc9bd997f44089967",
								Request:          "Please pray me ...",
								Requestor: model.Member{
									Name:        "John Doe1",
									Phone:       testutil.PhoneMember,
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
						},
					},
					Error: nil,
				},
			},

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
					PrayerCount:       50,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
					WeeklyPrayerLimit: 50,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
			},

			ExpectedGetItemCalls: 5,
			ExpectedScanCalls:    1,
		},
		{
			Description: "1 queued prayer and 2 intercessors, with only 1 intercessor available because the other" +
				"intercessor has an active prayer",

			MockScanResults: []struct {
				Output *dynamodb.ScanOutput
				Error  error
			}{
				{
					Output: &dynamodb.ScanOutput{
						Items: []map[string]types.AttributeValue{
							testutil.PrayerItem(model.Prayer{
								IntercessorPhone: "848d9497e7d3cf7cc9bd997f44089967",
								Request:          "Please pray me ...",
								Requestor: model.Member{
									Name:        "John Doe1",
									Phone:       testutil.PhoneMember,
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
						},
					},
					Error: nil,
				},
			},

			MockGetItemResults: []testutil.GetItemResult{
				testutil.IntercessorPhonesItem(testutil.PhoneIntercessor, testutil.PhoneAlt1),
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
					PrayerCount:       50,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "2024-12-01T01:00:00Z",
					WeeklyPrayerLimit: 50,
				}),
				// Prayer empty get response because there are no active prayers for this intercessor.
				testutil.EmptyGetResult(),
			},

			ExpectedMembers: []model.Member{
				{
					Intercessor:       true,
					Name:              "Intercessor2",
					Phone:             "+12222222222",
					PrayerCount:       1,
					SetupStage:        model.SignUpStepFinal,
					SetupStatus:       model.SetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 50,
				},
			},

			ExpectedPrayers: []model.Prayer{
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor2",
						Phone:             "+12222222222",
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 50,
					},
					IntercessorPhone: "+12222222222",
					Request:          "Please pray me ...",
					Requestor: model.Member{
						Name:        "John Doe1",
						Phone:       "+11234567890",
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerIntro,
					Phone: "+12222222222",
				},
				{
					Body:  messaging.MsgPrayerAssigned,
					Phone: "+11234567890",
				},
			},

			ExpectedDeleteItems: []struct {
				Key   string
				Table string
			}{
				{
					Key:   "848d9497e7d3cf7cc9bd997f44089967",
					Table: "QueuedPrayer",
				},
			},

			ExpectedGetItemCalls:    5,
			ExpectedPutItemCalls:    2,
			ExpectedDeleteItemCalls: 1,
			ExpectedScanCalls:       1,
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
				// Handles failures for error mocks.
				if err := svc.AssignQueuedPrayers(ctx); err == nil {
					t.Fatalf("expected error, got nil")
				}
				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases.
				if err := svc.AssignQueuedPrayers(ctx); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				test.RunAllCommonTests(ddbMock, txtMock, t, tc)
			}
		})
	}
}

func TestRemindActiveIntercessors(t *testing.T) {
	testCases := []test.Case{
		{
			Description: "3 active prayers and all 3 need prayer reminder text messages",

			MockScanResults: []struct {
				Output *dynamodb.ScanOutput
				Error  error
			}{
				{
					Output: &dynamodb.ScanOutput{
						Items: []map[string]types.AttributeValue{
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
								ReminderCount:    1,
								ReminderDate:     "2024-12-01T01:00:00Z",
								Request:          "Please pray for me ...",
								Requestor: model.Member{
									Name:        "John Doe1",
									Phone:       testutil.PhoneMember,
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
							testutil.PrayerItem(model.Prayer{
								Intercessor: model.Member{
									Intercessor:       true,
									Name:              "Intercessor2",
									Phone:             testutil.PhoneAlt1,
									PrayerCount:       1,
									SetupStage:        model.SignUpStepFinal,
									SetupStatus:       model.SetupComplete,
									WeeklyPrayerDate:  "dummy date",
									WeeklyPrayerLimit: 5,
								},
								IntercessorPhone: testutil.PhoneAlt1,
								ReminderCount:    1,
								ReminderDate:     "2024-12-01T01:00:00Z",
								Request:          "Please pray for me too ...",
								Requestor: model.Member{
									Name:        "John Doe2",
									Phone:       "+14567890123",
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
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
								ReminderCount:    1,
								ReminderDate:     "2024-12-01T01:00:00Z",
								Request:          "Pray for me also! ...",
								Requestor: model.Member{
									Name:        "John Doe3",
									Phone:       "+18901234567",
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
						},
					},
					Error: nil,
				},
			},

			ExpectedPrayers: []model.Prayer{
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             "+11111111111",
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+11111111111",
					ReminderCount:    2,
					ReminderDate:     "date changed",
					Request:          "Please pray for me ...",
					Requestor: model.Member{
						Name:        "John Doe1",
						Phone:       "+11234567890",
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor2",
						Phone:             "+12222222222",
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+12222222222",
					ReminderCount:    2,
					ReminderDate:     "date changed",
					Request:          "Please pray for me too ...",
					Requestor: model.Member{
						Name:        "John Doe2",
						Phone:       "+14567890123",
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor3",
						Phone:             "+13333333333",
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+13333333333",
					ReminderCount:    2,
					ReminderDate:     "date changed",
					Request:          "Pray for me also! ...",
					Requestor: model.Member{
						Name:        "John Doe3",
						Phone:       "+18901234567",
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerReminder,
					Phone: "+11111111111",
				},
				{
					Body:  messaging.MsgPrayerReminder,
					Phone: "+12222222222",
				},
				{
					Body:  messaging.MsgPrayerReminder,
					Phone: "+13333333333",
				},
			},

			ExpectedPutItemCalls:  3,
			ExpectedScanCalls:     1,
			ExpectedSendTextCalls: 3,
		},
		{
			Description: "2 active prayers and only 1 needs prayer reminder text messages",

			MockScanResults: []struct {
				Output *dynamodb.ScanOutput
				Error  error
			}{
				{
					Output: &dynamodb.ScanOutput{
						Items: []map[string]types.AttributeValue{
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
								ReminderCount:    1,
								ReminderDate:     time.Now().Format(time.RFC3339),
								Request:          "Please pray for me ...",
								Requestor: model.Member{
									Name:        "John Doe1",
									Phone:       testutil.PhoneMember,
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
							testutil.PrayerItem(model.Prayer{
								Intercessor: model.Member{
									Intercessor:       true,
									Name:              "Intercessor2",
									Phone:             testutil.PhoneAlt1,
									PrayerCount:       1,
									SetupStage:        model.SignUpStepFinal,
									SetupStatus:       model.SetupComplete,
									WeeklyPrayerDate:  "dummy date",
									WeeklyPrayerLimit: 5,
								},
								IntercessorPhone: testutil.PhoneAlt1,
								ReminderCount:    1,
								ReminderDate:     "2024-12-01T01:00:00Z",
								Request:          "Please pray for me too ...",
								Requestor: model.Member{
									Name:        "John Doe2",
									Phone:       "+14567890123",
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
						},
					},
					Error: nil,
				},
			},

			ExpectedPrayers: []model.Prayer{
				{
					Intercessor: model.Member{
						Intercessor:       true,
						Name:              "Intercessor2",
						Phone:             "+12222222222",
						PrayerCount:       1,
						SetupStage:        model.SignUpStepFinal,
						SetupStatus:       model.SetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+12222222222",
					ReminderCount:    2,
					ReminderDate:     "date changed",
					Request:          "Please pray for me too ...",
					Requestor: model.Member{
						Name:        "John Doe2",
						Phone:       "+14567890123",
						SetupStage:  model.SignUpStepFinal,
						SetupStatus: model.SetupComplete,
					},
				},
			},

			ExpectedTexts: []messaging.TextMessage{
				{
					Body:  messaging.MsgPrayerReminder,
					Phone: "+12222222222",
				},
			},

			ExpectedPutItemCalls:  1,
			ExpectedScanCalls:     1,
			ExpectedSendTextCalls: 1,
		},
		{
			Description: "2 active prayers and none prayer reminder text messages - nothing gets updated",

			MockScanResults: []struct {
				Output *dynamodb.ScanOutput
				Error  error
			}{
				{
					Output: &dynamodb.ScanOutput{
						Items: []map[string]types.AttributeValue{
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
								ReminderCount:    1,
								ReminderDate:     time.Now().Format(time.RFC3339),
								Request:          "Please pray for me ...",
								Requestor: model.Member{
									Name:        "John Doe1",
									Phone:       testutil.PhoneMember,
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
							testutil.PrayerItem(model.Prayer{
								Intercessor: model.Member{
									Intercessor:       true,
									Name:              "Intercessor2",
									Phone:             testutil.PhoneAlt1,
									PrayerCount:       1,
									SetupStage:        model.SignUpStepFinal,
									SetupStatus:       model.SetupComplete,
									WeeklyPrayerDate:  "dummy date",
									WeeklyPrayerLimit: 5,
								},
								IntercessorPhone: testutil.PhoneAlt1,
								ReminderCount:    1,
								ReminderDate:     time.Now().Format(time.RFC3339),
								Request:          "Please pray for me too ...",
								Requestor: model.Member{
									Name:        "John Doe2",
									Phone:       "+14567890123",
									SetupStage:  model.SignUpStepFinal,
									SetupStatus: model.SetupComplete,
								},
							}).Output.Item,
						},
					},
					Error: nil,
				},
			},

			ExpectedScanCalls: 1,
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
				if err := svc.RemindActiveIntercessors(ctx); err == nil {
					t.Fatalf("expected error, got nil")
				}
				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases.
				if err := svc.RemindActiveIntercessors(ctx); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				test.RunAllCommonTests(ddbMock, txtMock, t, tc)
			}
		})
	}
}
