package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/utility"
)

type Router struct {
	members   repository.MemberRepository
	blocked   repository.BlockedPhonesRepository
	memberSvc *MemberService
	prayerSvc *PrayerService
	adminSvc  *AdminService
}

func NewRouter(
	members repository.MemberRepository,
	blocked repository.BlockedPhonesRepository,
	memberSvc *MemberService,
	prayerSvc *PrayerService,
	adminSvc *AdminService,
) *Router {
	return &Router{
		members:   members,
		blocked:   blocked,
		memberSvc: memberSvc,
		prayerSvc: prayerSvc,
		adminSvc:  adminSvc,
	}
}

func (r *Router) Handle(ctx context.Context, msg domain.TextMessage) error {
	mem, err := r.members.Get(ctx, msg.Phone)
	if err != nil {
		return utility.LogAndWrapError(ctx, err, "failure during stage PRE", "phone", msg.Phone, "msg", msg.Body)
	}

	blockedPhones, err := r.blocked.Get(ctx)
	if err != nil {
		return utility.LogAndWrapError(ctx, err, "failure during stage PRE", "phone", msg.Phone, "msg", msg.Body)
	}

	isBlocked := slices.Contains(blockedPhones.Phones, mem.Phone)
	cleanMsg := cleanStr(msg.Body)

	var stageName string
	var stageErr error

	switch {
	case isBlocked:
		stageName = "BLOCKED USER"
		slog.WarnContext(ctx, "blocked user dropping message", "phone", mem.Phone, "msg", msg.Body)

	case strings.Contains(strings.ToLower(msg.Body), "#block"):
		stageName = "ADD BLOCKED USER"
		stageErr = r.adminSvc.BlockUser(ctx, msg, *mem, blockedPhones)

	case cleanMsg == "help":
		stageName = "HELP"
		stageErr = r.memberSvc.Help(ctx, *mem)

	case cleanMsg == "cancel" || cleanMsg == "stop":
		stageName = "MEMBER DELETE"
		stageErr = r.memberSvc.Delete(ctx, *mem)

	case cleanMsg == "pray" || mem.SetupStatus == domain.MemberSetupInProgress:
		stageName = "SIGN UP"
		stageErr = r.memberSvc.SignUp(ctx, msg, *mem)

	case mem.SetupStatus == "":
		stageName = "DROP MESSAGE"
		slog.WarnContext(ctx, "non registered user dropping message", "phone", mem.Phone, "msg", msg.Body)

	case cleanMsg == "prayed":
		stageName = "COMPLETE PRAYER"
		stageErr = r.prayerSvc.Complete(ctx, *mem)

	case mem.SetupStatus == domain.MemberSetupComplete:
		stageName = "PRAYER REQUEST"
		stageErr = r.prayerSvc.Request(ctx, msg, *mem)

	default:
		err = errors.New("unexpected text message input/member status")
		return utility.LogAndWrapError(ctx, err, "could not satisfy any required conditions", "phone", mem.Phone, "msg", msg.Body)
	}

	slog.InfoContext(ctx, fmt.Sprintf("Starting stage: %s", stageName), "phone", mem.Phone, "message", msg.Body)
	if stageErr != nil {
		return utility.LogAndWrapError(ctx, stageErr, "failure during stage "+stageName, "phone", mem.Phone, "msg", msg.Body)
	}

	return nil
}
