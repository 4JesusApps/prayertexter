package testutil

import (
	"strconv"

	"github.com/4JesusApps/prayertexter/internal/model"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Phone constants for use in test fixtures.
const (
	PhoneMember      = "+11234567890"
	PhoneIntercessor = "+11111111111"
	PhoneAdmin       = "+17777777777"
	PhoneAlt1        = "+12222222222"
	PhoneAlt2        = "+13333333333"
	PhoneAlt3        = "+14444444444"
	PhoneAlt4        = "+18888888888"
)

// GetItemResult matches the anonymous struct type used in test.Case.MockGetItemResults.
type GetItemResult struct {
	Output *dynamodb.GetItemOutput
	Error  error
}

// CompleteMember returns a Member with setup complete (non-admin, non-intercessor).
func CompleteMember() model.Member {
	return model.Member{
		Name:              "John Doe",
		Phone:             PhoneMember,
		SetupStage:        model.SignUpStepFinal,
		SetupStatus:       model.SetupComplete,
		WeeklyPrayerDate:  "",
		WeeklyPrayerLimit: 0,
	}
}

// CompleteIntercessor returns a Member with setup complete and Intercessor=true.
func CompleteIntercessor() model.Member {
	return model.Member{
		Intercessor:       true,
		Name:              "Intercessor1",
		Phone:             PhoneIntercessor,
		PrayerCount:       1,
		SetupStage:        model.SignUpStepFinal,
		SetupStatus:       model.SetupComplete,
		WeeklyPrayerDate:  "2025-02-16T23:54:01Z",
		WeeklyPrayerLimit: 5,
	}
}

// AdminMember returns a Member with setup complete and Administrator=true.
func AdminMember() model.Member {
	return model.Member{
		Administrator: true,
		Name:          "Admin User",
		Phone:         PhoneAdmin,
		SetupStage:    model.SignUpStepFinal,
		SetupStatus:   model.SetupComplete,
	}
}

// MemberItem returns a GetItemResult containing a DynamoDB representation of the given Member.
func MemberItem(m model.Member) GetItemResult {
	item := map[string]types.AttributeValue{
		"Administrator":     &types.AttributeValueMemberBOOL{Value: m.Administrator},
		"Intercessor":       &types.AttributeValueMemberBOOL{Value: m.Intercessor},
		"Name":              &types.AttributeValueMemberS{Value: m.Name},
		"Phone":             &types.AttributeValueMemberS{Value: m.Phone},
		"PrayerCount":       &types.AttributeValueMemberN{Value: strconv.Itoa(m.PrayerCount)},
		"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(m.SetupStage)},
		"SetupStatus":       &types.AttributeValueMemberS{Value: m.SetupStatus},
		"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: m.WeeklyPrayerDate},
		"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: strconv.Itoa(m.WeeklyPrayerLimit)},
	}

	return GetItemResult{
		Output: &dynamodb.GetItemOutput{Item: item},
	}
}

// PrayerItem returns a GetItemResult containing a DynamoDB representation of a Prayer.
func PrayerItem(p model.Prayer) GetItemResult {
	item := map[string]types.AttributeValue{
		"Intercessor":      memberAttributeMap(p.Intercessor),
		"IntercessorPhone": &types.AttributeValueMemberS{Value: p.IntercessorPhone},
		"ReminderCount":    &types.AttributeValueMemberN{Value: strconv.Itoa(p.ReminderCount)},
		"ReminderDate":     &types.AttributeValueMemberS{Value: p.ReminderDate},
		"Request":          &types.AttributeValueMemberS{Value: p.Request},
		"Requestor":        memberAttributeMap(p.Requestor),
	}

	return GetItemResult{
		Output: &dynamodb.GetItemOutput{Item: item},
	}
}

// IntercessorPhonesItem returns a GetItemResult containing a DynamoDB representation
// of IntercessorPhones with the given phone numbers.
func IntercessorPhonesItem(phones ...string) GetItemResult {
	phoneValues := make([]types.AttributeValue, len(phones))
	for i, p := range phones {
		phoneValues[i] = &types.AttributeValueMemberS{Value: p}
	}

	item := map[string]types.AttributeValue{
		"Key":    &types.AttributeValueMemberS{Value: model.IntercessorPhonesKeyValue},
		"Phones": &types.AttributeValueMemberL{Value: phoneValues},
	}

	return GetItemResult{
		Output: &dynamodb.GetItemOutput{Item: item},
	}
}

// BlockedPhonesItem returns a GetItemResult containing a DynamoDB representation
// of BlockedPhones with the given phone numbers.
func BlockedPhonesItem(phones ...string) GetItemResult {
	phoneValues := make([]types.AttributeValue, len(phones))
	for i, p := range phones {
		phoneValues[i] = &types.AttributeValueMemberS{Value: p}
	}

	item := map[string]types.AttributeValue{
		"Key":    &types.AttributeValueMemberS{Value: model.BlockedPhonesKeyValue},
		"Phones": &types.AttributeValueMemberL{Value: phoneValues},
	}

	return GetItemResult{
		Output: &dynamodb.GetItemOutput{Item: item},
	}
}

// EmptyGetResult returns a GetItemResult with an empty item, representing a "not found" response.
func EmptyGetResult() GetItemResult {
	return GetItemResult{
		Output: &dynamodb.GetItemOutput{},
	}
}

// memberAttributeMap converts a Member to a DynamoDB nested map attribute value.
func memberAttributeMap(m model.Member) *types.AttributeValueMemberM {
	return &types.AttributeValueMemberM{
		Value: map[string]types.AttributeValue{
			"Administrator":     &types.AttributeValueMemberBOOL{Value: m.Administrator},
			"Intercessor":       &types.AttributeValueMemberBOOL{Value: m.Intercessor},
			"Name":              &types.AttributeValueMemberS{Value: m.Name},
			"Phone":             &types.AttributeValueMemberS{Value: m.Phone},
			"PrayerCount":       &types.AttributeValueMemberN{Value: strconv.Itoa(m.PrayerCount)},
			"SetupStage":        &types.AttributeValueMemberN{Value: strconv.Itoa(m.SetupStage)},
			"SetupStatus":       &types.AttributeValueMemberS{Value: m.SetupStatus},
			"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: m.WeeklyPrayerDate},
			"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: strconv.Itoa(m.WeeklyPrayerLimit)},
		},
	}
}
