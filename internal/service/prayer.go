package service

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/4JesusApps/prayertexter/internal/apperr"
	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
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
	introMsg, err := messaging.Render(messaging.PrayerIntroTmpl, struct{ Name string }{pryr.Requestor.Name})
	if err != nil {
		return err
	}

	reservedIntercessor, rollbackReservation, err := s.reserveIntercessor(ctx, intr.Phone)
	if err != nil {
		return err
	}

	pryr.Intercessor = *reservedIntercessor
	pryr.IntercessorPhone = reservedIntercessor.Phone
	pryr.ReminderCount = 0
	pryr.ReminderDate = time.Now().Format(time.RFC3339)
	if err = s.prayers.Save(ctx, &pryr, false); err != nil {
		if rollbackErr := rollbackReservation(ctx); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}

	msg := introMsg + pryr.Request + "\n\n" + messaging.MsgPrayed
	if err = s.sender.SendMessage(ctx, pryr.Intercessor.Phone, msg); err != nil {
		rollbackErr := s.rollbackAssignedPrayer(ctx, pryr.IntercessorPhone, rollbackReservation)
		if rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}

	slog.InfoContext(ctx, "assigned prayer successfully", "intercessor", pryr.Intercessor.Phone)
	return nil
}

func (s *PrayerService) FindIntercessors(ctx context.Context, skipPhones ...string) ([]domain.Member, error) {
	return s.findIntercessors(ctx, s.cfg.IntercessorsPerPrayer, skipPhones...)
}

func (s *PrayerService) findIntercessors(ctx context.Context, desired int, skipPhones ...string) ([]domain.Member, error) {
	allPhones, err := s.intercessors.Get(ctx)
	if err != nil {
		return nil, err
	}

	for _, skipPhone := range skipPhones {
		allPhones.RemovePhone(skipPhone)
	}

	var intercessors []domain.Member

	for len(intercessors) < desired {
		randPhones := allPhones.GenRandPhones(desired - len(intercessors))
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
			if len(intercessors) >= desired {
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
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrIntercessorUnavailable
	}
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

	if intr.WeeklyPrayerLimit <= 0 {
		return nil, ErrIntercessorUnavailable
	}

	if intr.PrayerCount < intr.WeeklyPrayerLimit {
		return intr, nil
	}

	canReset, err := canResetPrayerCount(*intr)
	if err != nil {
		return nil, err
	}
	if !canReset {
		return nil, ErrIntercessorUnavailable
	}

	return intr, nil
}

func (s *PrayerService) reserveIntercessor(
	ctx context.Context, phone string,
) (*domain.Member, func(context.Context) error, error) {
	intr, err := s.members.Get(ctx, phone)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil, ErrIntercessorUnavailable
	}
	if err != nil {
		return nil, nil, err
	}

	isActive, err := s.prayers.Exists(ctx, intr.Phone)
	if err != nil {
		return nil, nil, err
	}
	if isActive {
		return nil, nil, ErrIntercessorUnavailable
	}
	if intr.WeeklyPrayerLimit <= 0 {
		return nil, nil, ErrIntercessorUnavailable
	}

	original := *intr
	if intr.PrayerCount < intr.WeeklyPrayerLimit {
		intr.PrayerCount++
	} else {
		var canReset bool
		canReset, err = canResetPrayerCount(*intr)
		if err != nil {
			return nil, nil, err
		}
		if !canReset {
			return nil, nil, ErrIntercessorUnavailable
		}
		intr.PrayerCount = 1
		intr.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
	}

	if err = s.members.Save(ctx, intr); err != nil {
		return nil, nil, err
	}

	rollback := func(ctx context.Context) error {
		return s.members.Save(ctx, &original)
	}

	return intr, rollback, nil
}

func (s *PrayerService) rollbackAssignedPrayer(
	ctx context.Context, intercessorPhone string, rollbackReservation func(context.Context) error,
) error {
	var rollbackErrs []error
	if err := s.prayers.Delete(ctx, intercessorPhone, false); err != nil {
		rollbackErrs = append(rollbackErrs, err)
	}
	if err := rollbackReservation(ctx); err != nil {
		rollbackErrs = append(rollbackErrs, err)
	}
	return errors.Join(rollbackErrs...)
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
		QueueID:          id,
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
	if errors.Is(err, repository.ErrNotFound) {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgNoActivePrayer)
	}
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

