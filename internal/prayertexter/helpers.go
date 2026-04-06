package prayertexter

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
)

const (
	// MinNameLetters is the minimum number of letters required in a valid name.
	MinNameLetters = 2
	// MinRequestWords is the minimum number of words required in a prayer request.
	MinRequestWords = 5
	// WeekDays is the number of days in a week, used for prayer count reset calculation.
	WeekDays = 7
	// HoursPerDay is used for converting time differences to days.
	HoursPerDay = 24
)

// checkIfProfanity reports whether there is profanity in a text message. If there is, it will inform the sender.
func checkIfProfanity(ctx context.Context, smsClnt messaging.TextSender, mem object.Member, msg messaging.TextMessage, profanityChecker *messaging.ProfanityChecker) (bool, error) {
	profanity := profanityChecker.Check(msg.Body)
	if profanity != "" {
		msg := strings.Replace(messaging.MsgProfanityDetected, "PLACEHOLDER", profanity, 1)
		if err := mem.SendMessage(ctx, smsClnt, msg); err != nil {
			return true, err
		}

		return true, nil
	}

	return false, nil
}

// checkIfNameValid reports whether a name is valid. A valid name is at least 2 characters long and does not contain any
// numbers. If name is invalid, it will inform the sender.
func checkIfNameValid(ctx context.Context, smsClnt messaging.TextSender, mem object.Member) (bool, error) {
	letterCount := 0
	isValid := true

	for _, ch := range mem.Name {
		switch {
		case unicode.IsLetter(ch):
			letterCount++
		case ch == ' ':
			// Do nothing; spaces are fine but don't count toward letters.
		default:
			isValid = false
		}
	}

	if letterCount < MinNameLetters {
		isValid = false
	}

	if !isValid {
		if err := mem.SendMessage(ctx, smsClnt, messaging.MsgInvalidName); err != nil {
			return isValid, err
		}

		return isValid, nil
	}

	return isValid, nil
}

func checkIfRequestValid(ctx context.Context, smsClnt messaging.TextSender, msg messaging.TextMessage, mem object.Member) (bool, error) {
	if len(strings.Fields(msg.Body)) < MinRequestWords {
		if err := mem.SendMessage(ctx, smsClnt, messaging.MsgInvalidRequest); err != nil {
			return false, err
		}

		return false, nil
	}

	return true, nil
}

// handleTriggerWords performs the necessary actions for any trigger words in the message and then removes the trigger
// words from the message body. Trigger words start with a #.
func handleTriggerWords(msg *messaging.TextMessage, mem *object.Member) {
	//nolint:gocritic // ignoring switch statement warning because this will be expanded in the future
	switch {
	case strings.Contains(strings.ToLower(msg.Body), "#anon"):
		mem.Name = "Anonymous"
		re := regexp.MustCompile(`(?i)#anon`)
		msg.Body = strings.TrimSpace(re.ReplaceAllString(msg.Body, ""))
		// Add future trigger words as new cases
	}
}

func canResetPrayerCount(intr object.Member) (bool, error) {

	currentTime := time.Now()
	previousTime, err := time.Parse(time.RFC3339, intr.WeeklyPrayerDate)
	if err != nil {
		return false, err
	}
	diffDays := currentTime.Sub(previousTime).Hours() / float64(HoursPerDay)
	return diffDays > float64(WeekDays), nil
}
