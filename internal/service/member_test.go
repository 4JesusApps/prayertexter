package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
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
	expectedBody := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgSignUpConfirmation
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", expectedBody).Return(nil)

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
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{
		Phones: []string{"+11234567890", "+19999999999"},
	}, nil)
	s.intercessors.EXPECT().Save(s.ctx, mock.MatchedBy(func(p *domain.IntercessorPhones) bool {
		return len(p.Phones) == 1 && p.Phones[0] == "+19999999999"
	})).Return(nil)
	s.prayers.EXPECT().Get(s.ctx, "+11234567890", false).Return(nil, repository.ErrNotFound)
	s.members.EXPECT().Delete(s.ctx, "+11234567890").Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgRemoveUser).Return(nil)

	err := s.svc.Delete(s.ctx, domain.Member{Phone: "+11234567890", Intercessor: true})
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpStageThree() {
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.SetupStage == domain.MemberSignUpStepThree && m.Intercessor
	})).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgPrayerNumRequest).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStage: domain.MemberSignUpStepTwo}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "2", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpFinalIntercessor() {
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{
		Phones: []string{"+19999999999"},
	}, nil)
	s.intercessors.EXPECT().Save(s.ctx, mock.MatchedBy(func(p *domain.IntercessorPhones) bool {
		return len(p.Phones) == 2
	})).Return(nil)
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.SetupStatus == domain.MemberSetupComplete &&
			m.SetupStage == domain.MemberSignUpStepFinal &&
			m.WeeklyPrayerLimit == 5
	})).Return(nil)
	expectedBody := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgIntercessorInstructions + "\n\n" +
		messaging.MsgSignUpConfirmation
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", expectedBody).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStage: domain.MemberSignUpStepThree}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "5", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpFinalIntercessor_WrongInput() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgWrongInput).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStage: domain.MemberSignUpStepThree}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "abc", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpFinalIntercessor_ZeroLimit() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgInvalidPrayerLimit).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStage: domain.MemberSignUpStepThree}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "0", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestDelete_Intercessor_WithActivePrayer() {
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{
		Phones: []string{"+11234567890"},
	}, nil)
	s.intercessors.EXPECT().Save(s.ctx, mock.MatchedBy(func(p *domain.IntercessorPhones) bool {
		return len(p.Phones) == 0
	})).Return(nil)
	s.prayers.EXPECT().Get(s.ctx, "+11234567890", false).Return(&domain.Prayer{
		Request:          "original prayer",
		IntercessorPhone: "+11234567890",
		Requestor:        domain.Member{Phone: "+19999999999"},
		ReminderCount:    2,
		ReminderDate:     "2026-01-02T15:04:05Z",
	}, nil)
	s.prayers.EXPECT().Delete(s.ctx, "+11234567890", false).Return(nil)
	s.prayers.EXPECT().Save(s.ctx, mock.MatchedBy(func(p *domain.Prayer) bool {
		return p.Request == "original prayer" &&
			p.IntercessorPhone != "+11234567890" &&
			p.Intercessor == domain.Member{} &&
			p.QueueID == p.IntercessorPhone &&
			p.ReminderCount == 0 &&
			p.ReminderDate == "" &&
			!p.RequestorNotified
	}), true).Return(nil)
	s.members.EXPECT().Delete(s.ctx, "+11234567890").Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgRemoveUser).Return(nil)

	err := s.svc.Delete(s.ctx, domain.Member{Phone: "+11234567890", Intercessor: true})
	s.NoError(err)
}

func (s *MemberServiceSuite) TestDelete_Intercessor_CleanupFailureDoesNotDeleteMember() {
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{
		Phones: []string{"+11234567890"},
	}, nil)
	s.intercessors.EXPECT().Save(s.ctx, mock.Anything).Return(nil)
	s.prayers.EXPECT().Get(s.ctx, "+11234567890", false).Return(nil, errors.New("boom"))
	s.intercessors.EXPECT().Save(s.ctx, mock.MatchedBy(func(p *domain.IntercessorPhones) bool {
		return len(p.Phones) == 1 && p.Phones[0] == "+11234567890"
	})).Return(nil)

	err := s.svc.Delete(s.ctx, domain.Member{Phone: "+11234567890", Intercessor: true})
	s.Error(err)
}

