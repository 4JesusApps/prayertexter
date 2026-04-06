// Package service contains all business logic for the prayertexter application.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"unicode"

	"github.com/4JesusApps/prayertexter/internal/apperrors"
	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/model"
)

const (
	preStage            = "PRE"
	blockedUserStage    = "BLOCKED USER"
	addBlockedUserStage = "ADD BLOCKED USER"
	helpStage           = "HELP"
	memberDeleteStage   = "MEMBER DELETE"
	signUpStage         = "SIGN UP"
	dropMessageStage    = "DROP MESSAGE"
	completePrayerStage = "COMPLETE PRAYER"
	prayerRequestStage  = "PRAYER REQUEST"
	stageErrPrefix      = "failure during stage "
)

// profanityExceptions are words removed from the default profanity filter because it is too sensitive.
var profanityExceptions = []string{"jerk", "ass", "butt"} //nolint:gochecknoglobals // constant-like configuration

// Service holds all dependencies for the prayertexter business logic.
type Service struct {
	cfg       *config.Config
	ddb       db.DDBConnecter
	sms       messaging.TextSender
	profanity *messaging.ProfanityChecker
}

// NewService creates a Service with all required dependencies.
func NewService(cfg *config.Config, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) *Service {
	return &Service{
		cfg:       cfg,
		ddb:       ddbClnt,
		sms:       smsClnt,
		profanity: messaging.NewProfanityChecker(profanityExceptions),
	}
}

