package service_test

import (
	"context"
	"strings"
	"testing"
	"time"

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
	confirmMsg, _ := messaging.Render(messaging.PrayerConfirmationTmpl, struct{ Name string }{"Intercessor"})
	s.sender.EXPECT().SendMessage(s.ctx, "+19999999999", confirmMsg).Return(nil)
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
	s.prayers.EXPECT().Save(s.ctx, mock.MatchedBy(func(p *domain.Prayer) bool {
		return p.Request == "please pray for my health and well being today" &&
			p.Requestor.Phone == "+11234567890" &&
			p.IntercessorPhone != ""
	}), true).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgPrayerQueued).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStatus: domain.MemberSetupComplete}
	err := s.svc.Request(
		s.ctx,
		domain.TextMessage{Body: "please pray for my health and well being today", Phone: "+11234567890"},
		mem,
	)
	s.NoError(err)
}

func (s *PrayerServiceSuite) TestFindIntercessors_UnderLimit() {
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{
		Phones: []string{"+18888888888", "+19999999999"},
	}, nil)
	s.members.EXPECT().Get(s.ctx, "+18888888888").Return(&domain.Member{
		Phone: "+18888888888", PrayerCount: 0, WeeklyPrayerLimit: 5,
	}, nil)
	s.prayers.EXPECT().Exists(s.ctx, "+18888888888").Return(false, nil)
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.Phone == "+18888888888" && m.PrayerCount == 1
	})).Return(nil)
	s.members.EXPECT().Get(s.ctx, "+19999999999").Return(&domain.Member{
		Phone: "+19999999999", PrayerCount: 0, WeeklyPrayerLimit: 5,
	}, nil)
	s.prayers.EXPECT().Exists(s.ctx, "+19999999999").Return(false, nil)
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.Phone == "+19999999999" && m.PrayerCount == 1
	})).Return(nil)

	result, err := s.svc.FindIntercessors(s.ctx, "+11234567890")
	s.Require().NoError(err)
	s.Len(result, 2)
}

func (s *PrayerServiceSuite) TestFindIntercessors_AtLimit_ResetEligible() {
	oldDate := time.Now().Add(-8 * 24 * time.Hour).Format(time.RFC3339)

	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{
		Phones: []string{"+18888888888", "+19999999999"},
	}, nil)
	s.members.EXPECT().Get(s.ctx, "+18888888888").Return(&domain.Member{
		Phone: "+18888888888", PrayerCount: 5, WeeklyPrayerLimit: 5, WeeklyPrayerDate: oldDate,
	}, nil)
	s.prayers.EXPECT().Exists(s.ctx, "+18888888888").Return(false, nil)
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.Phone == "+18888888888" && m.PrayerCount == 1
	})).Return(nil)
	s.members.EXPECT().Get(s.ctx, "+19999999999").Return(&domain.Member{
		Phone: "+19999999999", PrayerCount: 5, WeeklyPrayerLimit: 5, WeeklyPrayerDate: oldDate,
	}, nil)
	s.prayers.EXPECT().Exists(s.ctx, "+19999999999").Return(false, nil)
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.Phone == "+19999999999" && m.PrayerCount == 1
	})).Return(nil)

	result, err := s.svc.FindIntercessors(s.ctx, "+11234567890")
	s.Require().NoError(err)
	s.Len(result, 2)
}

func (s *PrayerServiceSuite) TestFindIntercessors_AllHaveActivePrayers() {
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{
		Phones: []string{"+18888888888", "+19999999999"},
	}, nil)
	s.members.EXPECT().Get(s.ctx, "+18888888888").Return(&domain.Member{
		Phone: "+18888888888",
	}, nil)
	s.prayers.EXPECT().Exists(s.ctx, "+18888888888").Return(true, nil)
	s.members.EXPECT().Get(s.ctx, "+19999999999").Return(&domain.Member{
		Phone: "+19999999999",
	}, nil)
	s.prayers.EXPECT().Exists(s.ctx, "+19999999999").Return(true, nil)

	_, err := s.svc.FindIntercessors(s.ctx, "+11234567890")
	s.ErrorIs(err, service.ErrNoAvailableIntercessors)
}

func (s *PrayerServiceSuite) TestFindIntercessors_AtLimit_NotResetEligible() {
	recentDate := time.Now().Format(time.RFC3339)

	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{
		Phones: []string{"+18888888888", "+19999999999"},
	}, nil)
	s.members.EXPECT().Get(s.ctx, "+18888888888").Return(&domain.Member{
		Phone: "+18888888888", PrayerCount: 5, WeeklyPrayerLimit: 5, WeeklyPrayerDate: recentDate,
	}, nil)
	s.prayers.EXPECT().Exists(s.ctx, "+18888888888").Return(false, nil)
	s.members.EXPECT().Get(s.ctx, "+19999999999").Return(&domain.Member{
		Phone: "+19999999999", PrayerCount: 5, WeeklyPrayerLimit: 5, WeeklyPrayerDate: recentDate,
	}, nil)
	s.prayers.EXPECT().Exists(s.ctx, "+19999999999").Return(false, nil)

	_, err := s.svc.FindIntercessors(s.ctx, "+11234567890")
	s.ErrorIs(err, service.ErrNoAvailableIntercessors)
}

