package prayertexter

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestGetPrayerTable(t *testing.T) {
	table := getPrayerTable(true)
	if table != prayersQueueTable {
		t.Errorf("expected prayer table to be %v, got %v", prayersQueueTable, table)
	}

	table = getPrayerTable(false)
	if table != activePrayersTable {
		t.Errorf("expected prayer table to be %v, got %v", activePrayersTable, table)
	}
}

func TestCheckIfActivePrayer(t *testing.T) {
	mockGetItemResults := []struct {
		Output *dynamodb.GetItemOutput
		Error  error
	}{
		{
			// This is an empty ddb response, meaning that the key does not exist in ddb
			// we are simulating the prayer not active with this empty response
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
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: "2025-02-13T23:54:01Z"},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "5"},
						},
					},
					"IntercessorPhone": &types.AttributeValueMemberS{Value: "+11111111111"},
					"Request":          &types.AttributeValueMemberS{Value: "I need prayer for..."},
					"Requestor": &types.AttributeValueMemberM{
						Value: map[string]types.AttributeValue{
							"Intercessor":       &types.AttributeValueMemberBOOL{Value: false},
							"Name":              &types.AttributeValueMemberS{Value: "John Doe"},
							"Phone":             &types.AttributeValueMemberS{Value: "+11234567890"},
							"PrayerCount":       &types.AttributeValueMemberN{Value: "0"},
							"SetupStage":        &types.AttributeValueMemberN{Value: "99"},
							"SetupStatus":       &types.AttributeValueMemberS{Value: "completed"},
							"WeeklyPrayerDate":  &types.AttributeValueMemberS{Value: ""},
							"WeeklyPrayerLimit": &types.AttributeValueMemberN{Value: "0"},
						},
					},
				},
			},
			Error: nil,
		},
		{
			Output: nil,
			Error:  errors.New("random failure"),
		},
	}

	ddbMock := &MockDDBConnecter{}
	ddbMock.GetItemResults = mockGetItemResults

	isActive, err := isPrayerActive(ddbMock, "+11111111111")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	} else if isActive {
		t.Errorf("expected return of false (inactive prayer), got %v", isActive)
	}

	isActive, err = isPrayerActive(ddbMock, "+11111111111")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	} else if !isActive {
		t.Errorf("expected return of true (active prayer), got %v", isActive)
	}

	_, err = isPrayerActive(ddbMock, "+11111111111")
	if err == nil {
		t.Errorf("expected error, got %v", err)
	}
}