// MainFlow processes an incoming text message and routes it to the appropriate handler.
func (s *Service) MainFlow(ctx context.Context, msg messaging.TextMessage) error {
	mem, err := s.getMember(ctx, msg.Phone)
	if err != nil {
		return apperrors.LogAndWrapError(ctx, err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	blockedPhones, err := s.getBlockedPhones(ctx)
	if err != nil {
		return apperrors.LogAndWrapError(ctx, err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}
	isBlocked := slices.Contains(blockedPhones.Phones, mem.Phone)

	var stageErr error
	cleanMsg := cleanStr(msg.Body)

	switch {
	case isBlocked:
		slog.WarnContext(ctx, "blocked user dropping message", "phone", mem.Phone, "msg", msg.Body)
		stageErr = s.executeStage(ctx, mem, msg, blockedUserStage, func() error {
			return nil
		})

	case strings.Contains(strings.ToLower(msg.Body), "#block"):
		stageErr = s.executeStage(ctx, mem, msg, addBlockedUserStage, func() error {
			return s.blockUser(ctx, msg, mem, blockedPhones)
		})

	case cleanMsg == "help":
		stageErr = s.executeStage(ctx, mem, msg, helpStage, func() error {
			return s.sendMessage(ctx, mem.Phone, messaging.MsgHelp)
		})

	case cleanMsg == "cancel" || cleanMsg == "stop":
		stageErr = s.executeStage(ctx, mem, msg, memberDeleteStage, func() error {
			return s.memberDelete(ctx, mem)
		})

	case cleanMsg == "pray" || mem.SetupStatus == model.SetupInProgress:
		stageErr = s.executeStage(ctx, mem, msg, signUpStage, func() error {
			return s.signUp(ctx, msg, mem)
		})

	case mem.SetupStatus == "":
		slog.WarnContext(ctx, "non registered user dropping message", "phone", mem.Phone, "msg", msg.Body)
		stageErr = s.executeStage(ctx, mem, msg, dropMessageStage, func() error {
			return nil
		})

	case cleanMsg == "prayed":
		stageErr = s.executeStage(ctx, mem, msg, completePrayerStage, func() error {
			return s.completePrayer(ctx, mem)
		})

	case mem.SetupStatus == model.SetupComplete:
		stageErr = s.executeStage(ctx, mem, msg, prayerRequestStage, func() error {
			return s.prayerRequest(ctx, msg, mem)
		})

	default:
		err := errors.New("unexpected text message input/member status")
		return apperrors.LogAndWrapError(ctx, err, "could not satisfy any required conditions", "phone", mem.Phone, "msg",
			msg.Body)
	}

	if stageErr != nil {
		return stageErr
	}

	return nil
}

// RunJobs runs all scheduled statecontroller jobs.
func (s *Service) RunJobs(ctx context.Context) {
	const (
		assignQueuedPrayersJob = "Assign Queued Prayers"
		remindActivePrayersJob = "Remind Intercessors with Active Prayers"
	)

	if err := s.AssignQueuedPrayers(ctx); err != nil {
		apperrors.LogError(ctx, err, "failed job", "job", assignQueuedPrayersJob)
	} else {
		slog.InfoContext(ctx, "finished job", "job", assignQueuedPrayersJob)
	}

	if err := s.RemindActiveIntercessors(ctx); err != nil {
		apperrors.LogError(ctx, err, "failed job", "job", remindActivePrayersJob)
	} else {
		slog.InfoContext(ctx, "finished job", "job", remindActivePrayersJob)
	}
}

func cleanStr(str string) string {
	var sb strings.Builder
	sb.Grow(len(str))
	for _, ch := range str {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			sb.WriteRune(unicode.ToLower(ch))
		}
	}

	return sb.String()
}

func (s *Service) executeStage(ctx context.Context, mem *model.Member, msg messaging.TextMessage, stageName string, stageFunc func() error) error {
	slog.InfoContext(ctx, fmt.Sprintf("Starting stage: %s", stageName), "phone", mem.Phone, "message", msg.Body)
	if stageErr := stageFunc(); stageErr != nil {
		return apperrors.LogAndWrapError(ctx, stageErr, stageErrPrefix+stageName, "phone", mem.Phone, "msg", msg.Body)
	}

	return nil
}

// sendMessage sends an SMS to a phone number.
func (s *Service) sendMessage(ctx context.Context, phone, body string) error {
	msg := messaging.TextMessage{Body: body, Phone: phone}
	return messaging.SendText(ctx, s.sms, msg)
}

// DB helper methods — these replace the CRUD methods that were on the object types.

func (s *Service) getMember(ctx context.Context, phone string) (*model.Member, error) {
	mem, err := db.GetDdbObject[model.Member](ctx, s.ddb, model.MemberKey, phone, s.cfg.AWS.DB.MemberTable)
	if err != nil {
		return nil, err
	}

	// Return a member with just the phone set if not found in DB (matches old behavior).
	if mem.Phone == "" {
		return &model.Member{Phone: phone}, nil
	}

	return mem, nil
}

func (s *Service) putMember(ctx context.Context, mem *model.Member) error {
	return db.PutDdbObject(ctx, s.ddb, s.cfg.AWS.DB.MemberTable, mem)
}

func (s *Service) deleteMember(ctx context.Context, phone string) error {
	return db.DelDdbItem(ctx, s.ddb, model.MemberKey, phone, s.cfg.AWS.DB.MemberTable)
}

func (s *Service) getBlockedPhones(ctx context.Context) (*model.BlockedPhones, error) {
	bp, err := db.GetDdbObject[model.BlockedPhones](ctx, s.ddb, model.BlockedPhonesKey, model.BlockedPhonesKeyValue,
		s.cfg.AWS.DB.BlockedPhonesTable)
	if err != nil {
		return nil, err
	}

	if bp.Key == "" {
		return &model.BlockedPhones{}, nil
	}

	return bp, nil
}

func (s *Service) putBlockedPhones(ctx context.Context, bp *model.BlockedPhones) error {
	bp.Key = model.BlockedPhonesKeyValue
	return db.PutDdbObject(ctx, s.ddb, s.cfg.AWS.DB.BlockedPhonesTable, bp)
}

func (s *Service) getIntercessorPhones(ctx context.Context) (*model.IntercessorPhones, error) {
	ip, err := db.GetDdbObject[model.IntercessorPhones](ctx, s.ddb, model.IntercessorPhonesKey,
		model.IntercessorPhonesKeyValue, s.cfg.AWS.DB.IntercessorPhonesTable)
	if err != nil {
		return nil, err
	}

	if ip.Key == "" {
		return &model.IntercessorPhones{}, nil
	}

	return ip, nil
}

func (s *Service) putIntercessorPhones(ctx context.Context, ip *model.IntercessorPhones) error {
	ip.Key = model.IntercessorPhonesKeyValue
	return db.PutDdbObject(ctx, s.ddb, s.cfg.AWS.DB.IntercessorPhonesTable, ip)
}

func (s *Service) getActivePrayer(ctx context.Context, intercessorPhone string) (*model.Prayer, error) {
	pryr, err := db.GetDdbObject[model.Prayer](ctx, s.ddb, model.PrayerKey, intercessorPhone,
		s.cfg.AWS.DB.ActivePrayerTable)
	if err != nil {
		return nil, err
	}

	if pryr.IntercessorPhone == "" {
		return &model.Prayer{IntercessorPhone: intercessorPhone}, nil
	}

	return pryr, nil
}

func (s *Service) putActivePrayer(ctx context.Context, pryr *model.Prayer) error {
	return db.PutDdbObject(ctx, s.ddb, s.cfg.AWS.DB.ActivePrayerTable, pryr)
}

func (s *Service) deleteActivePrayer(ctx context.Context, intercessorPhone string) error {
	return db.DelDdbItem(ctx, s.ddb, model.PrayerKey, intercessorPhone, s.cfg.AWS.DB.ActivePrayerTable)
}

func (s *Service) putQueuedPrayer(ctx context.Context, pryr *model.Prayer) error {
	return db.PutDdbObject(ctx, s.ddb, s.cfg.AWS.DB.QueuedPrayerTable, pryr)
}

func (s *Service) deleteQueuedPrayer(ctx context.Context, intercessorPhone string) error {
	return db.DelDdbItem(ctx, s.ddb, model.PrayerKey, intercessorPhone, s.cfg.AWS.DB.QueuedPrayerTable)
}

func (s *Service) scanActivePrayers(ctx context.Context) ([]model.Prayer, error) {
	return db.GetAllObjects[model.Prayer](ctx, s.ddb, s.cfg.AWS.DB.ActivePrayerTable)
}

func (s *Service) scanQueuedPrayers(ctx context.Context) ([]model.Prayer, error) {
	return db.GetAllObjects[model.Prayer](ctx, s.ddb, s.cfg.AWS.DB.QueuedPrayerTable)
}

func (s *Service) isPrayerActive(ctx context.Context, phone string) (bool, error) {
	pryr, err := s.getActivePrayer(ctx, phone)
	if err != nil {
		return false, apperrors.WrapError(err, "failed to check if Prayer is active")
	}

	return pryr.IsActive(), nil
}

func (s *Service) isMemberActive(ctx context.Context, phone string) (bool, error) {
	mem, err := s.getMember(ctx, phone)
	if err != nil {
		return false, apperrors.WrapError(err, "failed to check if Member is active")
	}

	return mem.IsActive(), nil
}
