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

type PrayerServiceSuite struct {
	suite.Suite
	svc          *service.PrayerService
	members      *repomocks.MockMemberRepository
	intercessors *repomocks.MockIntercessorPhonesRepository
	prayers      *repomocks.MockPrayerRepository
	sender       *msgmocks.MockMessageSender
	ctx          context.Context
}

func (s *PrayerServiceSuite) SetupTest() {
	s.members = repomocks.NewMockMemberRepository(s.T())
	s.intercessors = repomocks.NewMockIntercessorPhonesRepository(s.T())
	s.prayers = repomocks.NewMockPrayerRepository(s.T())
	s.sender = msgmocks.NewMockMessageSender(s.T())
	s.ctx = context.Background()
	s.svc = service.NewPrayerService(s.members, s.intercessors, s.prayers, s.sender, config.Config{
		IntercessorsPerPrayer: 2,
		PrayerReminderHours:   3,
	})
}

func (s *PrayerServiceSuite) TestComplete_NoActivePrayer() {
	s.prayers.EXPECT().Get(s.ctx, "+11234567890", false).Return(&domain.Prayer{}, nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgNoActivePrayer).Return(nil)

	err := s.svc.Complete(s.ctx, domain.Member{Phone: "+11234567890"})
	s.NoError(err)
}

func (s *PrayerServiceSuite) TestComplete_WithActivePrayer() {
	requestor := domain.Member{Phone: "+19999999999", Name: "Requestor"}
	intercessor := domain.Member{Phone: "+11234567890", Name: "Intercessor"}

	s.prayers.EXPECT().Get(s.ctx, "+11234567890", false).Return(&domain.Prayer{
		Request:          "Please pray for me",
		Requestor:        requestor,
		IntercessorPhone: "+11234567890",
		Intercessor:      intercessor,
	}, nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgPrayerThankYou).Return(nil)
	s.members.EXPECT().Exists(s.ctx, "+19999999999").Return(true, nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+19999999999", mock.Anything).Return(nil)
	s.prayers.EXPECT().Delete(s.ctx, "+11234567890", false).Return(nil)

	err := s.svc.Complete(s.ctx, intercessor)
	s.NoError(err)
}

func (s *PrayerServiceSuite) TestRequest_InvalidRequest() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgInvalidRequest).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStatus: domain.MemberSetupComplete}
	err := s.svc.Request(s.ctx, domain.TextMessage{Body: "pray", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *PrayerServiceSuite) TestRequest_Queued() {
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{Phones: []string{}}, nil)
	s.prayers.EXPECT().Save(s.ctx, mock.Anything, true).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgPrayerQueued).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStatus: domain.MemberSetupComplete}
	err := s.svc.Request(
		s.ctx,
		domain.TextMessage{Body: "please pray for my health and well being today", Phone: "+11234567890"},
		mem,
	)
	s.NoError(err)
}

func TestPrayerServiceSuite(t *testing.T) {
	suite.Run(t, new(PrayerServiceSuite))
}
