package prayertexter

import (
	"log/slog"
)

type MockTextService struct {
	SendTextCalls   int
	SendTextInputs  []TextMessage
	SendTextResults []struct {
		Error error
	}
}

func (mock *MockTextService) sendText(msg TextMessage) error {
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
