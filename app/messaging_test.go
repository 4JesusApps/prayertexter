package prayertexter

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
)

type MockTextSender struct {
	SendTextCalls   int
	SendTextInputs  []*pinpointsmsvoicev2.SendTextMessageInput
	SendTextResults []struct {
		Error error
	}
}

func (m *MockTextSender) SendTextMessage(ctx context.Context,
	params *pinpointsmsvoicev2.SendTextMessageInput,
	optFns ...func(*pinpointsmsvoicev2.Options)) (*pinpointsmsvoicev2.SendTextMessageOutput, error) {

	m.SendTextCalls++
	m.SendTextInputs = append(m.SendTextInputs, params)

	// Default result if no results are configured to avoid index out of bounds
	if len(m.SendTextResults) <= m.SendTextCalls-1 {
		return &pinpointsmsvoicev2.SendTextMessageOutput{}, nil
	}

	result := m.SendTextResults[m.SendTextCalls-1]

	return &pinpointsmsvoicev2.SendTextMessageOutput{}, result.Error
}

func TestSendText(t *testing.T) {
	msg := TextMessage{
		Body:  "test text message",
		Phone: "+11234567890",
	}

	txtMock := &MockTextSender{}

	if err := sendText(txtMock, msg); err != nil {
		t.Errorf("unexpected error, %v", err)
	}

	receivedText := TextMessage{
		Body:  *txtMock.SendTextInputs[0].MessageBody,
		Phone: *txtMock.SendTextInputs[0].DestinationPhoneNumber,
	}

	msg.Body = msgPre + msg.Body + "\n\n" + msgPost

	if receivedText != msg {
		t.Errorf("expected txt %v, got %v", msg, receivedText)
	}

	if *txtMock.SendTextInputs[0].OriginationIdentity != prayerTexterPhone {
		t.Errorf("expected phone number %v, got %v", prayerTexterPhone, *txtMock.SendTextInputs[0].OriginationIdentity)
	}
}

func TestCheckProfanity(t *testing.T) {
	msg := TextMessage{Body: "test text message, no profanity"}
	profanity := msg.checkProfanity()
	if profanity != "" {
		t.Errorf("expected no profanity, got %v", profanity)
	}

	msg.Body = "this message contains profanity, sh!t!"
	profanity = msg.checkProfanity()
	if profanity == "" {
		t.Errorf("expected profanity, got none (empty string): %v", profanity)
	}
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
