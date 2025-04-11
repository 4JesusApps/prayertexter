/*
Package prayertexter is the main package for the prayertexter application. This package contains all of the main
application logic such as the sign up process, prayer request process, and prayer confirmation process. This package
is the starting point for all received text messages and decides what to do with the message based on message content
and sender member status.
*/
package prayertexter

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

const (
	preStage            = "PRE"
	helpStage           = "HELP"
	memberDeleteStage   = "MEMBER DELETE"
	signUpStage         = "SIGN UP"
	dropMessageStage    = "DROP MESSAGE"
	completePrayerStage = "COMPLETE PRAYER"
	prayerRequestStage  = "PRAYER REQUEST"
	postStage           = "POST"
	stageErrPrefix      = "failure during stage "
)

// MainFlow is the start of the prayertexter application. It receives a text message as a parameter and based on the
// message content and sender phone number, it decides what operations to perform.
func MainFlow(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender,
	msg messaging.TextMessage) error {
	config.InitConfig()

	currTime := time.Now().Format(time.RFC3339)
	id, err := utility.GenerateID()
	if err != nil {
		return utility.LogAndWrapError(err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	state := object.State{}
	state.Status, state.TimeStart, state.ID, state.Message = object.StateInProgress, currTime, id, msg
	if err := state.Update(ctx, ddbClnt, false); err != nil {
		return utility.LogAndWrapError(err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	mem := object.Member{Phone: msg.Phone}
	if err := mem.Get(ctx, ddbClnt); err != nil {
		return utility.LogAndWrapError(err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	var stageErr error

	switch {
	// HELP STAGE
	// Responds with contact info and is a requirement for the 10DLC phone number provider to get sent to to anyone
	// regardless whether they are a member or not.
	case strings.ToLower(msg.Body) == "help":
		stageErr = executeStage(ctx, ddbClnt, helpStage, &state, func() error {
			return mem.SendMessage(ctx, smsClnt, messaging.MsgHelp)
		})

	// MEMBER DELETE STAGE
	// Removes member from prayertexter.
	case strings.ToLower(msg.Body) == "cancel" || strings.ToLower(msg.Body) == "stop":
		stageErr = executeStage(ctx, ddbClnt, memberDeleteStage, &state, func() error {
			return memberDelete(ctx, ddbClnt, smsClnt, mem)
		})

	// SIGN UP STAGE
	// Initial member sign up process.
	case strings.ToLower(msg.Body) == "pray" || mem.SetupStatus == object.MemberSetupInProgress:
		stageErr = executeStage(ctx, ddbClnt, signUpStage, &state, func() error {
			return signUp(ctx, ddbClnt, smsClnt, msg, mem)
		})

	// DROP MESSAGE STAGE
	// Drops all messages if they do not meet any of the previous criteria. This serves as a catch all to drop any
	// messages of non members (other than help and sign up messages).
	case mem.SetupStatus == "":
		slog.Warn("non registered user dropping message", "phone", mem.Phone, "msg", msg.Body)
		stageErr = executeStage(ctx, ddbClnt, dropMessageStage, &state, func() error {
			return nil
		})

	// COMPLETE PRAYER STAGE
	// Intercessors confirm that they prayed for a prayer, a confirmations is sent out to the prayer requestor, and the
	// prayer is marked as completed.
	case strings.ToLower(msg.Body) == "prayed":
		stageErr = executeStage(ctx, ddbClnt, completePrayerStage, &state, func() error {
			return completePrayer(ctx, ddbClnt, smsClnt, mem)
		})

	// PRAYER REQUEST STAGE
	// Assigns a prayer request to intercessors.
	case mem.SetupStatus == object.MemberSetupComplete:
		stageErr = executeStage(ctx, ddbClnt, prayerRequestStage, &state, func() error {
			return prayerRequest(ctx, ddbClnt, smsClnt, msg, mem)
		})

	// This should never happen and if it does then it is a bug.
	default:
		err := errors.New("unexpected text message input/member status")
		return utility.LogAndWrapError(err, "could not satisfy any required conditions", "phone", mem.Phone, "msg",
			msg.Body)
	}

	if stageErr != nil {
		return stageErr
	}

	if err := state.Update(ctx, ddbClnt, true); err != nil {
		return utility.LogAndWrapError(err, stageErrPrefix+postStage, "phone", mem.Phone, "msg", msg.Body)
	}

	return nil
}

func executeStage(ctx context.Context, ddbClnt db.DDBConnecter, stageName string, state *object.State,
	stageFunc func() error) error {
	state.Stage = stageName
	if err := state.Update(ctx, ddbClnt, false); err != nil {
		return utility.LogAndWrapError(err, stageErrPrefix+stageName, "phone", state.Message.Phone, "msg",
			state.Message.Body)
	}

	if stageErr := stageFunc(); stageErr != nil {
		state.Error = stageErr.Error()
		state.Status = object.StateFailed
		if updateErr := state.Update(ctx, ddbClnt, false); updateErr != nil {
			return utility.LogAndWrapError(updateErr, stageErrPrefix+stageName, "stage error", stageErr.Error(),
				"phone", state.Message.Phone, "msg", state.Message.Body)
		}

		return utility.LogAndWrapError(stageErr, stageErrPrefix+stageName, "phone", state.Message.Phone,
			"msg", state.Message.Body)
	}

	return nil
}

func signUp(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, msg messaging.TextMessage,
	mem object.Member) error {
	switch {
	case strings.ToLower(msg.Body) == "pray":
		if err := signUpStageOne(ctx, ddbClnt, smsClnt, mem); err != nil {
			return utility.WrapError(err, "failed sign up stage 1")
		}
	case msg.Body != "2" && mem.SetupStage == object.MemberSignUpStepOne:
		if err := signUpStageTwo(ctx, ddbClnt, smsClnt, mem, msg, false); err != nil {
			return utility.WrapError(err, "failed sign up stage 2")
		}
	case msg.Body == "2" && mem.SetupStage == object.MemberSignUpStepOne:
		if err := signUpStageTwo(ctx, ddbClnt, smsClnt, mem, msg, true); err != nil {
			return utility.WrapError(err, "failed sign up stage 2")
		}
	case msg.Body == "1" && mem.SetupStage == object.MemberSignUpStepTwo:
		if err := signUpFinalPrayerMessage(ctx, ddbClnt, smsClnt, mem); err != nil {
			return utility.WrapError(err, "failed sign up final prayer message")
		}
	case msg.Body == "2" && mem.SetupStage == object.MemberSignUpStepTwo:
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

func signUpStageOne(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender,
	mem object.Member) error {
	mem.SetupStatus = object.MemberSetupInProgress
	mem.SetupStage = object.MemberSignUpStepOne
	if err := mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgNameRequest)
}

func signUpStageTwo(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member,
	msg messaging.TextMessage, isAnon bool) error {
	mem.SetupStage = object.MemberSignUpStepTwo
	if isAnon {
		mem.Name = "Anonymous"
	} else {
		mem.Name = msg.Body
	}

	if err := mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgMemberTypeRequest)
}

func signUpFinalPrayerMessage(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender,
	mem object.Member) error {
	mem.SetupStatus = object.MemberSetupComplete
	mem.SetupStage = object.MemberSignUpStepFinal
	mem.Intercessor = false
	if err := mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgSignUpConfirmation

	return mem.SendMessage(ctx, smsClnt, body)
}

func signUpStageThree(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender,
	mem object.Member) error {
	mem.SetupStage = object.MemberSignUpStepThree
	mem.Intercessor = true
	if err := mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgPrayerNumRequest)
}

func signUpFinalIntercessorMessage(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender,
	mem object.Member, msg messaging.TextMessage) error {
	num, err := strconv.Atoi(msg.Body)
	if err != nil {
		return signUpWrongInput(ctx, smsClnt, mem, msg)
	}

	phones := object.IntercessorPhones{}
	if err := phones.Get(ctx, ddbClnt); err != nil {
		return err
	}

	phones.AddPhone(mem.Phone)
	if err := phones.Put(ctx, ddbClnt); err != nil {
		return err
	}

	mem.SetupStatus = object.MemberSetupComplete
	mem.SetupStage = object.MemberSignUpStepFinal
	mem.WeeklyPrayerLimit = num
	mem.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
	if err := mem.Put(ctx, ddbClnt); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgIntercessorInstructions + "\n\n" +
		messaging.MsgSignUpConfirmation

	return mem.SendMessage(ctx, smsClnt, body)
}

func signUpWrongInput(ctx context.Context, smsClnt messaging.TextSender, mem object.Member,
	msg messaging.TextMessage) error {
	slog.Warn("wrong input received during sign up", "member", mem.Phone, "msg", msg)

	return mem.SendMessage(ctx, smsClnt, messaging.MsgWrongInput)
}

func memberDelete(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, mem object.Member) error {
	if err := mem.Delete(ctx, ddbClnt); err != nil {
		return err
	}
	if mem.Intercessor {
		if err := removeIntercessor(ctx, ddbClnt, mem); err != nil {
			return err
		}
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgRemoveUser)
}

func removeIntercessor(ctx context.Context, ddbClnt db.DDBConnecter, mem object.Member) error {
	phones := object.IntercessorPhones{}
	if err := phones.Get(ctx, ddbClnt); err != nil {
		return err
	}
	phones.RemovePhone(mem.Phone)
	if err := phones.Put(ctx, ddbClnt); err != nil {
		return err
	}

	// Moves an active prayer from the intercessor being removed to the Prayer queue. This is done to ensure that the
	// prayer eventually gets assigned to another intercessor.
	return moveActivePrayer(ctx, ddbClnt, mem)
}

func moveActivePrayer(ctx context.Context, ddbClnt db.DDBConnecter, mem object.Member) error {
	isActive, err := object.IsPrayerActive(ctx, ddbClnt, mem.Phone)
	if err != nil {
		return err
	}

	if isActive {
		pryr := object.Prayer{IntercessorPhone: mem.Phone}
		if err := pryr.Get(ctx, ddbClnt, false); err != nil {
			return err
		}

		if err := pryr.Delete(ctx, ddbClnt, false); err != nil {
			return err
		}

		// A random ID is generated since queued Prayers do not have an intercessor assigned to them. We use the ID in
		// place if the intercessors phone number until there is an available intercessor, at which time the ID will get
		// changed to the available intercessors phone number.
		id, err := utility.GenerateID()
		if err != nil {
			return err
		}
		pryr.IntercessorPhone, pryr.Intercessor = id, object.Member{}

		return pryr.Put(ctx, ddbClnt, true)
	}

	return nil
}

func prayerRequest(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender,
	msg messaging.TextMessage, mem object.Member) error {
	profanity := msg.CheckProfanity()
	if profanity != "" {
		msg := strings.Replace(messaging.MsgProfanityFound, "PLACEHOLDER", profanity, 1)
		if err := mem.SendMessage(ctx, smsClnt, msg); err != nil {
			return err
		}
		return nil
	}

	intercessors, err := FindIntercessors(ctx, ddbClnt, mem.Phone)
	if err != nil && errors.Is(err, utility.ErrNoAvailableIntercessors) {
		if err := queuePrayer(ctx, ddbClnt, smsClnt, msg, mem); err != nil {
			return utility.WrapError(err, "failed to queue prayer")
		}
		return nil
	} else if err != nil {
		return utility.WrapError(err, "failed to find intercessor")
	}

	for _, intr := range intercessors {
		pryr := object.Prayer{
			Intercessor:      intr,
			IntercessorPhone: intr.Phone,
			Request:          msg.Body,
			Requestor:        mem,
		}
		if err := pryr.Put(ctx, ddbClnt, false); err != nil {
			return err
		}

		msg := strings.Replace(messaging.MsgPrayerIntro, "PLACEHOLDER", mem.Name, 1)
		if err := intr.SendMessage(ctx, smsClnt, msg+pryr.Request); err != nil {
			return err
		}
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgPrayerSentOut)
}

// FindIntercessors returns a slice of Member intercessors that are available to be assigned a prayer request. If there
// are no available intercessors, it will return an error.
func FindIntercessors(ctx context.Context, ddbClnt db.DDBConnecter, skipPhone string) ([]object.Member, error) {
	allPhones, err := getAndPreparePhones(ctx, ddbClnt, skipPhone)
	if err != nil {
		return nil, err
	}

	var intercessors []object.Member
	intercessorsPerPrayer := viper.GetInt(object.IntercessorsPerPrayerConfigPath)

	for len(intercessors) < intercessorsPerPrayer {
		randPhones := allPhones.GenRandPhones()
		if randPhones == nil {
			// There are no more intercessors to process/check from IntercessorPhones
			if len(intercessors) > 0 {
				// There is at least one intercessor that has been found. This will be returned even though it is less
				// than desired because returning less is better than returning none.
				return intercessors, nil
			}

			// There are not any intercessors available at all.
			return nil, utility.ErrNoAvailableIntercessors
		}

		for _, phn := range randPhones {
			intr, err := processIntercessor(ctx, ddbClnt, phn)
			if err != nil && errors.Is(err, utility.ErrIntercessorUnavailable) {
				allPhones.RemovePhone(phn)
				continue
			} else if err != nil {
				return nil, err
			}

			intercessors = append(intercessors, *intr)
			allPhones.RemovePhone(phn)
		}
	}

	return intercessors, nil
}

func getAndPreparePhones(ctx context.Context, ddbClnt db.DDBConnecter, skipPhone string) (object.IntercessorPhones,
	error) {
	phones := object.IntercessorPhones{}
	if err := phones.Get(ctx, ddbClnt); err != nil {
		return phones, err
	}

	// Removes the prayer requestors phone number from IntercessorPhones so that they do not get assigned to pray for
	// their own prayer request. This could happen if the prayer requestor is also an intercessor.
	utility.RemoveItem(&phones.Phones, skipPhone)

	return phones, nil
}

func processIntercessor(ctx context.Context, ddbClnt db.DDBConnecter, phone string) (*object.Member, error) {
	intr := object.Member{Phone: phone}
	if err := intr.Get(ctx, ddbClnt); err != nil {
		return nil, err
	}

	isActive, err := object.IsPrayerActive(ctx, ddbClnt, intr.Phone)
	if err != nil {
		return nil, err
	}
	if isActive {
		// This intercessor already has 1 active prayer and is therefor unavailable. Each intercessor can only have a
		// maximum of 1 active prayer request at any given time.
		return nil, utility.ErrIntercessorUnavailable
	}

	if intr.PrayerCount < intr.WeeklyPrayerLimit {
		intr.PrayerCount++
	} else {
		if canResetPrayerCount(intr) {
			// Reset intercessor's weekly prayer count
			intr.PrayerCount = 1
			intr.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
		} else {
			return nil, utility.ErrIntercessorUnavailable
		}
	}

	if err := intr.Put(ctx, ddbClnt); err != nil {
		return nil, err
	}
	return &intr, nil
}

func canResetPrayerCount(intr object.Member) bool {
	weekDays := 7
	dayHours := 24

	currentTime := time.Now()
	previousTime, err := time.Parse(time.RFC3339, intr.WeeklyPrayerDate)
	if err != nil {
		return false
	}
	diffDays := currentTime.Sub(previousTime).Hours() / float64(dayHours)
	return diffDays > float64(weekDays)
}

func queuePrayer(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, msg messaging.TextMessage,
	mem object.Member) error {
	pryr := object.Prayer{}
	// A random ID is generated since queued Prayers do not have an intercessor assigned to them. We use the ID in place
	// if the intercessors phone number until there is an available intercessor, at which time the ID will get changed
	// to the available intercessors phone number.
	id, err := utility.GenerateID()
	if err != nil {
		return err
	}

	pryr.IntercessorPhone, pryr.Request, pryr.Requestor = id, msg.Body, mem

	if err := pryr.Put(ctx, ddbClnt, true); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgPrayerQueued)
}

func completePrayer(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender,
	mem object.Member) error {
	pryr := object.Prayer{IntercessorPhone: mem.Phone}
	if err := pryr.Get(ctx, ddbClnt, false); err != nil {
		return err
	}

	if pryr.Request == "" {
		// Get Dynamodb calls return empty data if the key does not exist in the table. Therefor if prayer request is
		// empty here, it means that it did not exist in the database.
		if err := mem.SendMessage(ctx, smsClnt, messaging.MsgNoActivePrayer); err != nil {
			return err
		}
		return nil
	}

	if err := mem.SendMessage(ctx, smsClnt, messaging.MsgPrayerThankYou); err != nil {
		return err
	}

	msg := strings.Replace(messaging.MsgPrayerConfirmation, "PLACEHOLDER", mem.Name, 1)

	isActive, err := object.IsMemberActive(ctx, ddbClnt, pryr.Requestor.Phone)
	if err != nil {
		return err
	}

	if isActive {
		if err := pryr.Requestor.SendMessage(ctx, smsClnt, msg); err != nil {
			return err
		}
	} else {
		slog.Warn("Skip sending message, member is not active", "recipient", pryr.Requestor.Phone, "body", msg)
	}

	return pryr.Delete(ctx, ddbClnt, false)
}
