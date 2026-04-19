package service_test

import (
	"context"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	msgmocks "github.com/4JesusApps/prayertexter/internal/mocks/messaging"
	repomocks "github.com/4JesusApps/prayertexter/internal/mocks/repository"
)

type AdminServiceSuite struct {
	suite.Suite
	svc          *service.AdminService
	memberSvc    *service.MemberService
	members      *repomocks.MockMemberRepository
	blocked      *repomocks.MockBlockedPhonesRepository
	intercessors *repomocks.MockIntercessorPhonesRepository
	prayers      *repomocks.MockPrayerRepository
	sender       *msgmocks.MockMessageSender
	ctx          context.Context
}

func (s *AdminServiceSuite) SetupTest() {
	s.members = repomocks.NewMockMemberRepository(s.T())
	s.blocked = repomocks.NewMockBlockedPhonesRepository(s.T())
	s.intercessors = repomocks.NewMockIntercessorPhonesRepository(s.T())
	s.prayers = repomocks.NewMockPrayerRepository(s.T())
	s.sender = msgmocks.NewMockMessageSender(s.T())
	s.ctx = context.Background()
	s.memberSvc = service.NewMemberService(s.members, s.intercessors, s.prayers, s.sender, config.Config{})
	s.svc = service.NewAdminService(s.members, s.blocked, s.sender, s.memberSvc)
}

func (s *AdminServiceSuite) TestBlockUser_NotAdmin() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgUnauthorized).Return(nil)

	mem := domain.Member{Phone: "+11234567890", Administrator: false}
	blocked := &domain.BlockedPhones{}
	err := s.svc.BlockUser(s.ctx, domain.TextMessage{Body: "#block 777-777-7777"}, mem, blocked)
	s.NoError(err)
}

func (s *AdminServiceSuite) TestBlockUser_InvalidPhone() {
	s.sender.EXPECT().SendMessage(s.ctx, "+17777777777", messaging.MsgInvalidPhone).Return(nil)

	mem := domain.Member{Phone: "+17777777777", Administrator: true}
	blocked := &domain.BlockedPhones{}
	err := s.svc.BlockUser(s.ctx, domain.TextMessage{Body: "#block 123"}, mem, blocked)
	s.NoError(err)
}

func (s *AdminServiceSuite) TestBlockUser_AlreadyBlocked() {
	s.sender.EXPECT().SendMessage(s.ctx, "+17777777777", messaging.MsgUserAlreadyBlocked).Return(nil)

	mem := domain.Member{Phone: "+17777777777", Administrator: true}
	blocked := &domain.BlockedPhones{Phones: []string{"+11234567890"}}
	err := s.svc.BlockUser(s.ctx, domain.TextMessage{Body: "#block 123-456-7890"}, mem, blocked)
	s.NoError(err)
}

func (s *AdminServiceSuite) TestBlockUser_Success_NonIntercessor() {
	s.blocked.EXPECT().Save(s.ctx, mock.Anything).Return(nil)
	s.members.EXPECT().Get(s.ctx, "+11234567890").Return(&domain.Member{
		Phone: "+11234567890", Name: "Bad User",
	}, nil)
	s.members.EXPECT().Delete(s.ctx, "+11234567890").Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgRemoveUser).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgBlockedNotification+messaging.MsgHelp).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+17777777777", messaging.MsgSuccessfullyBlocked).Return(nil)

	mem := domain.Member{Phone: "+17777777777", Administrator: true}
	blocked := &domain.BlockedPhones{Phones: []string{"+12222222222"}}
	err := s.svc.BlockUser(s.ctx, domain.TextMessage{Body: "#block 123-456-7890"}, mem, blocked)
	s.NoError(err)
}

func TestAdminServiceSuite(t *testing.T) {
	suite.Run(t, new(AdminServiceSuite))
}
