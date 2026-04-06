package service

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/model"
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

func (s *Service) checkIfProfanity(ctx context.Context, mem *model.Member, msg messaging.TextMessage) (bool, error) {
	profanity := s.profanity.Check(msg.Body)
	if profanity != "" {
		notif := strings.Replace(messaging.MsgProfanityDetected, "PLACEHOLDER", profanity, 1)
		if err := s.sendMessage(ctx, mem.Phone, notif); err != nil {
			return true, err
		}

		return true, nil
	}

	return false, nil
}

func (s *Service) checkIfNameValid(ctx context.Context, mem *model.Member) (bool, error) {
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
		if err := s.sendMessage(ctx, mem.Phone, messaging.MsgInvalidName); err != nil {
			return isValid, err
		}

		return isValid, nil
	}

	return isValid, nil
}

func (s *Service) checkIfRequestValid(ctx context.Context, msg messaging.TextMessage, mem *model.Member) (bool, error) {
	if len(strings.Fields(msg.Body)) < MinRequestWords {
		if err := s.sendMessage(ctx, mem.Phone, messaging.MsgInvalidRequest); err != nil {
			return false, err
		}

		return false, nil
	}

	return true, nil
}

func (s *Service) handleTriggerWords(msg *messaging.TextMessage, mem *model.Member) {
	//nolint:gocritic // ignoring switch statement warning because this will be expanded in the future
	switch {
	case strings.Contains(strings.ToLower(msg.Body), "#anon"):
		mem.Name = "Anonymous"
		re := regexp.MustCompile(`(?i)#anon`)
		msg.Body = strings.TrimSpace(re.ReplaceAllString(msg.Body, ""))
	}
}

func canResetPrayerCount(intr *model.Member) (bool, error) {
	currentTime := time.Now()
	previousTime, err := time.Parse(time.RFC3339, intr.WeeklyPrayerDate)
	if err != nil {
		return false, err
	}
	diffDays := currentTime.Sub(previousTime).Hours() / float64(HoursPerDay)
	return diffDays > float64(WeekDays), nil
}
