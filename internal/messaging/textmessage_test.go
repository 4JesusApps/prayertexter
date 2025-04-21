package messaging_test

import (
	"context"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/test/mock"
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
}

// func TestSendRealText(t *testing.T) {
// 	mem := Member{
// 		Phone: "+16572171678",
// 	}

// 	smsClnt, err := GetSmsClient()
// 	if err != nil {
// 		t.Errorf("unexpected error, %v", err)
// 	}

// 	if err := mem.sendMessage(smsClnt, "test text message"); err != nil {
// 		t.Errorf("unexpected error, %v", err)
// 	}
// }
