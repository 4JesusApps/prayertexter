package prayertexter

import (
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestSendMessage(t *testing.T) {
	txtBody := "test message"
	expectedTexts := []TextMessage{
		{
			Body:  msgPre + txtBody + "\n\n" + msgPost,
			Phone: "123-456-7890",
		},
	}

	member := Member{
		Intercessor: false,
		Name:        "John Doe",
		Phone:       "123-456-7890",
		SetupStage:  99,
		SetupStatus: "completed",
	}

	txtMock := &MockTextSender{}
	ddbMock := &MockDDBConnecter{}
	if err := member.sendMessage(ddbMock, txtMock, txtBody); err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if !reflect.DeepEqual(expectedTexts, txtMock.SendTextInputs) {
		t.Errorf("expected TextMessage %v, got %v", expectedTexts, txtMock.SendTextInputs)
	}
}

func TestCheckIfActiveMember(t *testing.T) {
	mockGetItemResults := []struct {
		Output *dynamodb.GetItemOutput
		Error  error
	}{
		{
			// This is an empty ddb response, meaning that the key does not exist in ddb
			// we are simulating the member not active with this empty response
			Output: &dynamodb.GetItemOutput{},
			Error:  nil,
		},
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
			Output: nil,
			Error:  errors.New("random failure"),
		},
	}

	ddbMock := &MockDDBConnecter{}
	ddbMock.GetItemResults = mockGetItemResults

	isActive, err := isMemberActive(ddbMock, "123-456-7890")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	} else if isActive {
		t.Errorf("expected return of false (inactive member), got %v", isActive)
	}

	isActive, err = isMemberActive(ddbMock, "123-456-7890")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	} else if !isActive {
		t.Errorf("expected return of true (active member), got %v", isActive)
	}

	_, err = isMemberActive(ddbMock, "123-456-7890")
	if err == nil {
		t.Errorf("expected error, got %v", err)
	}
}
