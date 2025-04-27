package statecontroller_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/statecontroller"
	"github.com/4JesusApps/prayertexter/internal/test"
	"github.com/4JesusApps/prayertexter/internal/test/mock"
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
							{
								"IntercessorPhone": &types.AttributeValueMemberS{Value: "848d9497e7d3cf7cc9bd997f44089967"},
								"Request":          &types.AttributeValueMemberS{Value: "Please pray for me ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe1"},
										"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
							{
								"IntercessorPhone": &types.AttributeValueMemberS{Value: "c3a3c79412496c510609c3d5110fbf14"},
								"Request":          &types.AttributeValueMemberS{Value: "Please pray for me too ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe2"},
										"Phone":       &types.AttributeValueMemberS{Value: "+14567890123"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
							{
								"IntercessorPhone": &types.AttributeValueMemberS{Value: "9d7158545d5423200bbad27f88d4950c"},
								"Request":          &types.AttributeValueMemberS{Value: "Pray for me also! ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe3"},
										"Phone":       &types.AttributeValueMemberS{Value: "+18901234567"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
						},
					},
					Error: nil,
				},
			},

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
								&types.AttributeValueMemberS{Value: "+16666666666"},
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
							"PrayerCount":       &types.AttributeValueMemberN{Value: "25"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "50"},
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
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
								&types.AttributeValueMemberS{Value: "+14444444444"},
								&types.AttributeValueMemberS{Value: "+15555555555"},
								&types.AttributeValueMemberS{Value: "+16666666666"},
							}},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor3"},
							"Phone":             &types.AttributeValueMemberS{Value: "+13333333333"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "55"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "120"},
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
							"PrayerCount":       &types.AttributeValueMemberN{Value: "8"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "1000"},
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
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
								&types.AttributeValueMemberS{Value: "+12222222222"},
								&types.AttributeValueMemberS{Value: "+13333333333"},
								&types.AttributeValueMemberS{Value: "+14444444444"},
								&types.AttributeValueMemberS{Value: "+15555555555"},
								&types.AttributeValueMemberS{Value: "+16666666666"},
							}},
						},
					},
					Error: nil,
				},
				{
					Output: &dynamodb.GetItemOutput{
						Item: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor5"},
							"Phone":             &types.AttributeValueMemberS{Value: "+15555555555"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "88"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "89"},
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
							"Name":              &types.AttributeValueMemberS{Value: "Intercessor6"},
							"Phone":             &types.AttributeValueMemberS{Value: "+16666666666"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5555"},
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
					Name:              "Intercessor2",
					Phone:             "+12222222222",
					PrayerCount:       26,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 50,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor3",
					Phone:             "+13333333333",
					PrayerCount:       56,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 120,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor4",
					Phone:             "+14444444444",
					PrayerCount:       9,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 1000,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor5",
					Phone:             "+15555555555",
					PrayerCount:       89,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 89,
				},
				{
					Intercessor:       true,
					Name:              "Intercessor6",
					Phone:             "+16666666666",
					PrayerCount:       2,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 5555,
				},
			},

			ExpectedPrayers: []object.Prayer{
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor1",
						Phone:             "+11111111111",
						PrayerCount:       2,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+11111111111",
					Request:          "Please pray for me ...",
					Requestor: object.Member{
						Name:        "John Doe1",
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
						PrayerCount:       26,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 50,
					},
					IntercessorPhone: "+12222222222",
					Request:          "Please pray for me ...",
					Requestor: object.Member{
						Name:        "John Doe1",
						Phone:       "+11234567890",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor3",
						Phone:             "+13333333333",
						PrayerCount:       56,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 120,
					},
					IntercessorPhone: "+13333333333",
					Request:          "Please pray for me too ...",
					Requestor: object.Member{
						Name:        "John Doe2",
						Phone:       "+14567890123",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor4",
						Phone:             "+14444444444",
						PrayerCount:       9,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 1000,
					},
					IntercessorPhone: "+14444444444",
					Request:          "Please pray for me too ...",
					Requestor: object.Member{
						Name:        "John Doe2",
						Phone:       "+14567890123",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor5",
						Phone:             "+15555555555",
						PrayerCount:       89,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 89,
					},
					IntercessorPhone: "+15555555555",
					Request:          "Pray for me also! ...",
					Requestor: object.Member{
						Name:        "John Doe3",
						Phone:       "+18901234567",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor6",
						Phone:             "+16666666666",
						PrayerCount:       2,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5555,
					},
					IntercessorPhone: "+16666666666",
					Request:          "Pray for me also! ...",
					Requestor: object.Member{
						Name:        "John Doe3",
						Phone:       "+18901234567",
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
					Table: object.DefaultQueuedPrayersTable,
				},
				{
					Key:   "c3a3c79412496c510609c3d5110fbf14",
					Table: object.DefaultQueuedPrayersTable,
				},
				{
					Key:   "9d7158545d5423200bbad27f88d4950c",
					Table: object.DefaultQueuedPrayersTable,
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
							{
								"IntercessorPhone": &types.AttributeValueMemberS{Value: "848d9497e7d3cf7cc9bd997f44089967"},
								"Request":          &types.AttributeValueMemberS{Value: "Please pray me ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe1"},
										"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
							{
								"IntercessorPhone": &types.AttributeValueMemberS{Value: "c3a3c79412496c510609c3d5110fbf14"},
								"Request":          &types.AttributeValueMemberS{Value: "Please pray me too ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe2"},
										"Phone":       &types.AttributeValueMemberS{Value: "+14567890123"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
						},
					},
					Error: nil,
				},
			},

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
							"Key": &types.AttributeValueMemberS{Value: object.IntercessorPhonesKeyValue},
							"Phones": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberS{Value: "+11111111111"},
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
							"PrayerCount":       &types.AttributeValueMemberN{Value: "2"},
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
									"PrayerCount":       &types.AttributeValueMemberN{Value: "2"},
									"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
									"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
									"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
								},
							},
							"IntercessorPhone": &types.AttributeValueMemberS{Value: "+11111111111"},
							"Request":          &types.AttributeValueMemberS{Value: "Please pray me .."},
							"Requestor": &types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
									"Name":        &types.AttributeValueMemberS{Value: "John Doe1"},
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
					Name:              "Intercessor1",
					Phone:             "+11111111111",
					PrayerCount:       2,
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
						PrayerCount:       2,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+11111111111",
					Request:          "Please pray me ...",
					Requestor: object.Member{
						Name:        "John Doe1",
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
					Table: object.DefaultQueuedPrayersTable,
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
							{
								"IntercessorPhone": &types.AttributeValueMemberS{Value: "848d9497e7d3cf7cc9bd997f44089967"},
								"Request":          &types.AttributeValueMemberS{Value: "Please pray me ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe1"},
										"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
						},
					},
					Error: nil,
				},
			},

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
							"PrayerCount":       &types.AttributeValueMemberN{Value: "50"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "50"},
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
							{
								"IntercessorPhone": &types.AttributeValueMemberS{Value: "848d9497e7d3cf7cc9bd997f44089967"},
								"Request":          &types.AttributeValueMemberS{Value: "Please pray me ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe1"},
										"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
						},
					},
					Error: nil,
				},
			},

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
							"PrayerCount":       &types.AttributeValueMemberN{Value: "50"},
							"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
							"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "50"},
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
					Name:              "Intercessor2",
					Phone:             "+12222222222",
					PrayerCount:       1,
					SetupStage:        object.MemberSignUpStepFinal,
					SetupStatus:       object.MemberSetupComplete,
					WeeklyPrayerDate:  "dummy date/time",
					WeeklyPrayerLimit: 50,
				},
			},

			ExpectedPrayers: []object.Prayer{
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor2",
						Phone:             "+12222222222",
						PrayerCount:       1,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 50,
					},
					IntercessorPhone: "+12222222222",
					Request:          "Please pray me ...",
					Requestor: object.Member{
						Name:        "John Doe1",
						Phone:       "+11234567890",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
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
					Table: object.DefaultQueuedPrayersTable,
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
			config.InitConfig()

			if tc.ExpectedError {
				// Handles failures for error mocks.
				if err := statecontroller.AssignQueuedPrayers(ctx, ddbMock, txtMock); err == nil {
					t.Fatalf("expected error, got nil")
				}
				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases.
				if err := statecontroller.AssignQueuedPrayers(ctx, ddbMock, txtMock); err != nil {
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
							{
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
								"ReminderCount":    &types.AttributeValueMemberN{Value: "1"},
								"ReminderDate":     &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
								"Request":          &types.AttributeValueMemberS{Value: "Please pray for me ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe1"},
										"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
							{
								"Intercessor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
										"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
										"Phone":             &types.AttributeValueMemberS{Value: "+12222222222"},
										"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
										"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
										"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
										"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
									},
								},
								"IntercessorPhone": &types.AttributeValueMemberS{Value: "+12222222222"},
								"ReminderCount":    &types.AttributeValueMemberN{Value: "1"},
								"ReminderDate":     &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
								"Request":          &types.AttributeValueMemberS{Value: "Please pray for me too ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe2"},
										"Phone":       &types.AttributeValueMemberS{Value: "+14567890123"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
							{
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
								"ReminderCount":    &types.AttributeValueMemberN{Value: "1"},
								"ReminderDate":     &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
								"Request":          &types.AttributeValueMemberS{Value: "Pray for me also! ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe3"},
										"Phone":       &types.AttributeValueMemberS{Value: "+18901234567"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
						},
					},
					Error: nil,
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
					ReminderCount:    2,
					ReminderDate:     "date changed",
					Request:          "Please pray for me ...",
					Requestor: object.Member{
						Name:        "John Doe1",
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
					ReminderCount:    2,
					ReminderDate:     "date changed",
					Request:          "Please pray for me too ...",
					Requestor: object.Member{
						Name:        "John Doe2",
						Phone:       "+14567890123",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
					},
				},
				{
					Intercessor: object.Member{
						Intercessor:       true,
						Name:              "Intercessor3",
						Phone:             "+13333333333",
						PrayerCount:       1,
						SetupStage:        object.MemberSignUpStepFinal,
						SetupStatus:       object.MemberSetupComplete,
						WeeklyPrayerDate:  "dummy date/time",
						WeeklyPrayerLimit: 5,
					},
					IntercessorPhone: "+13333333333",
					ReminderCount:    2,
					ReminderDate:     "date changed",
					Request:          "Pray for me also! ...",
					Requestor: object.Member{
						Name:        "John Doe3",
						Phone:       "+18901234567",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
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
							{
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
								"ReminderCount":    &types.AttributeValueMemberN{Value: "1"},
								"ReminderDate":     &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
								"Request":          &types.AttributeValueMemberS{Value: "Please pray for me ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe1"},
										"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
							{
								"Intercessor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
										"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
										"Phone":             &types.AttributeValueMemberS{Value: "+12222222222"},
										"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
										"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
										"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
										"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
									},
								},
								"IntercessorPhone": &types.AttributeValueMemberS{Value: "+12222222222"},
								"ReminderCount":    &types.AttributeValueMemberN{Value: "1"},
								"ReminderDate":     &types.AttributeValueMemberS{Value: "2024-12-01T01:00:00Z"},
								"Request":          &types.AttributeValueMemberS{Value: "Please pray for me too ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe2"},
										"Phone":       &types.AttributeValueMemberS{Value: "+14567890123"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
						},
					},
					Error: nil,
				},
			},

			ExpectedPrayers: []object.Prayer{
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
					ReminderCount:    2,
					ReminderDate:     "date changed",
					Request:          "Please pray for me too ...",
					Requestor: object.Member{
						Name:        "John Doe2",
						Phone:       "+14567890123",
						SetupStage:  object.MemberSignUpStepFinal,
						SetupStatus: object.MemberSetupComplete,
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
							{
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
								"ReminderCount":    &types.AttributeValueMemberN{Value: "1"},
								"ReminderDate":     &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
								"Request":          &types.AttributeValueMemberS{Value: "Please pray for me ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe1"},
										"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
							{
								"Intercessor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor":       &types.AttributeValueMemberBOOL{Value: true},
										"Name":              &types.AttributeValueMemberS{Value: "Intercessor2"},
										"Phone":             &types.AttributeValueMemberS{Value: "+12222222222"},
										"PrayerCount":       &types.AttributeValueMemberN{Value: "1"},
										"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus":       &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
										"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "dummy date"},
										"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
									},
								},
								"IntercessorPhone": &types.AttributeValueMemberS{Value: "+12222222222"},
								"ReminderCount":    &types.AttributeValueMemberN{Value: "1"},
								"ReminderDate":     &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
								"Request":          &types.AttributeValueMemberS{Value: "Please pray for me too ..."},
								"Requestor": &types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Intercessor": &types.AttributeValueMemberBOOL{Value: false},
										"Name":        &types.AttributeValueMemberS{Value: "John Doe2"},
										"Phone":       &types.AttributeValueMemberS{Value: "+14567890123"},
										"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
										"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
									},
								},
							},
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
			config.InitConfig()

			if tc.ExpectedError {
				// Handles failures for error mocks.
				if err := statecontroller.RemindActiveIntercessors(ctx, ddbMock, txtMock); err == nil {
					t.Fatalf("expected error, got nil")
				}
				test.ValidateNumMethodCalls(ddbMock, txtMock, t, tc)
			} else {
				// Handles success test cases.
				if err := statecontroller.RemindActiveIntercessors(ctx, ddbMock, txtMock); err != nil {
					t.Fatalf("unexpected error starting MainFlow: %v", err)
				}

				test.RunAllCommonTests(ddbMock, txtMock, t, tc)
			}
		})
	}
}