func (s *MemberServiceSuite) TestSignUpStageOne_CaseInsensitive() {
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.SetupStatus == domain.MemberSetupInProgress && m.SetupStage == domain.MemberSignUpStepOne
	})).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgNameRequest).Return(nil)

	err := s.svc.SignUp(
		s.ctx,
		domain.TextMessage{Body: "Pray", Phone: "+11234567890"},
		domain.Member{Phone: "+11234567890"},
	)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpStageOne_StripsInvalidChars() {
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.SetupStatus == domain.MemberSetupInProgress && m.SetupStage == domain.MemberSignUpStepOne
	})).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgNameRequest).Return(nil)

	err := s.svc.SignUp(
		s.ctx,
		domain.TextMessage{Body: "  P!r@a$y!!  ", Phone: "+11234567890"},
		domain.Member{Phone: "+11234567890"},
	)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpStageTwo_Profanity() {
	rendered, _ := messaging.Render(messaging.ProfanityDetectedTmpl, struct{ Word string }{"fuck"})
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", rendered).Return(nil)

	mem := domain.Member{
		Phone:       "+11234567890",
		SetupStage:  domain.MemberSignUpStepOne,
		SetupStatus: domain.MemberSetupInProgress,
	}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "fuck", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpStageTwo_NameTooShort() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgInvalidName).Return(nil)

	mem := domain.Member{
		Phone:       "+11234567890",
		SetupStage:  domain.MemberSignUpStepOne,
		SetupStatus: domain.MemberSetupInProgress,
	}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "A", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpStageTwo_Anonymous() {
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.Name == "Anonymous" && m.SetupStage == domain.MemberSignUpStepTwo
	})).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgMemberTypeRequest).Return(nil)

	mem := domain.Member{
		Phone:       "+11234567890",
		SetupStage:  domain.MemberSignUpStepOne,
		SetupStatus: domain.MemberSetupInProgress,
	}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "2", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpFinalIntercessor_StripsCommas() {
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{}, nil)
	s.intercessors.EXPECT().Save(s.ctx, mock.Anything).Return(nil)
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.WeeklyPrayerLimit == 10
	})).Return(nil)
	expectedBody := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgIntercessorInstructions + "\n\n" +
		messaging.MsgSignUpConfirmation
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", expectedBody).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStage: domain.MemberSignUpStepThree}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "1,0", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpFinalIntercessor_IntercessorPhonesSaveFails() {
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{}, nil)
	s.intercessors.EXPECT().Save(s.ctx, mock.Anything).Return(errors.New("save failed"))

	mem := domain.Member{Phone: "+11234567890", SetupStage: domain.MemberSignUpStepThree}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "5", Phone: "+11234567890"}, mem)
	s.Error(err)
}

func (s *MemberServiceSuite) TestSignUp_WrongMemberTypeReply() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgWrongInput).Return(nil)

	mem := domain.Member{
		Phone:       "+11234567890",
		SetupStage:  domain.MemberSignUpStepTwo,
		SetupStatus: domain.MemberSetupInProgress,
	}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "3", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestDelete_MemberDeleteFails() {
	s.members.EXPECT().Delete(s.ctx, "+11234567890").Return(errors.New("boom"))

	err := s.svc.Delete(s.ctx, domain.Member{Phone: "+11234567890", Intercessor: false})
	s.Error(err)
}

func (s *MemberServiceSuite) TestHelp_DuringSignUp() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgHelp).Return(nil)

	mem := domain.Member{
		Phone:       "+11234567890",
		SetupStage:  domain.MemberSignUpStepOne,
		SetupStatus: domain.MemberSetupInProgress,
	}
	err := s.svc.Help(s.ctx, mem)
	s.NoError(err)
}

func TestMemberServiceSuite(t *testing.T) {
	suite.Run(t, new(MemberServiceSuite))
}
