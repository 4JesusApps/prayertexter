package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/4JesusApps/prayertexter/internal/apperrors"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/model"
)

func (s *Service) prayerRequest(ctx context.Context, msg messaging.TextMessage, mem *model.Member) error {
	hasProfanity, err := s.checkIfProfanity(ctx, mem, msg)
	if err != nil {
		return err
	} else if hasProfanity {
		return nil
	}

	isValid, err := s.checkIfRequestValid(ctx, msg, mem)
	if err != nil {
		return err
	} else if !isValid {
		return nil
	}

	s.handleTriggerWords(&msg, mem)

	intercessors, err := s.FindIntercessors(ctx, mem.Phone)
	if err != nil && errors.Is(err, apperrors.ErrNoAvailableIntercessors) {
		slog.WarnContext(ctx, "no intercessors available", "request", msg.Body, "requestor", msg.Phone)
		if err = s.queuePrayer(ctx, msg, mem); err != nil {
			return apperrors.WrapError(err, "failed to queue prayer")
		}
		return nil
	} else if err != nil {
		return apperrors.WrapError(err, "failed to find intercessors")
	}

	for _, intr := range intercessors {
		pryr := model.Prayer{
			Request:   msg.Body,
			Requestor: *mem,
		}

		if err = s.AssignPrayer(ctx, &pryr, &intr); err != nil {
			return err
		}
	}

	return s.sendMessage(ctx, mem.Phone, messaging.MsgPrayerAssigned)
}

// AssignPrayer saves a prayer to the active prayers table and sends the intercessor a text message.
func (s *Service) AssignPrayer(ctx context.Context, pryr *model.Prayer, intr *model.Member) error {
	pryr.Intercessor = *intr
	pryr.IntercessorPhone = intr.Phone
	if err := s.putActivePrayer(ctx, pryr); err != nil {
		return err
	}

	msg := strings.Replace(messaging.MsgPrayerIntro, "PLACEHOLDER", pryr.Requestor.Name, 1)
	msg = msg + pryr.Request + "\n\n" + messaging.MsgPrayed
	err := s.sendMessage(ctx, pryr.Intercessor.Phone, msg)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "assigned prayer successfully")
	return nil
}

// FindIntercessors returns a slice of available intercessors for a prayer request.
func (s *Service) FindIntercessors(ctx context.Context, skipPhone string) ([]model.Member, error) {
	allPhones, err := s.getAndPreparePhones(ctx, skipPhone)
	if err != nil {
		return nil, err
	}

	var intercessors []model.Member
	intercessorsPerPrayer := s.cfg.Prayer.IntercessorsPerPrayer

	for len(intercessors) < intercessorsPerPrayer {
		randPhones := allPhones.GenRandPhones(intercessorsPerPrayer)
		if randPhones == nil {
			slog.InfoContext(ctx, "there are no more intercessors left to check")
			if len(intercessors) > 0 {
				slog.InfoContext(ctx, "there is at least one intercessor found, returning this even though it is less "+
					"than the desired number of intercessors per prayer")
				return intercessors, nil
			}

			return nil, apperrors.ErrNoAvailableIntercessors
		}

		for _, phn := range randPhones {
			if len(intercessors) >= intercessorsPerPrayer {
				return intercessors, nil
			}

			var intr *model.Member
			intr, err = s.processIntercessor(ctx, phn)
			if err != nil && errors.Is(err, apperrors.ErrIntercessorUnavailable) {
				allPhones.RemovePhone(phn)
				continue
			} else if err != nil {
				return nil, err
			}

			intercessors = append(intercessors, *intr)
			allPhones.RemovePhone(phn)
			slog.InfoContext(ctx, "found one available intercessor")
		}
	}

	return intercessors, nil
}

func (s *Service) getAndPreparePhones(ctx context.Context, skipPhone string) (*model.IntercessorPhones, error) {
	phones, err := s.getIntercessorPhones(ctx)
	if err != nil {
		return nil, err
	}

	model.RemoveItem(&phones.Phones, skipPhone)

	return phones, nil
}

func (s *Service) processIntercessor(ctx context.Context, phone string) (*model.Member, error) {
	intr, err := s.getMember(ctx, phone)
	if err != nil {
		return nil, err
	}

	isActive, err := s.isPrayerActive(ctx, intr.Phone)
	if err != nil {
		return nil, err
	}
	if isActive {
		return nil, apperrors.ErrIntercessorUnavailable
	}

	if intr.PrayerCount < intr.WeeklyPrayerLimit {
		intr.PrayerCount++
	} else {
		var canReset bool
		canReset, err = canResetPrayerCount(intr)
		if err != nil {
			return nil, err
		}
		if canReset {
			intr.PrayerCount = 1
			intr.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
		} else {
			return nil, apperrors.ErrIntercessorUnavailable
		}
	}

	if err = s.putMember(ctx, intr); err != nil {
		return nil, err
	}
	return intr, nil
}

func (s *Service) queuePrayer(ctx context.Context, msg messaging.TextMessage, mem *model.Member) error {
	id, err := model.GenerateID()
	if err != nil {
		return err
	}

	pryr := model.Prayer{
		IntercessorPhone: id,
		Request:          msg.Body,
		Requestor:        *mem,
	}

	if err = s.putQueuedPrayer(ctx, &pryr); err != nil {
		return err
	}

	return s.sendMessage(ctx, mem.Phone, messaging.MsgPrayerQueued)
}
