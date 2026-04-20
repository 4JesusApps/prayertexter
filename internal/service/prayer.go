package service

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/apperr"
)

type PrayerService struct {
	members      repository.MemberRepository
	intercessors repository.IntercessorPhonesRepository
	prayers      repository.PrayerRepository
	sender       messaging.MessageSender
	cfg          config.Config
}

func NewPrayerService(
	members repository.MemberRepository,
	intercessors repository.IntercessorPhonesRepository,
	prayers repository.PrayerRepository,
	sender messaging.MessageSender,
	cfg config.Config,
) *PrayerService {
	return &PrayerService{
		members:      members,
		intercessors: intercessors,
		prayers:      prayers,
		sender:       sender,
		cfg:          cfg,
	}
}

func (s *PrayerService) Request(ctx context.Context, msg domain.TextMessage, mem domain.Member) error {
	profanity := messaging.CheckProfanity(msg.Body)
	if profanity != "" {
		rendered, err := messaging.Render(messaging.ProfanityDetectedTmpl, struct{ Word string }{profanity})
		if err != nil {
			return err
		}
		return s.sender.SendMessage(ctx, mem.Phone, rendered)
	}

	if !isRequestValid(msg) {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgInvalidRequest)
	}

	handleTriggerWords(&msg, &mem)

	intercessors, err := s.FindIntercessors(ctx, mem.Phone)
	if err != nil && errors.Is(err, ErrNoAvailableIntercessors) {
		slog.WarnContext(ctx, "no intercessors available", "request", msg.Body, "requestor", msg.Phone)
		return s.queuePrayer(ctx, msg, mem)
	} else if err != nil {
		return apperr.WrapError(err, "failed to find intercessors")
	}

	for _, intr := range intercessors {
		pryr := domain.Prayer{
			Request:   msg.Body,
			Requestor: mem,
		}
		if err = s.AssignPrayer(ctx, pryr, intr); err != nil {
			return err
		}
	}

	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgPrayerAssigned)
}

func isRequestValid(msg domain.TextMessage) bool {
	minWords := 5
	return len(strings.Fields(msg.Body)) >= minWords
}

func handleTriggerWords(msg *domain.TextMessage, mem *domain.Member) {
	if strings.Contains(strings.ToLower(msg.Body), "#anon") {
		mem.Name = "Anonymous"
		re := regexp.MustCompile(`(?i)#anon`)
		msg.Body = strings.TrimSpace(re.ReplaceAllString(msg.Body, ""))
	}
}

func (s *PrayerService) AssignPrayer(ctx context.Context, pryr domain.Prayer, intr domain.Member) error {
	pryr.Intercessor = intr
	pryr.IntercessorPhone = intr.Phone
	if err := s.prayers.Save(ctx, &pryr, false); err != nil {
		return err
	}

	introMsg, err := messaging.Render(messaging.PrayerIntroTmpl, struct{ Name string }{pryr.Requestor.Name})
	if err != nil {
		return err
	}
	msg := introMsg + pryr.Request + "\n\n" + messaging.MsgPrayed
	if err = s.sender.SendMessage(ctx, pryr.Intercessor.Phone, msg); err != nil {
		return err
	}

	slog.InfoContext(ctx, "assigned prayer successfully")
	return nil
}

