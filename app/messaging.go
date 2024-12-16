package prayertexter

import (
	"log/slog"
)

const (
	msgNameRequest             = "Text your name, or 2 to stay anonymous"
	msgMemberTypeRequest       = "Text 1 for prayer request, or 2 to be added to the intercessors list (to pray for others)"
	msgPrayerInstructions      = "You are now signed up to send prayer requests! Please send them directly to this number."
	msgPrayerNumRequest        = "Send the max number of prayer texts you are willing to receive and pray for per week."
	msgIntercessorInstructions = "You are now signed up to receive prayer requests. Please try to pray for the requests ASAP. Once you are done praying, send 'prayed' back to this number for confirmation."
	msgWrongInput              = "Wrong input received during sign up process. Please try again."
	msgRemoveUser              = "You have been removed from prayer texter. If you ever want to sign back up, text the word pray to this number."
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
