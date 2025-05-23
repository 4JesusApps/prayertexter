package object_test

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/test/mock"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestSendMessage(t *testing.T) {
	t.Run("send text and verify results", func(t *testing.T) {
		txtBody := "test message"
		expectedText := messaging.TextMessage{
			Body:  messaging.MsgPre + txtBody + "\n\n" + messaging.MsgPost,
			Phone: "+11234567890",
		}

		member := object.Member{
			Intercessor: false,
			Name:        "John Doe",
			Phone:       "+11234567890",
			SetupStage:  object.MemberSignUpStepFinal,
			SetupStatus: object.MemberSetupComplete,
		}

		txtMock := &mock.TextSender{}
		ctx := context.Background()
		if err := member.SendMessage(ctx, txtMock, txtBody); err != nil {
			t.Errorf("unexpected error %v", err)
		}

		actualText := messaging.TextMessage{
			Body:  *txtMock.SendTextInputs[0].MessageBody,
			Phone: *txtMock.SendTextInputs[0].DestinationPhoneNumber,
		}

		if !reflect.DeepEqual(expectedText, actualText) {
			t.Errorf("expected TextMessage %v, got %v", expectedText, actualText)
		}
	})
}

func TestIsMemberActive(t *testing.T) {
	mockGetItemResults := []struct {
		Output *dynamodb.GetItemOutput
		Error  error
	}{
		{
			// This is an empty ddb response. This simulating the member not active with this empty response, because
			// dynamodb will return empty data if the key is not present in the database.
			Output: &dynamodb.GetItemOutput{},
			Error:  nil,
		},
		{
			Output: &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"Name":        &types.AttributeValueMemberS{Value: "John Doe"},
					"Phone":       &types.AttributeValueMemberS{Value: "+11234567890"},
					"SetupStage":  &types.AttributeValueMemberN{Value: strconv.Itoa(object.MemberSignUpStepFinal)},
					"SetupStatus": &types.AttributeValueMemberS{Value: object.MemberSetupComplete},
				},
			},
			Error: nil,
		},
		{
			Output: nil,
			Error:  errors.New("random failure"),
		},
	}

	ddbMock := &mock.DDBConnecter{}
	ddbMock.GetItemResults = mockGetItemResults
	ctx := context.Background()

	t.Run("Member is not active", func(t *testing.T) {
		isActive, err := object.IsMemberActive(ctx, ddbMock, "+11234567890")
		if err != nil {
			t.Errorf("unexpected error %v", err)
		} else if isActive {
			t.Errorf("expected return of false (inactive member), got %v", isActive)
		}
	})

	t.Run("Member is active", func(t *testing.T) {
		isActive, err := object.IsMemberActive(ctx, ddbMock, "+11234567890")
		if err != nil {
			t.Errorf("unexpected error %v", err)
		} else if !isActive {
			t.Errorf("expected return of true (active member), got %v", isActive)
		}
	})
	t.Run("returns error on get Member dynamodb call", func(t *testing.T) {
		_, err := object.IsMemberActive(ctx, ddbMock, "+11234567890")
		if err == nil {
			t.Errorf("expected error, got %v", err)
		}
	})
}
