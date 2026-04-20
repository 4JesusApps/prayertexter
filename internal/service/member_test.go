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

type MemberServiceSuite struct {
	suite.Suite
	svc          *service.MemberService
	members      *repomocks.MockMemberRepository
	intercessors *repomocks.MockIntercessorPhonesRepository
	prayers      *repomocks.MockPrayerRepository
	sender       *msgmocks.MockMessageSender
	ctx          context.Context
}

func (s *MemberServiceSuite) SetupTest() {
	s.members = repomocks.NewMockMemberRepository(s.T())
	s.intercessors = repomocks.NewMockIntercessorPhonesRepository(s.T())
	s.prayers = repomocks.NewMockPrayerRepository(s.T())
	s.sender = msgmocks.NewMockMessageSender(s.T())
	s.ctx = context.Background()
	s.svc = service.NewMemberService(s.members, s.intercessors, s.prayers, s.sender, config.Config{
		IntercessorsPerPrayer: 2,
	})
}

func (s *MemberServiceSuite) TestHelp() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgHelp).Return(nil)

	err := s.svc.Help(s.ctx, domain.Member{Phone: "+11234567890"})
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpStageOne() {
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.SetupStatus == domain.MemberSetupInProgress && m.SetupStage == domain.MemberSignUpStepOne
	})).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgNameRequest).Return(nil)

	err := s.svc.SignUp(
		s.ctx,
		domain.TextMessage{Body: "pray", Phone: "+11234567890"},
		domain.Member{Phone: "+11234567890"},
	)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpStageTwo_ValidName() {
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.Name == "John Doe" && m.SetupStage == domain.MemberSignUpStepTwo
	})).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgMemberTypeRequest).Return(nil)

	mem := domain.Member{
		Phone:       "+11234567890",
		SetupStage:  domain.MemberSignUpStepOne,
		SetupStatus: domain.MemberSetupInProgress,
	}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "John Doe", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpStageTwo_InvalidName() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgInvalidName).Return(nil)

	mem := domain.Member{
		Phone:       "+11234567890",
		SetupStage:  domain.MemberSignUpStepOne,
		SetupStatus: domain.MemberSetupInProgress,
	}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "1", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpFinalPrayer() {
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.SetupStatus == domain.MemberSetupComplete && !m.Intercessor
	})).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", mock.Anything).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStage: domain.MemberSignUpStepTwo}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "1", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestDelete_NonIntercessor() {
	s.members.EXPECT().Delete(s.ctx, "+11234567890").Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgRemoveUser).Return(nil)

	err := s.svc.Delete(s.ctx, domain.Member{Phone: "+11234567890", Intercessor: false})
	s.NoError(err)
}

func (s *MemberServiceSuite) TestDelete_Intercessor_NoActivePrayer() {
	s.members.EXPECT().Delete(s.ctx, "+11234567890").Return(nil)
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{
		Key:    "IntercessorPhones",
		Phones: []string{"+11234567890", "+19999999999"},
	}, nil)
	s.intercessors.EXPECT().Save(s.ctx, mock.MatchedBy(func(p *domain.IntercessorPhones) bool {
		return len(p.Phones) == 1 && p.Phones[0] == "+19999999999"
	})).Return(nil)
	s.prayers.EXPECT().Exists(s.ctx, "+11234567890").Return(false, nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgRemoveUser).Return(nil)

	err := s.svc.Delete(s.ctx, domain.Member{Phone: "+11234567890", Intercessor: true})
	s.NoError(err)
}

func TestMemberServiceSuite(t *testing.T) {
	suite.Run(t, new(MemberServiceSuite))
}
