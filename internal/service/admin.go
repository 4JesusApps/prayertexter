package service

import (
	"context"
	"errors"
	"regexp"
	"slices"

	"github.com/4JesusApps/prayertexter/internal/apperrors"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/model"
)

func (s *Service) blockUser(ctx context.Context, msg messaging.TextMessage, mem *model.Member, blockedPhones *model.BlockedPhones) error {
	if !mem.Administrator {
		if err := s.sendMessage(ctx, mem.Phone, messaging.MsgUnauthorized); err != nil {
			return err
		}
		return nil
	}

	phone, err := extractPhone(msg.Body)
	if errors.Is(err, apperrors.ErrInvalidPhone) {
		if err = s.sendMessage(ctx, mem.Phone, messaging.MsgInvalidPhone); err != nil {
			return err
		}
		return nil
	}

	phone = "+1" + phone
	if slices.Contains(blockedPhones.Phones, phone) {
		if err = s.sendMessage(ctx, mem.Phone, messaging.MsgUserAlreadyBlocked); err != nil {
			return err
		}
		return nil
	}

	blockedPhones.AddPhone(phone)
	if err = s.putBlockedPhones(ctx, blockedPhones); err != nil {
		return err
	}

	blockedUser, err := s.getMember(ctx, phone)
	if err != nil {
		return err
	}

	if err = s.memberDelete(ctx, blockedUser); err != nil {
		return err
	}

	if err = s.sendMessage(ctx, blockedUser.Phone, messaging.MsgBlockedNotification+messaging.MsgHelp); err != nil {
		return err
	}

	return s.sendMessage(ctx, mem.Phone, messaging.MsgSuccessfullyBlocked)
}

func extractPhone(msg string) (string, error) {
	var phoneRE = regexp.MustCompile(`\(?\b(\d{3})\)?[\s\-]?(\d{3})[\s\-]?(\d{4})\b`)

	const phoneRegexMatchCount = 4
	matches := phoneRE.FindStringSubmatch(msg)
	if len(matches) != phoneRegexMatchCount {
		return "", apperrors.ErrInvalidPhone
	}

	return matches[1] + matches[2] + matches[3], nil
}
