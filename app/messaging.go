package prayertexter

import (
	"log/slog"

	goaway "github.com/TwiN/go-away"
)

const (
	// sign up messages
	msgNameRequest             = "Text your name, or text 2 to stay anonymous"
	msgMemberTypeRequest       = "Text 1 for prayer request, or text 2 to be added to the intercessors list (to pray for others)"
	msgPrayerInstructions      = "You are now signed up to send prayer requests! Please send them directly to this number."
	msgPrayerNumRequest        = "Send the max number of prayer texts you are willing to receive and pray for each week."
	msgIntercessorInstructions = "You are now signed up to receive prayer requests. Please try to pray for the requests ASAP. Once you are done praying, send 'prayed' back to this number for confirmation."
	msgWrongInput              = "Wrong input received during sign up process. Please try again."

	// remove user messages
	msgRemoveUser = "You have been removed from prayer texter. If you ever want to sign back up, text the word pray to this number."

	// prayer request messages
	msgProfanityFound = "There was profanity found in your prayer request:\n\nPLACEHOLDER\n\nPlease try the request again without this word or words."
	msgPrayerIntro    = "Hello! Please pray for PLACEHOLDER:\n"
	msgPrayerSentOut  = "Your prayer request has been sent out!"

	// prayer completion messages
	msgNoActivePrayer     = "You have no more active prayers to mark as prayed"
	msgPrayerThankYou     = "Thank you for praying!"
	msgPrayerConfirmation = "You're prayer request has been prayed for by PLACEHOLDER"
)

type TextMessage struct {
	Body  string `json:"body"`
	Phone string `json:"phone-number"`
}

type TextSender interface {
	sendText(msg TextMessage) error
}

type FakeTextService struct{}

func (s FakeTextService) sendText(msg TextMessage) error {
	slog.Info("Sent text message", "recipient", msg.Phone, "body", msg.Body)
	return nil
}

func (t TextMessage) checkProfanity() string {
	// We need to remove some words from the profanity filter because it is too sensitive
	removedWords := []string{"jerk"}
	profanities := goaway.DefaultProfanities

	for _, word := range removedWords {
		remove(profanities, word)
	}

	return goaway.ExtractProfanity(t.Body)
}

func remove(words []string, word string) []string {
	for i, v := range words {
		if v == word {
			return append(words[:i], words[i+1:]...)
		}
	}
	return words
}
