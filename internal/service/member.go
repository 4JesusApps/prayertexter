package service

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
)

type MemberService struct {
	members      repository.MemberRepository
	intercessors repository.IntercessorPhonesRepository
	prayers      repository.PrayerRepository
	sender       messaging.MessageSender
	cfg          config.Config
}

func NewMemberService(
	members repository.MemberRepository,
	intercessors repository.IntercessorPhonesRepository,
	prayers repository.PrayerRepository,
	sender messaging.MessageSender,
	cfg config.Config,
) *MemberService {
	return &MemberService{
		members:      members,
		intercessors: intercessors,
		prayers:      prayers,
		sender:       sender,
		cfg:          cfg,
	}
}

func (s *MemberService) Help(ctx context.Context, mem domain.Member) error {
	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgHelp)
}

func (s *MemberService) Delete(ctx context.Context, mem domain.Member) error {
	if err := s.members.Delete(ctx, mem.Phone); err != nil {
		return err
	}
	if mem.Intercessor {
		if err := s.removeIntercessor(ctx, mem); err != nil {
			return err
		}
	}
	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgRemoveUser)
}

func (s *MemberService) removeIntercessor(ctx context.Context, mem domain.Member) error {
	phones, err := s.intercessors.Get(ctx)
	if err != nil {
		return err
	}
	phones.RemovePhone(mem.Phone)
	if err = s.intercessors.Save(ctx, phones); err != nil {
		return err
	}
	return s.moveActivePrayer(ctx, mem)
}

func (s *MemberService) moveActivePrayer(ctx context.Context, mem domain.Member) error {
	isActive, err := s.prayers.Exists(ctx, mem.Phone)
	if err != nil {
		return err
	}
	if !isActive {
		return nil
	}

	pryr, err := s.prayers.Get(ctx, mem.Phone, false)
	if err != nil {
		return err
	}

	if err = s.prayers.Delete(ctx, mem.Phone, false); err != nil {
		return err
	}

	id, err := generateID()
	if err != nil {
		return err
	}
	pryr.IntercessorPhone = id
	pryr.Intercessor = domain.Member{}

	return s.prayers.Save(ctx, pryr, true)
}

func (s *MemberService) SignUp(ctx context.Context, msg domain.TextMessage, mem domain.Member) error {
	cleanMsg := cleanStr(msg.Body)

	switch {
	case cleanMsg == "pray":
		return s.signUpStageOne(ctx, mem)
	case mem.SetupStage == domain.MemberSignUpStepOne:
		return s.signUpStageTwo(ctx, msg, mem)
	case cleanMsg == "1" && mem.SetupStage == domain.MemberSignUpStepTwo:
		return s.signUpFinalPrayer(ctx, mem)
	case cleanMsg == "2" && mem.SetupStage == domain.MemberSignUpStepTwo:
		return s.signUpStageThree(ctx, mem)
	case mem.SetupStage == domain.MemberSignUpStepThree:
		return s.signUpFinalIntercessor(ctx, msg, mem)
	default:
		return s.signUpWrongInput(ctx, mem, msg)
	}
}

func (s *MemberService) signUpStageOne(ctx context.Context, mem domain.Member) error {
	mem.SetupStatus = domain.MemberSetupInProgress
	mem.SetupStage = domain.MemberSignUpStepOne
	if err := s.members.Save(ctx, &mem); err != nil {
		return err
	}
	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgNameRequest)
}

func (s *MemberService) signUpStageTwo(ctx context.Context, msg domain.TextMessage, mem domain.Member) error {
	profanity := messaging.CheckProfanity(msg.Body)
	if profanity != "" {
		rendered, err := messaging.Render(messaging.ProfanityDetectedTmpl, struct{ Word string }{profanity})
		if err != nil {
			return err
		}
		return s.sender.SendMessage(ctx, mem.Phone, rendered)
	}

	if cleanStr(msg.Body) == "2" {
		mem.Name = "Anonymous"
	} else {
		mem.Name = msg.Body
	}

	if !isNameValid(mem.Name) {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgInvalidName)
	}

	mem.SetupStage = domain.MemberSignUpStepTwo
	if err := s.members.Save(ctx, &mem); err != nil {
		return err
	}
	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgMemberTypeRequest)
}

func (s *MemberService) signUpFinalPrayer(ctx context.Context, mem domain.Member) error {
	mem.SetupStatus = domain.MemberSetupComplete
	mem.SetupStage = domain.MemberSignUpStepFinal
	mem.Intercessor = false
	if err := s.members.Save(ctx, &mem); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgSignUpConfirmation
	return s.sender.SendMessage(ctx, mem.Phone, body)
}

func (s *MemberService) signUpStageThree(ctx context.Context, mem domain.Member) error {
	mem.SetupStage = domain.MemberSignUpStepThree
	mem.Intercessor = true
	if err := s.members.Save(ctx, &mem); err != nil {
		return err
	}
	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgPrayerNumRequest)
}

func (s *MemberService) signUpFinalIntercessor(ctx context.Context, msg domain.TextMessage, mem domain.Member) error {
	num, err := strconv.Atoi(cleanStr(msg.Body))
	if err != nil {
		return s.signUpWrongInput(ctx, mem, msg)
	}

	phones, err := s.intercessors.Get(ctx)
	if err != nil {
		return err
	}

	phones.AddPhone(mem.Phone)
	if err = s.intercessors.Save(ctx, phones); err != nil {
		return err
	}

	mem.SetupStatus = domain.MemberSetupComplete
	mem.SetupStage = domain.MemberSignUpStepFinal
	mem.WeeklyPrayerLimit = num
	mem.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
	if err = s.members.Save(ctx, &mem); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgIntercessorInstructions + "\n\n" +
		messaging.MsgSignUpConfirmation
	return s.sender.SendMessage(ctx, mem.Phone, body)
}

func (s *MemberService) signUpWrongInput(ctx context.Context, mem domain.Member, msg domain.TextMessage) error {
	slog.WarnContext(ctx, "wrong input received during sign up", "member", mem.Phone, "msg", msg)
	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgWrongInput)
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

func isNameValid(name string) bool {
	letterCount := 0
	minLetters := 2

	for _, ch := range name {
		switch {
		case unicode.IsLetter(ch):
			letterCount++
		case ch == ' ':
			// Spaces are fine but don't count.
		default:
			return false
		}
	}

	return letterCount >= minLetters
}
