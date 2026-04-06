package prayertexter

import (
	"context"
	"log/slog"
	"strings"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
)

func completePrayer(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member) error {
	pryr := object.Prayer{IntercessorPhone: mem.Phone}
	if err := pryr.Get(ctx, ddbClnt, false); err != nil {
		return err
	}

	if pryr.Request == "" {
		// Get Dynamodb calls return empty data if the key does not exist in the table. Therefor if prayer request is
		// empty here, it means that it did not exist in the database.
		if err := mem.SendMessage(ctx, smsClnt, messaging.MsgNoActivePrayer); err != nil {
			return err
		}
		return nil
	}

	if err := mem.SendMessage(ctx, smsClnt, messaging.MsgPrayerThankYou); err != nil {
		return err
	}

	msg := strings.Replace(messaging.MsgPrayerConfirmation, "PLACEHOLDER", mem.Name, 1)

	isActive, err := object.IsMemberActive(ctx, ddbClnt, pryr.Requestor.Phone)
	if err != nil {
		return err
	}

	if isActive {
		if err = pryr.Requestor.SendMessage(ctx, smsClnt, msg); err != nil {
			return err
		}
	} else {
		slog.WarnContext(ctx, "Skip sending message, member is not active", "recipient", pryr.Requestor.Phone,
			"body", msg)
	}

	return pryr.Delete(ctx, ddbClnt, false)
}