func (s *PrayerServiceSuite) TestRequest_WithAnon() {
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{
		Phones: []string{"+18888888888", "+19999999999"},
	}, nil)
	s.members.EXPECT().Get(s.ctx, "+18888888888").Return(&domain.Member{
		Phone: "+18888888888", PrayerCount: 0, WeeklyPrayerLimit: 5,
	}, nil)
	s.prayers.EXPECT().Exists(s.ctx, "+18888888888").Return(false, nil)
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.Phone == "+18888888888"
	})).Return(nil)
	s.members.EXPECT().Get(s.ctx, "+19999999999").Return(&domain.Member{
		Phone: "+19999999999", PrayerCount: 0, WeeklyPrayerLimit: 5,
	}, nil)
	s.prayers.EXPECT().Exists(s.ctx, "+19999999999").Return(false, nil)
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.Phone == "+19999999999"
	})).Return(nil)

	s.prayers.EXPECT().Save(s.ctx, mock.MatchedBy(func(p *domain.Prayer) bool {
		return p.Requestor.Name == "Anonymous" && !strings.Contains(p.Request, "#anon")
	}), false).Return(nil).Times(2)
	introMsg, _ := messaging.Render(messaging.PrayerIntroTmpl, struct{ Name string }{"Anonymous"})
	expectedPrayerMsg := introMsg + "please pray for my family and friends" + "\n\n" + messaging.MsgPrayed
	s.sender.EXPECT().SendMessage(s.ctx, "+18888888888", expectedPrayerMsg).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+19999999999", expectedPrayerMsg).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgPrayerAssigned).Return(nil)

	mem := domain.Member{Phone: "+11234567890", Name: "RealName", SetupStatus: domain.MemberSetupComplete}
	err := s.svc.Request(
		s.ctx,
		domain.TextMessage{Body: "please pray for my family and friends #anon", Phone: "+11234567890"},
		mem,
	)
	s.NoError(err)
}

func (s *PrayerServiceSuite) TestAssignQueuedPrayers_Success() {
	queuedPrayer := domain.Prayer{
		IntercessorPhone: "queue-id-123",
		Request:          "please pray for me and my family today",
		Requestor:        domain.Member{Phone: "+11234567890", Name: "Requestor"},
	}

	s.prayers.EXPECT().GetAll(s.ctx, true).Return([]domain.Prayer{queuedPrayer}, nil)

	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{
		Phones: []string{"+18888888888", "+19999999999"},
	}, nil)
	s.members.EXPECT().Get(s.ctx, "+18888888888").Return(&domain.Member{
		Phone: "+18888888888", Name: "I1", PrayerCount: 0, WeeklyPrayerLimit: 5,
	}, nil)
	s.prayers.EXPECT().Exists(s.ctx, "+18888888888").Return(false, nil)
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.Phone == "+18888888888"
	})).Return(nil)
	s.members.EXPECT().Get(s.ctx, "+19999999999").Return(&domain.Member{
		Phone: "+19999999999", Name: "I2", PrayerCount: 0, WeeklyPrayerLimit: 5,
	}, nil)
	s.prayers.EXPECT().Exists(s.ctx, "+19999999999").Return(false, nil)
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.Phone == "+19999999999"
	})).Return(nil)

	s.prayers.EXPECT().Save(s.ctx, mock.MatchedBy(func(p *domain.Prayer) bool {
		return p.Request == "please pray for me and my family today" &&
			p.Requestor.Phone == "+11234567890"
	}), false).Return(nil).Times(2)
	introMsg, _ := messaging.Render(messaging.PrayerIntroTmpl, struct{ Name string }{"Requestor"})
	expectedPrayerMsg := introMsg + "please pray for me and my family today" + "\n\n" + messaging.MsgPrayed
	s.sender.EXPECT().SendMessage(s.ctx, "+18888888888", expectedPrayerMsg).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+19999999999", expectedPrayerMsg).Return(nil)

	s.prayers.EXPECT().Delete(s.ctx, "queue-id-123", true).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgPrayerAssigned).Return(nil)

	err := s.svc.AssignQueuedPrayers(s.ctx)
	s.NoError(err)
}

func (s *PrayerServiceSuite) TestRemindActiveIntercessors() {
	oldDate := time.Now().Add(-4 * time.Hour).Format(time.RFC3339)
	recentDate := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)

	prayers := []domain.Prayer{
		{
			IntercessorPhone: "+18888888888",
			Request:          "prayer 1",
			Requestor:        domain.Member{Name: "R1"},
			Intercessor:      domain.Member{Phone: "+18888888888"},
		},
		{
			IntercessorPhone: "+19999999999",
			Request:          "prayer 2",
			Requestor:        domain.Member{Name: "R2"},
			Intercessor:      domain.Member{Phone: "+19999999999"},
			ReminderDate:     oldDate,
		},
		{
			IntercessorPhone: "+17777777777",
			Request:          "prayer 3",
			Requestor:        domain.Member{Name: "R3"},
			Intercessor:      domain.Member{Phone: "+17777777777"},
			ReminderDate:     recentDate,
		},
	}

	s.prayers.EXPECT().GetAll(s.ctx, false).Return(prayers, nil)

	s.prayers.EXPECT().Save(s.ctx, mock.MatchedBy(func(p *domain.Prayer) bool {
		return p.IntercessorPhone == "+18888888888" && p.ReminderDate != ""
	}), false).Return(nil)

	s.prayers.EXPECT().Save(s.ctx, mock.MatchedBy(func(p *domain.Prayer) bool {
		return p.IntercessorPhone == "+19999999999" && p.ReminderCount == 1
	}), false).Return(nil)
	reminderMsg, _ := messaging.Render(messaging.PrayerReminderTmpl, struct{ Name string }{"R2"})
	expectedReminderMsg := reminderMsg + "prayer 2" + "\n\n" + messaging.MsgPrayed
	s.sender.EXPECT().SendMessage(s.ctx, "+19999999999", expectedReminderMsg).Return(nil)

	err := s.svc.RemindActiveIntercessors(s.ctx)
	s.NoError(err)
}

func TestPrayerServiceSuite(t *testing.T) {
	suite.Run(t, new(PrayerServiceSuite))
}
