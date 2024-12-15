package prayertexter

import (
	"log/slog"
)

type TextSender interface {
	SendText(msg TextMessage) error
}

type TextMessage struct {
	Body  string `json:"body"`
	Phone string `json:"phone-number"`
}

type FakeTextService struct{}

func (s FakeTextService) SendText(msg TextMessage) error {
	slog.Info("Sent text message", "recipient", msg.Phone, "body", msg.Body)
	return nil
}