func (s *PrayerService) RunScheduledJobs(ctx context.Context) error {
	var jobErrs []error

	if err := s.AssignQueuedPrayers(ctx); err != nil {
		apperr.LogError(ctx, err, "failed job", "job", "Assign Queued Prayers")
		jobErrs = append(jobErrs, err)
	} else {
		slog.InfoContext(ctx, "finished job", "job", "Assign Queued Prayers")
	}

	if err := s.RemindActiveIntercessors(ctx); err != nil {
		apperr.LogError(ctx, err, "failed job", "job", "Remind Intercessors with Active Prayers")
		jobErrs = append(jobErrs, err)
	} else {
		slog.InfoContext(ctx, "finished job", "job", "Remind Intercessors with Active Prayers")
	}

	return errors.Join(jobErrs...)
}

func (s *PrayerService) AssignQueuedPrayers(ctx context.Context) error {
	prayers, err := s.prayers.GetAll(ctx, true)
	if err != nil {
		return apperr.WrapError(err, "failed to get queued prayers")
	}

	for _, pryr := range prayers {
		if err = s.processQueuedPrayer(ctx, pryr); err != nil {
			return err
		}
	}

	return nil
}

func (s *PrayerService) processQueuedPrayer(ctx context.Context, pryr domain.Prayer) error {
	queueID := pryr.QueueID
	if queueID == "" {
		queueID = pryr.IntercessorPhone
		pryr.QueueID = queueID
	}

	activeAssignments, err := s.getActiveAssignmentsByQueueID(ctx, queueID)
	if err != nil {
		return apperr.WrapError(err, "failed to look up active queued assignments")
	}

	remaining := s.cfg.IntercessorsPerPrayer - len(activeAssignments)
	if remaining > 0 {
		var added []domain.Prayer
		added, err = s.assignAdditionalIntercessors(ctx, pryr, queueID, activeAssignments, remaining)
		if err != nil {
			return err
		}
		activeAssignments = append(activeAssignments, added...)
	}

	if len(activeAssignments) == 0 {
		return nil
	}

	return s.finalizeQueuedPrayer(ctx, pryr)
}

func (s *PrayerService) assignAdditionalIntercessors(
	ctx context.Context,
	pryr domain.Prayer,
	queueID string,
	activeAssignments []domain.Prayer,
	remaining int,
) ([]domain.Prayer, error) {
	skipPhones := []string{pryr.Requestor.Phone}
	for _, activePrayer := range activeAssignments {
		skipPhones = append(skipPhones, activePrayer.IntercessorPhone)
	}

	intercessors, err := s.findIntercessors(ctx, remaining, skipPhones...)
	if errors.Is(err, ErrNoAvailableIntercessors) {
		slog.WarnContext(ctx, "no intercessors available, leaving prayer queued", "queueID", queueID)
		return nil, nil
	}
	if err != nil {
		return nil, apperr.WrapError(err, "failed to find intercessors")
	}

	added := make([]domain.Prayer, 0, len(intercessors))
	for _, intr := range intercessors {
		pryr.QueueID = queueID
		if err = s.AssignPrayer(ctx, pryr, intr); err != nil {
			return nil, apperr.WrapError(err, "failed to assign prayer")
		}
		added = append(added, domain.Prayer{
			IntercessorPhone: intr.Phone,
			QueueID:          queueID,
		})
	}

	return added, nil
}

func (s *PrayerService) finalizeQueuedPrayer(ctx context.Context, pryr domain.Prayer) error {
	if !pryr.RequestorNotified {
		if err := s.sender.SendMessage(ctx, pryr.Requestor.Phone, messaging.MsgPrayerAssigned); err != nil {
			return err
		}
		pryr.RequestorNotified = true
		if err := s.prayers.Save(ctx, &pryr, true); err != nil {
			return err
		}
	}

	return s.prayers.Delete(ctx, pryr.IntercessorPhone, true)
}

func (s *PrayerService) getActiveAssignmentsByQueueID(ctx context.Context, queueID string) ([]domain.Prayer, error) {
	activePrayers, err := s.prayers.GetAll(ctx, false)
	if err != nil {
		return nil, err
	}

	assignments := make([]domain.Prayer, 0)
	for _, pryr := range activePrayers {
		if pryr.QueueID == queueID {
			assignments = append(assignments, pryr)
		}
	}

	return assignments, nil
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
