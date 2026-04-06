package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/4JesusApps/prayertexter/internal/apperrors"
	"github.com/4JesusApps/prayertexter/internal/messaging"
)

// AssignQueuedPrayers gets all prayers in the queued prayers table if any. It will then attempt to assign each prayer
// to intercessors if there are any available. If a prayer is assigned successfully, it sends the prayer request to the
// intercessors as well as sending a confirmation message to the prayer requestor.
func (s *Service) AssignQueuedPrayers(ctx context.Context) error {
	prayers, err := s.scanQueuedPrayers(ctx)
	if err != nil {
		return apperrors.WrapError(err, "failed to get queued prayers")
	}

	for _, pryr := range prayers {
		queuedPrayerID := pryr.IntercessorPhone

		intercessors, err := s.FindIntercessors(ctx, pryr.Requestor.Phone)
		if err != nil && errors.Is(err, apperrors.ErrNoAvailableIntercessors) {
			slog.WarnContext(ctx, "no intercessors available, exiting job")
			break
		} else if err != nil {
			return apperrors.WrapError(err, "failed to find intercessors")
		}

		for i := range intercessors {
			if err = s.AssignPrayer(ctx, &pryr, &intercessors[i]); err != nil {
				return apperrors.WrapError(err, "failed to assign prayer")
			}
		}

		if err = s.deleteQueuedPrayer(ctx, queuedPrayerID); err != nil {
			return err
		}

		if err = s.sendMessage(ctx, pryr.Requestor.Phone, messaging.MsgPrayerAssigned); err != nil {
			return err
		}
	}

	return nil
}

// RemindActiveIntercessors reminds intercessors who have active prayers that need attention.
func (s *Service) RemindActiveIntercessors(ctx context.Context) error {
	prayerReminderHours := s.cfg.Prayer.ReminderHours

	prayers, err := s.scanActivePrayers(ctx)
	if err != nil {
		return apperrors.WrapError(err, "failed to get active prayers")
	}

	currentTime := time.Now()
	for _, pryr := range prayers {
		if pryr.ReminderDate == "" {
			pryr.ReminderDate = currentTime.Format(time.RFC3339)
			if err = s.putActivePrayer(ctx, &pryr); err != nil {
				return err
			}
			continue
		}

		var previousTime time.Time
		previousTime, err = time.Parse(time.RFC3339, pryr.ReminderDate)
		if err != nil {
			return apperrors.WrapError(err, "failed to parse time")
		}
		diffTime := currentTime.Sub(previousTime).Hours()
		if diffTime > float64(prayerReminderHours) {
			pryr.ReminderCount++
			pryr.ReminderDate = currentTime.Format(time.RFC3339)
			if err = s.putActivePrayer(ctx, &pryr); err != nil {
				return err
			}

			msg := strings.Replace(messaging.MsgPrayerReminder, "PLACEHOLDER", pryr.Requestor.Name, 1)
			msg = msg + pryr.Request + "\n\n" + messaging.MsgPrayed
			if err = s.sendMessage(ctx, pryr.Intercessor.Phone, msg); err != nil {
				return err
			}
		}
	}

	return nil
}
