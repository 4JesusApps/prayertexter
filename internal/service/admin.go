package service

import (
	"context"
	"errors"
	"regexp"
	"slices"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
)

type AdminService struct {
	members   repository.MemberRepository
	blocked   repository.BlockedPhonesRepository
	sender    messaging.MessageSender
	memberSvc *MemberService
}

func NewAdminService(
	members repository.MemberRepository,
	blocked repository.BlockedPhonesRepository,
	sender messaging.MessageSender,
	memberSvc *MemberService,
) *AdminService {
	return &AdminService{
		members:   members,
		blocked:   blocked,
		sender:    sender,
		memberSvc: memberSvc,
	}
}

func (s *AdminService) BlockUser(ctx context.Context, msg domain.TextMessage, mem domain.Member, blockedPhones *domain.BlockedPhones) error {
	if !mem.Administrator {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgUnauthorized)
	}

	phone, err := extractPhone(msg.Body)
	if errors.Is(err, ErrInvalidPhone) {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgInvalidPhone)
	}

	phone = "+1" + phone
	if slices.Contains(blockedPhones.Phones, phone) {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgUserAlreadyBlocked)
	}

	blockedPhones.AddPhone(phone)
	if err = s.blocked.Save(ctx, blockedPhones); err != nil {
		return err
	}

	blockedUser, err := s.members.Get(ctx, phone)
	if err != nil {
		return err
	}

	if err = s.memberSvc.Delete(ctx, *blockedUser); err != nil {
		return err
	}

	if err = s.sender.SendMessage(ctx, phone, messaging.MsgBlockedNotification+messaging.MsgHelp); err != nil {
		return err
	}

	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgSuccessfullyBlocked)
}

var phoneRE = regexp.MustCompile(`\(?\b(\d{3})\)?[\s\-]?(\d{3})[\s\-]?(\d{4})\b`)

func extractPhone(msg string) (string, error) {

	matchNum := 4
	matches := phoneRE.FindStringSubmatch(msg)
	if len(matches) != matchNum {
		return "", ErrInvalidPhone
	}

	return matches[1] + matches[2] + matches[3], nil
}
