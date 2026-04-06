package prayertexter

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/utility"
)

func signUp(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, msg messaging.TextMessage, mem object.Member, profanityChecker *messaging.ProfanityChecker) error {
	cleanMsg := cleanStr(msg.Body)

	switch {
	case cleanMsg == "pray":
		if err := signUpStageOne(ctx, ddbClnt, smsClnt, mem); err != nil {
			return utility.WrapError(err, "failed sign up stage 1")
		}
	case mem.SetupStage == object.MemberSignUpStepOne:
		if err := signUpStageTwo(ctx, ddbClnt, smsClnt, mem, msg, profanityChecker); err != nil {
			return utility.WrapError(err, "failed sign up stage 2")
		}
	case cleanMsg == "1" && mem.SetupStage == object.MemberSignUpStepTwo:
		if err := signUpFinalPrayerMessage(ctx, ddbClnt, smsClnt, mem); err != nil {
			return utility.WrapError(err, "failed sign up final prayer message")
		}
	case cleanMsg == "2" && mem.SetupStage == object.MemberSignUpStepTwo:
		if err := signUpStageThree(ctx, ddbClnt, smsClnt, mem); err != nil {
			return utility.WrapError(err, "failed sign up stage 3")
		}
	case mem.SetupStage == object.MemberSignUpStepThree:
		if err := signUpFinalIntercessorMessage(ctx, ddbClnt, smsClnt, mem, msg); err != nil {
			return utility.WrapError(err, "failed sign up final intercessor message")
		}
	default:
		if err := signUpWrongInput(ctx, smsClnt, mem, msg); err != nil {
			return utility.WrapError(err, "failed sign up wrong input")
		}
	}

	return nil
}

func signUpStageOne(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member) error {
	mem.SetupStatus = object.MemberSetupInProgress
	mem.SetupStage = object.MemberSignUpStepOne
	if err := mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgNameRequest)
}

func signUpStageTwo(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member, msg messaging.TextMessage, profanityChecker *messaging.ProfanityChecker) error {
	hasProfanity, err := checkIfProfanity(ctx, smsClnt, mem, msg, profanityChecker)
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

	isValid, err := checkIfNameValid(ctx, smsClnt, mem)
	if err != nil {
		return err
	} else if !isValid {
		return nil
	}

	mem.SetupStage = object.MemberSignUpStepTwo

	if err = mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgMemberTypeRequest)
}

func signUpFinalPrayerMessage(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member) error {
	mem.SetupStatus = object.MemberSetupComplete
	mem.SetupStage = object.MemberSignUpStepFinal
	mem.Intercessor = false
	if err := mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgSignUpConfirmation

	return mem.SendMessage(ctx, smsClnt, body)
}

func signUpStageThree(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member) error {
	mem.SetupStage = object.MemberSignUpStepThree
	mem.Intercessor = true
	if err := mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgPrayerNumRequest)
}

func signUpFinalIntercessorMessage(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member, msg messaging.TextMessage) error {
	num, err := strconv.Atoi(cleanStr(msg.Body))
	if err != nil {
		return signUpWrongInput(ctx, smsClnt, mem, msg)
	}

	phones := object.IntercessorPhones{}
	if err = phones.Get(ctx, ddbClnt); err != nil {
		return err
	}

	phones.AddPhone(mem.Phone)
	if err = phones.Put(ctx, ddbClnt); err != nil {
		return err
	}

	mem.SetupStatus = object.MemberSetupComplete
	mem.SetupStage = object.MemberSignUpStepFinal
	mem.WeeklyPrayerLimit = num
	mem.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
	if err = mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgIntercessorInstructions + "\n\n" +
		messaging.MsgSignUpConfirmation

	return mem.SendMessage(ctx, smsClnt, body)
}

func signUpWrongInput(ctx context.Context, smsClnt messaging.TextSender, mem object.Member, msg messaging.TextMessage) error {
	slog.WarnContext(ctx, "wrong input received during sign up", "member", mem.Phone, "msg", msg)

	return mem.SendMessage(ctx, smsClnt, messaging.MsgWrongInput)
}