func (s *PrayerService) FindIntercessors(ctx context.Context, skipPhone string) ([]domain.Member, error) {
	allPhones, err := s.intercessors.Get(ctx)
	if err != nil {
		return nil, err
	}

	allPhones.RemovePhone(skipPhone)

	var intercessors []domain.Member

	for len(intercessors) < s.cfg.IntercessorsPerPrayer {
		randPhones := allPhones.GenRandPhones(s.cfg.IntercessorsPerPrayer)
		if randPhones == nil {
			slog.InfoContext(ctx, "there are no more intercessors left to check")
			if len(intercessors) > 0 {
				slog.InfoContext(ctx, "there is at least one intercessor found, returning this even though it is less "+
					"than the desired number of intercessors per prayer")
				return intercessors, nil
			}
			return nil, ErrNoAvailableIntercessors
		}

		for _, phn := range randPhones {
			if len(intercessors) >= s.cfg.IntercessorsPerPrayer {
				return intercessors, nil
			}

			var intr *domain.Member
			intr, err = s.processIntercessor(ctx, phn)
			if err != nil && errors.Is(err, ErrIntercessorUnavailable) {
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

func (s *PrayerService) processIntercessor(ctx context.Context, phone string) (*domain.Member, error) {
	intr, err := s.members.Get(ctx, phone)
	if err != nil {
		return nil, err
	}

	isActive, err := s.prayers.Exists(ctx, intr.Phone)
	if err != nil {
		return nil, err
	}
	if isActive {
		return nil, ErrIntercessorUnavailable
	}

	if intr.PrayerCount < intr.WeeklyPrayerLimit {
		intr.PrayerCount++
	} else {
		var canReset bool
		canReset, err = canResetPrayerCount(*intr)
		if err != nil {
			return nil, err
		}
		if canReset {
			intr.PrayerCount = 1
			intr.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
		} else {
			return nil, ErrIntercessorUnavailable
		}
	}

	if err = s.members.Save(ctx, intr); err != nil {
		return nil, err
	}
	return intr, nil
}

func canResetPrayerCount(intr domain.Member) (bool, error) {
	previousTime, err := time.Parse(time.RFC3339, intr.WeeklyPrayerDate)
	if err != nil {
		return false, err
	}
	return time.Since(previousTime) > 7*24*time.Hour, nil
}

func (s *PrayerService) queuePrayer(ctx context.Context, msg domain.TextMessage, mem domain.Member) error {
	id, err := generateID()
	if err != nil {
		return err
	}

	pryr := domain.Prayer{
		IntercessorPhone: id,
		Request:          msg.Body,
		Requestor:        mem,
	}

	if err = s.prayers.Save(ctx, &pryr, true); err != nil {
		return err
	}

	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgPrayerQueued)
}

func (s *PrayerService) Complete(ctx context.Context, mem domain.Member) error {
	pryr, err := s.prayers.Get(ctx, mem.Phone, false)
	if err != nil {
		return err
	}

	if pryr.Request == "" {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgNoActivePrayer)
	}

	if err = s.sender.SendMessage(ctx, mem.Phone, messaging.MsgPrayerThankYou); err != nil {
		return err
	}

	confirmMsg, err := messaging.Render(messaging.PrayerConfirmationTmpl, struct{ Name string }{mem.Name})
	if err != nil {
		return err
	}

	isActive, err := s.members.Exists(ctx, pryr.Requestor.Phone)
	if err != nil {
		return err
	}

	if isActive {
		if err = s.sender.SendMessage(ctx, pryr.Requestor.Phone, confirmMsg); err != nil {
			return err
		}
	} else {
		slog.WarnContext(ctx, "Skip sending message, member is not active", "recipient", pryr.Requestor.Phone,
			"body", confirmMsg)
	}

	return s.prayers.Delete(ctx, mem.Phone, false)
}

func (s *PrayerService) RunScheduledJobs(ctx context.Context) {
	if err := s.AssignQueuedPrayers(ctx); err != nil {
		apperr.LogError(ctx, err, "failed job", "job", "Assign Queued Prayers")
	} else {
		slog.InfoContext(ctx, "finished job", "job", "Assign Queued Prayers")
	}

	if err := s.RemindActiveIntercessors(ctx); err != nil {
		apperr.LogError(ctx, err, "failed job", "job", "Remind Intercessors with Active Prayers")
	} else {
		slog.InfoContext(ctx, "finished job", "job", "Remind Intercessors with Active Prayers")
	}
}

func (s *PrayerService) AssignQueuedPrayers(ctx context.Context) error {
	prayers, err := s.prayers.GetAll(ctx, true)
	if err != nil {
		return apperr.WrapError(err, "failed to get queued prayers")
	}

	for _, pryr := range prayers {
		var intercessors []domain.Member
		intercessors, err = s.FindIntercessors(ctx, pryr.Requestor.Phone)
		if err != nil && errors.Is(err, ErrNoAvailableIntercessors) {
			slog.WarnContext(ctx, "no intercessors available, exiting job")
			break
		} else if err != nil {
			return apperr.WrapError(err, "failed to find intercessors")
		}

		for _, intr := range intercessors {
			if err = s.AssignPrayer(ctx, pryr, intr); err != nil {
				return apperr.WrapError(err, "failed to assign prayer")
			}
		}

		if err = s.prayers.Delete(ctx, pryr.IntercessorPhone, true); err != nil {
			return err
		}

		if err = s.sender.SendMessage(ctx, pryr.Requestor.Phone, messaging.MsgPrayerAssigned); err != nil {
			return err
		}
	}

	return nil
}

func (s *PrayerService) RemindActiveIntercessors(ctx context.Context) error {
	prayers, err := s.prayers.GetAll(ctx, false)
	if err != nil {
		return apperr.WrapError(err, "failed to get active prayers")
	}

	currentTime := time.Now()
	for _, pryr := range prayers {
		if pryr.ReminderDate == "" {
			pryr.ReminderDate = currentTime.Format(time.RFC3339)
			if err = s.prayers.Save(ctx, &pryr, false); err != nil {
				return err
			}
			continue
		}

		var previousTime time.Time
		previousTime, err = time.Parse(time.RFC3339, pryr.ReminderDate)
		if err != nil {
			return apperr.WrapError(err, "failed to parse time")
		}
		diffTime := currentTime.Sub(previousTime).Hours()
		if diffTime > float64(s.cfg.PrayerReminderHours) {
			pryr.ReminderCount++
			pryr.ReminderDate = currentTime.Format(time.RFC3339)
			if err = s.prayers.Save(ctx, &pryr, false); err != nil {
				return err
			}

			var reminderMsg string
			reminderMsg, err = messaging.Render(
				messaging.PrayerReminderTmpl,
				struct{ Name string }{pryr.Requestor.Name},
			)
			if err != nil {
				return err
			}
			msg := reminderMsg + pryr.Request + "\n\n" + messaging.MsgPrayed
			if err = s.sender.SendMessage(ctx, pryr.Intercessor.Phone, msg); err != nil {
				return err
			}
		}
	}

	return nil
}
