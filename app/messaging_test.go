package prayertexter

import (
	"log/slog"
	"testing"
)

type MockTextService struct {
	SendTextCalls   int
	SendTextInputs  []TextMessage
	SendTextResults []struct {
		Error error
	}
}

func (mock *MockTextService) sendText(clnt DDBConnecter, msg TextMessage) error {
	slog.Info("MOCK SMS:", "recipient", msg.Phone, "body", msg.Body)

	mock.SendTextCalls++
	mock.SendTextInputs = append(mock.SendTextInputs, msg)

	// Default result if no results are configured to avoid index out of bounds
	if len(mock.SendTextResults) <= mock.SendTextCalls-1 {
		return nil
	}

	result := mock.SendTextResults[mock.SendTextCalls-1]

	return result.Error
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
