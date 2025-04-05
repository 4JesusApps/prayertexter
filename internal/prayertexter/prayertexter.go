package prayertexter

import (
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

	stageErrPrefix = "failure during stage "
)

func MainFlow(msg messaging.TextMessage, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	config.InitConfig()

	currTime := time.Now().Format(time.RFC3339)
	id, err := utility.GenerateID()
	if err != nil {
		return utility.LogAndWrapError(err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	state := object.State{}
	state.Status, state.TimeStart, state.ID, state.Message = object.StateInProgress, currTime, id, msg
	if err := state.Update(ddbClnt, false); err != nil {
		return utility.LogAndWrapError(err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	mem := object.Member{Phone: msg.Phone}
	if err := mem.Get(ddbClnt); err != nil {
		return utility.LogAndWrapError(err, stageErrPrefix+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	var stageErr error

	switch {
	// HELP STAGE
	// Responds with contact info and is a requirement to get sent to to anyone regardless whether they are a member or
	// not.
	case strings.ToLower(msg.Body) == "help":
		stageErr = executeStage(ddbClnt, helpStage, &state, func() error {
			return mem.SendMessage(smsClnt, messaging.MsgHelp)
		})

	// MEMBER DELETE STAGE
	// Removes member from prayertexter.
	case strings.ToLower(msg.Body) == "cancel" || strings.ToLower(msg.Body) == "stop":
		stageErr = executeStage(ddbClnt, memberDeleteStage, &state, func() error {
			return memberDelete(mem, ddbClnt, smsClnt)
		})

	// SIGN UP STAGE
	// Initial member sign up process.
	case strings.ToLower(msg.Body) == "pray" || mem.SetupStatus == object.MemberSetupInProgress:
		stageErr = executeStage(ddbClnt, signUpStage, &state, func() error {
			return signUp(msg, mem, ddbClnt, smsClnt)
		})

	// DROP MESSAGE STAGE
	// Drops all messages if they do not meet any of the previous criteria. This serves as a catch all to drop any
	// messages of non members.
	case mem.SetupStatus == "":
		slog.Warn("non registered user dropping message", "phone", mem.Phone, "msg", msg.Body)
		stageErr = executeStage(ddbClnt, dropMessageStage, &state, func() error {
			return nil
		})

	// COMPLETE PRAYER STAGE
	// Intercessors confirm that they prayed for a prayer, a confirmations is sent out to the prayer requestor, and the
	// prayer is marked as completed.
	case strings.ToLower(msg.Body) == "prayed":
		stageErr = executeStage(ddbClnt, completePrayerStage, &state, func() error {
			return completePrayer(mem, ddbClnt, smsClnt)
		})

	// PRAYER REQUEST STAGE
	// Assigns a prayer request to intercessors.
	case mem.SetupStatus == object.MemberSetupComplete:
		stageErr = executeStage(ddbClnt, prayerRequestStage, &state, func() error {
			return prayerRequest(msg, mem, ddbClnt, smsClnt)
		})

	// This should never happen and if it does then it is a bug.
	default:
		err := errors.New("unexpected text message input/member status")
		return utility.LogAndWrapError(err, "could not satisfy any required conditions", "phone", mem.Phone, "msg", msg.Body)
	}

	if stageErr != nil {
		return stageErr
	}

	if err := state.Update(ddbClnt, true); err != nil {
		return utility.LogAndWrapError(err, stageErrPrefix+postStage, "phone", mem.Phone, "msg", msg.Body)
	}

	return nil
}

func executeStage(ddbClnt db.DDBConnecter, stageName string, state *object.State, stageFunc func() error) error {
	state.Stage = stageName
	if err := state.Update(ddbClnt, false); err != nil {
		return utility.LogAndWrapError(err, stageErrPrefix+stageName, "phone", state.Message.Phone, "msg", state.Message.Body)
	}

	if stageErr := stageFunc(); stageErr != nil {
		state.Error = stageErr.Error()
		state.Status = object.StateFailed
		if updateErr := state.Update(ddbClnt, false); updateErr != nil {
			return utility.LogAndWrapError(updateErr, stageErrPrefix+stageName, "stage error", stageErr.Error(),
				"phone", state.Message.Phone, "msg", state.Message.Body)
		}

		return utility.LogAndWrapError(stageErr, stageErrPrefix+stageName, "phone", state.Message.Phone,
			"msg", state.Message.Body)
	}

	return nil
}

func signUp(msg messaging.TextMessage, mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	switch {
	case strings.ToLower(msg.Body) == "pray":
		if err := signUpStageOne(mem, ddbClnt, smsClnt); err != nil {
			return utility.WrapError(err, "failed sign up stage 1")
		}
	case msg.Body != "2" && mem.SetupStage == object.MemberSignUpStepOne:
		if err := signUpStageTwoA(mem, ddbClnt, smsClnt, msg); err != nil {
			return utility.WrapError(err, "failed sign up stage 2A")
		}
	case msg.Body == "2" && mem.SetupStage == object.MemberSignUpStepOne:
		if err := signUpStageTwoB(mem, ddbClnt, smsClnt); err != nil {
			return utility.WrapError(err, "failed sign up stage 2B")
		}
	case msg.Body == "1" && mem.SetupStage == object.MemberSignUpStepTwo:
		if err := signUpFinalPrayerMessage(mem, ddbClnt, smsClnt); err != nil {
			return utility.WrapError(err, "failed sign up final prayer message")
		}
	case msg.Body == "2" && mem.SetupStage == object.MemberSignUpStepTwo:
		if err := signUpStageThree(mem, ddbClnt, smsClnt); err != nil {
			return utility.WrapError(err, "failed sign up stage 3")
		}
	case mem.SetupStage == object.MemberSignUpStepThree:
		if err := signUpFinalIntercessorMessage(mem, ddbClnt, smsClnt, msg); err != nil {
			return utility.WrapError(err, "failed sign up final intercessor message")
		}
	default:
		if err := signUpWrongInput(mem, msg, smsClnt); err != nil {
			return utility.WrapError(err, "failed sign up wrong input")
		}
	}

	return nil
}

func signUpStageOne(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	mem.SetupStatus = object.MemberSetupInProgress
	mem.SetupStage = object.MemberSignUpStepOne
	if err := mem.Put(ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(smsClnt, messaging.MsgNameRequest)
}

func signUpStageTwoA(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender,
	msg messaging.TextMessage) error {
	mem.SetupStage = object.MemberSignUpStepTwo
	mem.Name = msg.Body
	if err := mem.Put(ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(smsClnt, messaging.MsgMemberTypeRequest)
}

func signUpStageTwoB(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	mem.SetupStage = object.MemberSignUpStepTwo
	mem.Name = "Anonymous"
	if err := mem.Put(ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(smsClnt, messaging.MsgMemberTypeRequest)
}

func signUpFinalPrayerMessage(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	mem.SetupStatus = object.MemberSetupComplete
	mem.SetupStage = object.MemberSignUpStepFinal
	mem.Intercessor = false
	if err := mem.Put(ddbClnt); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgSignUpConfirmation

	return mem.SendMessage(smsClnt, body)
}

func signUpStageThree(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	mem.SetupStage = object.MemberSignUpStepThree
	mem.Intercessor = true
	if err := mem.Put(ddbClnt); err != nil {
		return err
	}

	return mem.SendMessage(smsClnt, messaging.MsgPrayerNumRequest)
}

func signUpFinalIntercessorMessage(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender,
	msg messaging.TextMessage) error {
	num, err := strconv.Atoi(msg.Body)
	if err != nil {
		return signUpWrongInput(mem, msg, smsClnt)
	}

	phones := object.IntercessorPhones{}
	if err := phones.Get(ddbClnt); err != nil {
		return err
	}

	phones.AddPhone(mem.Phone)
	if err := phones.Put(ddbClnt); err != nil {
		return err
	}

	mem.SetupStatus = object.MemberSetupComplete
	mem.SetupStage = object.MemberSignUpStepFinal
	mem.WeeklyPrayerLimit = num
	mem.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
	if err := mem.Put(ddbClnt); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgIntercessorInstructions + "\n\n" +
		messaging.MsgSignUpConfirmation

	return mem.SendMessage(smsClnt, body)
}

func signUpWrongInput(mem object.Member, msg messaging.TextMessage, smsClnt messaging.TextSender) error {
	slog.Warn("wrong input received during sign up", "member", mem.Phone, "msg", msg)

	return mem.SendMessage(smsClnt, messaging.MsgWrongInput)
}

func memberDelete(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	if err := mem.Delete(ddbClnt); err != nil {
		return err
	}
	if mem.Intercessor {
		if err := removeIntercessor(mem, ddbClnt); err != nil {
			return err
		}
	}

	return mem.SendMessage(smsClnt, messaging.MsgRemoveUser)
}

func removeIntercessor(mem object.Member, ddbClnt db.DDBConnecter) error {
	phones := object.IntercessorPhones{}
	if err := phones.Get(ddbClnt); err != nil {
		return err
	}
	phones.RemovePhone(mem.Phone)
	if err := phones.Put(ddbClnt); err != nil {
		return err
	}

	// Moves an active prayer from the intercessor being removed to the Prayer queue. This is done to ensure that the
	// prayer eventually gets assigned to another intercessor.
	return moveActivePrayer(mem, ddbClnt)
}

func moveActivePrayer(mem object.Member, ddbClnt db.DDBConnecter) error {
	isActive, err := object.IsPrayerActive(ddbClnt, mem.Phone)
	if err != nil {
		return err
	}

	if isActive {
		pryr := object.Prayer{IntercessorPhone: mem.Phone}
		if err := pryr.Get(ddbClnt, false); err != nil {
			return err
		}

		if err := pryr.Delete(ddbClnt, false); err != nil {
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

		return pryr.Put(ddbClnt, true)
	}

	return nil
}

func prayerRequest(msg messaging.TextMessage, mem object.Member, ddbClnt db.DDBConnecter,
	smsClnt messaging.TextSender) error {
	profanity := msg.CheckProfanity()
	if profanity != "" {
		msg := strings.Replace(messaging.MsgProfanityFound, "PLACEHOLDER", profanity, 1)
		if err := mem.SendMessage(smsClnt, msg); err != nil {
			return err
		}
		return nil
	}

	intercessors, err := FindIntercessors(ddbClnt, mem.Phone)
	if err != nil && errors.Is(err, utility.ErrNoAvailableIntercessors) {
		if err := queuePrayer(msg, mem, ddbClnt, smsClnt); err != nil {
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
		if err := pryr.Put(ddbClnt, false); err != nil {
			return err
		}

		msg := strings.Replace(messaging.MsgPrayerIntro, "PLACEHOLDER", mem.Name, 1)
		if err := intr.SendMessage(smsClnt, msg+pryr.Request); err != nil {
			return err
		}
	}

	return mem.SendMessage(smsClnt, messaging.MsgPrayerSentOut)
}

func FindIntercessors(ddbClnt db.DDBConnecter, skipPhone string) ([]object.Member, error) {
	allPhones, err := getAndPreparePhones(ddbClnt, skipPhone)
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
			intr, err := processIntercessor(ddbClnt, phn)
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

func getAndPreparePhones(ddbClnt db.DDBConnecter, skipPhone string) (object.IntercessorPhones, error) {
	allPhones := object.IntercessorPhones{}
	if err := allPhones.Get(ddbClnt); err != nil {
		return allPhones, err
	}

	// Removes the prayer requestors phone number from IntercessorPhones so that they do not get assigned to pray for
	// their own prayer request. This could happen if the prayer requester is also an intercessor.
	utility.RemoveItem(&allPhones.Phones, skipPhone)

	return allPhones, nil
}

func processIntercessor(ddbClnt db.DDBConnecter, phone string) (*object.Member, error) {
	intr := object.Member{Phone: phone}
	if err := intr.Get(ddbClnt); err != nil {
		return nil, err
	}

	isActive, err := object.IsPrayerActive(ddbClnt, intr.Phone)
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

	if err := intr.Put(ddbClnt); err != nil {
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

func queuePrayer(msg messaging.TextMessage, mem object.Member, ddbClnt db.DDBConnecter,
	smsClnt messaging.TextSender) error {
	pryr := object.Prayer{}
	// A random ID is generated since queued Prayers do not have an intercessor assigned to them. We use the ID in place
	// if the intercessors phone number until there is an available intercessor, at which time the ID will get changed
	// to the available intercessors phone number.
	id, err := utility.GenerateID()
	if err != nil {
		return err
	}

	pryr.IntercessorPhone, pryr.Request, pryr.Requestor = id, msg.Body, mem

	if err := pryr.Put(ddbClnt, true); err != nil {
		return err
	}

	return mem.SendMessage(smsClnt, messaging.MsgPrayerQueued)
}

func completePrayer(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	pryr := object.Prayer{IntercessorPhone: mem.Phone}
	if err := pryr.Get(ddbClnt, false); err != nil {
		return err
	}

	if pryr.Request == "" {
		// Get Dynamodb calls return empty data if the key does not exist in the table. Therefor if prayer request is
		// empty here, it means that it did not exist in the database
		if err := mem.SendMessage(smsClnt, messaging.MsgNoActivePrayer); err != nil {
			return err
		}
		return nil
	}

	if err := mem.SendMessage(smsClnt, messaging.MsgPrayerThankYou); err != nil {
		return err
	}

	msg := strings.Replace(messaging.MsgPrayerConfirmation, "PLACEHOLDER", mem.Name, 1)

	isActive, err := object.IsMemberActive(ddbClnt, pryr.Requestor.Phone)
	if err != nil {
		return err
	}

	if isActive {
		if err := pryr.Requestor.SendMessage(smsClnt, msg); err != nil {
			return err
		}
	} else {
		slog.Warn("Skip sending message, member is not active", "recipient", pryr.Requestor.Phone, "body", msg)
	}

	return pryr.Delete(ddbClnt, false)
}
