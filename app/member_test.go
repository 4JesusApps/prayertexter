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
	expectedText := TextMessage{
		Body:  msgPre + txtBody + "\n\n" + msgPost,
		Phone: "+11234567890",
	}

	member := Member{
		Intercessor: false,
		Name:        "John Doe",
		Phone:       "+11234567890",
		SetupStage:  99,
		SetupStatus: "completed",
	}

	txtMock := &MockTextSender{}
	if err := member.sendMessage(txtMock, txtBody); err != nil {
		t.Errorf("unexpected error %v", err)
	}

	actualText := TextMessage{
		Body:  *txtMock.SendTextInputs[0].MessageBody,
		Phone: *txtMock.SendTextInputs[0].DestinationPhoneNumber,
	}

	if !reflect.DeepEqual(expectedText, actualText) {
		t.Errorf("expected TextMessage %v, got %v", expectedText, actualText)
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
					"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
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

	isActive, err := isMemberActive(ddbMock, "+11234567890")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	} else if isActive {
		t.Errorf("expected return of false (inactive member), got %v", isActive)
	}

	isActive, err = isMemberActive(ddbMock, "+11234567890")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	} else if !isActive {
		t.Errorf("expected return of true (active member), got %v", isActive)
	}

	_, err = isMemberActive(ddbMock, "+11234567890")
	if err == nil {
		t.Errorf("expected error, got %v", err)
	}
}
