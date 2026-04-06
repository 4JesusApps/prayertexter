package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/4JesusApps/prayertexter/internal/apperrors"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/model"
)

func (s *Service) signUp(ctx context.Context, msg messaging.TextMessage, mem *model.Member) error {
	cleanMsg := cleanStr(msg.Body)

	switch {
	case cleanMsg == "pray":
		if err := s.signUpStageOne(ctx, mem); err != nil {
			return apperrors.WrapError(err, "failed sign up stage 1")
		}
	case mem.SetupStage == model.SignUpStepOne:
		if err := s.signUpStageTwo(ctx, mem, msg); err != nil {
			return apperrors.WrapError(err, "failed sign up stage 2")
		}
	case cleanMsg == "1" && mem.SetupStage == model.SignUpStepTwo:
		if err := s.signUpFinalPrayerMessage(ctx, mem); err != nil {
			return apperrors.WrapError(err, "failed sign up final prayer message")
		}
	case cleanMsg == "2" && mem.SetupStage == model.SignUpStepTwo:
		if err := s.signUpStageThree(ctx, mem); err != nil {
			return apperrors.WrapError(err, "failed sign up stage 3")
		}
	case mem.SetupStage == model.SignUpStepThree:
		if err := s.signUpFinalIntercessorMessage(ctx, mem, msg); err != nil {
			return apperrors.WrapError(err, "failed sign up final intercessor message")
		}
	default:
		if err := s.signUpWrongInput(ctx, mem, msg); err != nil {
			return apperrors.WrapError(err, "failed sign up wrong input")
		}
	}

	return nil
}

func (s *Service) signUpStageOne(ctx context.Context, mem *model.Member) error {
	mem.SetupStatus = model.SetupInProgress
	mem.SetupStage = model.SignUpStepOne
	if err := s.putMember(ctx, mem); err != nil {
		return err
	}

	return s.sendMessage(ctx, mem.Phone, messaging.MsgNameRequest)
}

func (s *Service) signUpStageTwo(ctx context.Context, mem *model.Member, msg messaging.TextMessage) error {
	hasProfanity, err := s.checkIfProfanity(ctx, mem, msg)
	if err != nil {
		return err
	} else if hasProfanity {
		return nil
	}

	if cleanStr(msg.Body) == "2" {
		mem.Name = "Anonymous"
	} else {
		mem.Name = msg.Body
	}

	isValid, err := s.checkIfNameValid(ctx, mem)
	if err != nil {
		return err
	} else if !isValid {
		return nil
	}

	mem.SetupStage = model.SignUpStepTwo

	if err = s.putMember(ctx, mem); err != nil {
		return err
	}

	return s.sendMessage(ctx, mem.Phone, messaging.MsgMemberTypeRequest)
}

func (s *Service) signUpFinalPrayerMessage(ctx context.Context, mem *model.Member) error {
	mem.SetupStatus = model.SetupComplete
	mem.SetupStage = model.SignUpStepFinal
	mem.Intercessor = false
	if err := s.putMember(ctx, mem); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgSignUpConfirmation

	return s.sendMessage(ctx, mem.Phone, body)
}

func (s *Service) signUpStageThree(ctx context.Context, mem *model.Member) error {
	mem.SetupStage = model.SignUpStepThree
	mem.Intercessor = true
	if err := s.putMember(ctx, mem); err != nil {
		return err
	}

	return s.sendMessage(ctx, mem.Phone, messaging.MsgPrayerNumRequest)
}

func (s *Service) signUpFinalIntercessorMessage(ctx context.Context, mem *model.Member, msg messaging.TextMessage) error {
	num, err := strconv.Atoi(cleanStr(msg.Body))
	if err != nil {
		return s.signUpWrongInput(ctx, mem, msg)
	}

	phones, err := s.getIntercessorPhones(ctx)
	if err != nil {
		return err
	}

	phones.AddPhone(mem.Phone)
	if err = s.putIntercessorPhones(ctx, phones); err != nil {
		return err
	}

	mem.SetupStatus = model.SetupComplete
	mem.SetupStage = model.SignUpStepFinal
	mem.WeeklyPrayerLimit = num
	mem.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
	if err = s.putMember(ctx, mem); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgIntercessorInstructions + "\n\n" +
		messaging.MsgSignUpConfirmation

	return s.sendMessage(ctx, mem.Phone, body)
}

func (s *Service) signUpWrongInput(ctx context.Context, mem *model.Member, msg messaging.TextMessage) error {
	slog.WarnContext(ctx, "wrong input received during sign up", "member", mem.Phone, "msg", msg)

	return s.sendMessage(ctx, mem.Phone, messaging.MsgWrongInput)
}
