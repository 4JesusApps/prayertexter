package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/model"
)

func (s *Service) completePrayer(ctx context.Context, mem *model.Member) error {
	pryr, err := s.getActivePrayer(ctx, mem.Phone)
	if err != nil {
		return err
	}

	if !pryr.IsActive() {
		if sendErr := s.sendMessage(ctx, mem.Phone, messaging.MsgNoActivePrayer); sendErr != nil {
			return sendErr
		}
		return nil
	}

	if err = s.sendMessage(ctx, mem.Phone, messaging.MsgPrayerThankYou); err != nil {
		return err
	}

	msg := strings.Replace(messaging.MsgPrayerConfirmation, "PLACEHOLDER", mem.Name, 1)

	isActive, err := s.isMemberActive(ctx, pryr.Requestor.Phone)
	if err != nil {
		return err
	}

	if isActive {
		if err = s.sendMessage(ctx, pryr.Requestor.Phone, msg); err != nil {
			return err
		}
	} else {
		slog.WarnContext(ctx, "Skip sending message, member is not active", "recipient", pryr.Requestor.Phone,
			"body", msg)
	}

	return s.deleteActivePrayer(ctx, mem.Phone)
}
