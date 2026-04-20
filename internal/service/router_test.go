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

type RouterSuite struct {
	suite.Suite
	router       *service.Router
	members      *repomocks.MockMemberRepository
	blocked      *repomocks.MockBlockedPhonesRepository
	intercessors *repomocks.MockIntercessorPhonesRepository
	prayers      *repomocks.MockPrayerRepository
	sender       *msgmocks.MockMessageSender
	ctx          context.Context
}

func (s *RouterSuite) SetupTest() {
	s.members = repomocks.NewMockMemberRepository(s.T())
	s.blocked = repomocks.NewMockBlockedPhonesRepository(s.T())
	s.intercessors = repomocks.NewMockIntercessorPhonesRepository(s.T())
	s.prayers = repomocks.NewMockPrayerRepository(s.T())
	s.sender = msgmocks.NewMockMessageSender(s.T())
	s.ctx = context.Background()

	cfg := config.Config{IntercessorsPerPrayer: 2, PrayerReminderHours: 3}
	memberSvc := service.NewMemberService(s.members, s.intercessors, s.prayers, s.sender, cfg)
	prayerSvc := service.NewPrayerService(s.members, s.intercessors, s.prayers, s.sender, cfg)
	adminSvc := service.NewAdminService(s.members, s.blocked, s.sender, memberSvc)

	s.router = service.NewRouter(s.members, s.blocked, memberSvc, prayerSvc, adminSvc)
}

func (s *RouterSuite) TestRouteHelp() {
	s.members.EXPECT().Get(s.ctx, "+11234567890").Return(&domain.Member{Phone: "+11234567890"}, nil)
	s.blocked.EXPECT().Get(s.ctx).Return(&domain.BlockedPhones{}, nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgHelp).Return(nil)

	err := s.router.Handle(s.ctx, domain.TextMessage{Body: "HELP", Phone: "+11234567890"})
	s.NoError(err)
}

func (s *RouterSuite) TestRouteBlockedUser() {
	s.members.EXPECT().Get(s.ctx, "+11234567890").Return(&domain.Member{Phone: "+11234567890"}, nil)
	s.blocked.EXPECT().Get(s.ctx).Return(&domain.BlockedPhones{Phones: []string{"+11234567890"}}, nil)

	err := s.router.Handle(s.ctx, domain.TextMessage{Body: "anything", Phone: "+11234567890"})
	s.NoError(err)
}

func (s *RouterSuite) TestRouteSignUp() {
	s.members.EXPECT().Get(s.ctx, "+11234567890").Return(&domain.Member{Phone: "+11234567890"}, nil)
	s.blocked.EXPECT().Get(s.ctx).Return(&domain.BlockedPhones{}, nil)
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.SetupStatus == domain.MemberSetupInProgress
	})).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgNameRequest).Return(nil)

	err := s.router.Handle(s.ctx, domain.TextMessage{Body: "pray", Phone: "+11234567890"})
	s.NoError(err)
}

func (s *RouterSuite) TestRouteDropMessage() {
	s.members.EXPECT().Get(s.ctx, "+11234567890").Return(&domain.Member{Phone: "+11234567890"}, nil)
	s.blocked.EXPECT().Get(s.ctx).Return(&domain.BlockedPhones{}, nil)

	err := s.router.Handle(s.ctx, domain.TextMessage{Body: "random text", Phone: "+11234567890"})
	s.NoError(err)
}

func TestRouterSuite(t *testing.T) {
	suite.Run(t, new(RouterSuite))
}
