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
