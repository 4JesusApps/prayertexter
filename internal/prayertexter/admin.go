package prayertexter

import (
	"context"
	"errors"
	"regexp"
	"slices"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/utility"
)

func blockUser(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, msg messaging.TextMessage, mem object.Member, blockedPhones object.BlockedPhones) error {
	if !mem.Administrator {
		if err := mem.SendMessage(ctx, smsClnt, messaging.MsgUnauthorized); err != nil {
			return err
		}
		return nil
	}

	phone, err := extractPhone(msg.Body)
	if errors.Is(err, utility.ErrInvalidPhone) {
		if err = mem.SendMessage(ctx, smsClnt, messaging.MsgInvalidPhone); err != nil {
			return err
		}
		return nil
	}

	phone = "+1" + phone
	if slices.Contains(blockedPhones.Phones, phone) {
		if err = mem.SendMessage(ctx, smsClnt, messaging.MsgUserAlreadyBlocked); err != nil {
			return err
		}
		return nil
	}

	blockedPhones.AddPhone(phone)
	if err = blockedPhones.Put(ctx, ddbClnt); err != nil {
		return err
	}

	blockedUser := object.Member{Phone: phone}
	if err = blockedUser.Get(ctx, ddbClnt); err != nil {
		return err
	}

	if err = memberDelete(ctx, ddbClnt, smsClnt, blockedUser); err != nil {
		return err
	}

	if err = blockedUser.SendMessage(ctx, smsClnt, messaging.MsgBlockedNotification+messaging.MsgHelp); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgSuccessfullyBlocked)
}

func extractPhone(msg string) (string, error) {
	// regex matches:
	// (123) 456-7890
	// 123-456-7890
	// 1234567890
	var phoneRE = regexp.MustCompile(`\(?\b(\d{3})\)?[\s\-]?(\d{3})[\s\-]?(\d{4})\b`)

	// Regex match is 1 + each compile group, which is 3.
	const phoneRegexMatchCount = 4
	matches := phoneRE.FindStringSubmatch(msg)
	if len(matches) != phoneRegexMatchCount {
		return "", utility.ErrInvalidPhone
	}

	return matches[1] + matches[2] + matches[3], nil
}
