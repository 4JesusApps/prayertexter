package messaging_test

import (
	"context"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/test/mock"
	"github.com/aws/smithy-go"
)

func TestSendText(t *testing.T) {
	config.InitConfig()
	t.Run("send sms and verify inputs are correct", func(t *testing.T) {
		msg := messaging.TextMessage{
			Body:  "test text message",
			Phone: "+11234567890",
		}
		txtMock := &mock.TextSender{}
		ctx := context.Background()

		if err := messaging.SendText(ctx, txtMock, msg); err != nil {
			t.Errorf("unexpected error, %v", err)
		}

		receivedText := messaging.TextMessage{
			Body:  *txtMock.SendTextInputs[0].MessageBody,
			Phone: *txtMock.SendTextInputs[0].DestinationPhoneNumber,
		}

		msg.Body = messaging.MsgPre + msg.Body + "\n\n" + messaging.MsgPost

		if receivedText != msg {
			t.Errorf("expected txt %v, got %v", msg, receivedText)
		}

		if *txtMock.SendTextInputs[0].OriginationIdentity != messaging.DefaultPhonePool {
			t.Errorf("expected phone number %v, got %v", messaging.DefaultPhonePool,
				*txtMock.SendTextInputs[0].OriginationIdentity)
		}
	})
	t.Run("mock throttle message error to validate sms is retried 3 times after throttle error", func(t *testing.T) {
		// The number of retries should match the max attempts from the messaging package logic
		const wantRetries = 3

		txtMock := &mock.TextSender{}
		txtMock.SendTextResults = []struct {
			Error error
		}{
			{
				// smithy.GenericAPIError implements smithy.APIError
				Error: &smithy.GenericAPIError{
					Code:    "ThrottlingException",
					Message: "rate exceeded",
				},
			},
			{
				Error: &smithy.GenericAPIError{
					Code:    "ThrottlingException",
					Message: "rate exceeded",
				},
			},
			{
				Error: &smithy.GenericAPIError{
					Code:    "ThrottlingException",
					Message: "rate exceeded",
				},
			},
		}

		msg := messaging.TextMessage{
			Body:  "test text message",
			Phone: "+11234567890",
		}
		err := messaging.SendText(context.Background(), txtMock, msg)

		if err == nil {
			t.Fatal("expected an error after exhausting retries, got nil")
		}

		if txtMock.SendTextCalls != wantRetries {
			t.Errorf("expected SendTextMessage to be called %v times, got %v",
				wantRetries, txtMock.SendTextCalls)
		}
	})
}

func TestCheckProfanity(t *testing.T) {
	msg := messaging.TextMessage{Body: "test text message, no profanity"}

	t.Run("message does not have profanity", func(t *testing.T) {
		profanity := msg.CheckProfanity()
		if profanity != "" {
			t.Errorf("expected no profanity, got %v", profanity)
		}
	})

	t.Run("message has profanity", func(t *testing.T) {
		msg.Body = "this message contains profanity, sh!t!"
		profanity := msg.CheckProfanity()
		if profanity == "" {
			t.Errorf("expected profanity, got none (empty string): %v", profanity)
		}
	})

	t.Run("this should not detect profanity because spaces are added in between words", func(t *testing.T) {
		msg.Body = "sh it, fu ck"
		profanity := msg.CheckProfanity()
		if profanity != "" {
			t.Errorf("expected no profanity, got %v", profanity)
		}
	})
}

// func TestSendRealText(t *testing.T) {
// 	mem := Member{
// 		Phone: "+11111111111",
// 	}

// 	smsClnt, err := GetSmsClient()
// 	if err != nil {
// 		t.Errorf("unexpected error, %v", err)
// 	}

// 	if err := mem.sendMessage(smsClnt, "test text message"); err != nil {
// 		t.Errorf("unexpected error, %v", err)
// 	}
// }
