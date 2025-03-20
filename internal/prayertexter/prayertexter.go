package prayertexter

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/mshort55/prayertexter/internal/db"
	"github.com/mshort55/prayertexter/internal/messaging"
	"github.com/mshort55/prayertexter/internal/object"
	"github.com/mshort55/prayertexter/internal/utility"
)

const (
	stageErrPre = "failure during stage "
)

func MainFlow(msg messaging.TextMessage, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	const (
		preStage            = "PRE"
		helpStage           = "HELP"
		memberDeleteStage   = "MEMBER DELETE"
		signUpStage         = "SIGN UP"
		dropMessageStage    = "DROP MESSAGE"
		completePrayerStage = "COMPLETE PRAYER"
		prayerRequestStage  = "PRAYER REQUEST"
		postStage           = "POST"
	)

	currTime := time.Now().Format(time.RFC3339)
	id, err := utility.GenerateID()
	if err != nil {
		return utility.LogAndWrapError(err, stageErrPre+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	state := object.State{}
	state.Status, state.TimeStart, state.ID, state.Message = object.StateInProgress, currTime, id, msg
	if err := state.Update(ddbClnt, false); err != nil {
		return utility.LogAndWrapError(err, stageErrPre+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	mem := object.Member{Phone: msg.Phone}
	if err := mem.Get(ddbClnt); err != nil {
		return utility.LogAndWrapError(err, stageErrPre+preStage, "phone", msg.Phone, "msg", msg.Body)
	}

	var stageErr error

	switch {
	// HELP STAGE
	// This responds with contact info and is a requirement to get sent to to anyone regardless
	// whether they are a member or not.
	case strings.ToLower(msg.Body) == "help":
        stageErr = executeStage(ddbClnt, helpStage, &state, func() error {
            return mem.SendMessage(smsClnt, messaging.MsgHelp)
        })

	// MEMBER DELETE STAGE
	// This removes member from prayertexter.
	case strings.ToLower(msg.Body) == "cancel" || strings.ToLower(msg.Body) == "stop":
        stageErr = executeStage(ddbClnt, memberDeleteStage, &state, func() error {
            return memberDelete(mem, ddbClnt, smsClnt)
        })

	// SIGN UP STAGE
	// This is the initial sign up process.
	case strings.ToLower(msg.Body) == "pray" || mem.SetupStatus == object.MemberSetupInProgress:
        stageErr = executeStage(ddbClnt, signUpStage, &state, func() error {
            return signUp(msg, mem, ddbClnt, smsClnt)
        })

	// DROP MESSAGE STAGE
	// This will drop all messages if they do not meet any of the previous criteria. This serves
	// as a catch all to drop any messages of non members.
	case mem.SetupStatus == "":
		slog.Warn("non registered user dropping message", "phone", mem.Phone, "msg", msg.Body)
		stageErr = executeStage(ddbClnt, dropMessageStage, &state, func() error {
            return nil
        })

	// COMPLETE PRAYER STAGE
	// This is when intercessors pray for a prayer request and send back the confirmation that
	// they prayed. This will let the prayer requestor know that their prayer was prayed for.
	case strings.ToLower(msg.Body) == "prayed":
        stageErr = executeStage(ddbClnt, completePrayerStage, &state, func() error {
            return completePrayer(mem, ddbClnt, smsClnt)
        })

	// PRAYER REQUEST STAGE
	// This is for members sending in prayer requests. It assigns prayers to intercessors.
	case mem.SetupStatus == object.MemberSetupComplete:
        stageErr = executeStage(ddbClnt, prayerRequestStage, &state, func() error {
            return prayerRequest(msg, mem, ddbClnt, smsClnt)
        })

	// This should never happen and if it does then it is a bug.
	default:
		err := errors.New("unexpected text message input/member status")
		return utility.LogError(err, "could not satisfy any required conditions", "phone", mem.Phone, "msg", msg.Body)
	}

	if stageErr != nil {
        return stageErr
    }

	if err := state.Update(ddbClnt, true); err != nil {
		return utility.LogError(err, stageErrPre+postStage, "phone", mem.Phone, "msg", msg.Body)
	}

	return nil
}

func executeStage(ddbClnt db.DDBConnecter, stageName string, state *object.State, stageFunc func() error) error {
	state.Stage = stageName
	if err := state.Update(ddbClnt, false); err != nil {
		slog.Error(stageErrPre+stageName, "error", err)
		return err
	}

	if err := stageFunc(); err != nil {
		slog.Error(stageErrPre+stageName, "error", err)

		state.Error = err.Error()
		state.Status = object.StateFailed
		if updateErr := state.Update(ddbClnt, false); updateErr != nil {
			slog.Error(stageErrPre+stageName, "error", updateErr)
			return updateErr
		}

		return err
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
		if err := signUpWrongInput(mem, smsClnt); err != nil {
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

	if err := mem.SendMessage(smsClnt, messaging.MsgNameRequest); err != nil {
		return err
	}

	return nil
}

func signUpStageTwoA(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender,
	msg messaging.TextMessage) error {
	mem.SetupStage = object.MemberSignUpStepTwo
	mem.Name = msg.Body
	if err := mem.Put(ddbClnt); err != nil {
		return err
	}

	if err := mem.SendMessage(smsClnt, messaging.MsgMemberTypeRequest); err != nil {
		return err
	}

	return nil
}

func signUpStageTwoB(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	mem.SetupStage = object.MemberSignUpStepTwo
	mem.Name = "Anonymous"
	if err := mem.Put(ddbClnt); err != nil {
		return err
	}

	if err := mem.SendMessage(smsClnt, messaging.MsgMemberTypeRequest); err != nil {
		return err
	}

	return nil
}

func signUpFinalPrayerMessage(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	mem.SetupStatus = object.MemberSetupComplete
	mem.SetupStage = object.MemberSignUpStepFinal
	mem.Intercessor = false
	if err := mem.Put(ddbClnt); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgSignUpConfirmation
	if err := mem.SendMessage(smsClnt, body); err != nil {
		return err
	}

	return nil
}

func signUpStageThree(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	mem.SetupStage = object.MemberSignUpStepThree
	mem.Intercessor = true
	if err := mem.Put(ddbClnt); err != nil {
		return err
	}

	if err := mem.SendMessage(smsClnt, messaging.MsgPrayerNumRequest); err != nil {
		return err
	}

	return nil
}

func signUpFinalIntercessorMessage(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender,
	msg messaging.TextMessage) error {
	num, err := strconv.Atoi(msg.Body)
	if err != nil {
		return signUpWrongInput(mem, smsClnt)
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
	if err := mem.SendMessage(smsClnt, body); err != nil {
		return err
	}

	return nil
}

func signUpWrongInput(mem object.Member, smsClnt messaging.TextSender) error {
	slog.Warn("wrong input received during sign up", "member", mem.Phone)

	if err := mem.SendMessage(smsClnt, messaging.MsgWrongInput); err != nil {
		return err
	}

	return nil
}

func memberDelete(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	if err := mem.Delete(ddbClnt); err != nil {
		return err
	}
	if mem.Intercessor {
		phones := object.IntercessorPhones{}
		if err := phones.Get(ddbClnt); err != nil {
			return err
		}
		phones.RemovePhone(mem.Phone)
		if err := phones.Put(ddbClnt); err != nil {
			return err
		}

		// if object.Member has an active Prayer, then we need to move it to the prayer queue
		// so that the Prayer can get sent to someone else
		isActive, err := object.IsPrayerActive(ddbClnt, mem.Phone)
		if err != nil {
			return err
		} else if isActive {
			pryr := object.Prayer{IntercessorPhone: mem.Phone}
			if err := pryr.Get(ddbClnt, false); err != nil {
				return err
			}

			if err := pryr.Delete(ddbClnt, false); err != nil {
				return err
			}

			// random ID is generated here since queued Prayers do not have an intercessor assigned
			// to them
			id, err := utility.GenerateID()
			if err != nil {
				return err
			}
			pryr.IntercessorPhone, pryr.Intercessor = id, object.Member{}

			if err := pryr.Put(ddbClnt, true); err != nil {
				return err
			}
		}
	}

	if err := mem.SendMessage(smsClnt, messaging.MsgRemoveUser); err != nil {
		return err
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
	if err != nil {
		return utility.WrapError(err, "failed to find intercessor")
	} else if intercessors == nil {
		if err := queuePrayer(msg, mem, ddbClnt, smsClnt); err != nil {
			return utility.WrapError(err, "failed to queue prayer")
		}

		return nil
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

	if err := mem.SendMessage(smsClnt, messaging.MsgPrayerSentOut); err != nil {
		return err
	}

	return nil
}

func FindIntercessors(ddbClnt db.DDBConnecter, skipPhone string) ([]object.Member, error) {
	var intercessors []object.Member

	allPhones := object.IntercessorPhones{}
	if err := allPhones.Get(ddbClnt); err != nil {
		return nil, err
	}

	// this will remove the member's (prayer requestor) phone number from the intercessor phone
	// list so they don't get assigned to pray for their own prayer request
	utility.RemoveItem(&allPhones.Phones, skipPhone)

	for len(intercessors) < object.NumIntercessorsPerPrayer {
		randPhones := allPhones.GenRandPhones()
		if randPhones == nil {
			// this means that there are no more available intercessors for a prayer request
			if len(intercessors) != 0 {
				// this means that we found at least one intercessor, yet it is under the desired
				// number set for each prayer request. we will return the intercessor/s because
				// it's better than none
				return intercessors, nil
			} else if len(intercessors) == 0 {
				// this means that we cannot find a single intercessor for a prayer request
				return nil, nil
			}
		}

		for _, phn := range randPhones {
			intr := object.Member{Phone: phn}
			if err := intr.Get(ddbClnt); err != nil {
				return nil, err
			}

			isActive, err := object.IsPrayerActive(ddbClnt, intr.Phone)
			if err != nil {
				return nil, err
			}
			if isActive {
				// this means that intercessor already has 1 active prayer and cannot be used for
				// another 1. there is a limitation of 1 active prayer at a time per intercessor
				allPhones.RemovePhone(intr.Phone)
				continue
			}

			if intr.PrayerCount < intr.WeeklyPrayerLimit {
				intr.PrayerCount++
				intercessors = append(intercessors, intr)
				allPhones.RemovePhone(intr.Phone)
				if err := intr.Put(ddbClnt); err != nil {
					return nil, err
				}
			} else if intr.PrayerCount >= intr.WeeklyPrayerLimit {
				currentTime := time.Now()
				previousTime, err := time.Parse(time.RFC3339, intr.WeeklyPrayerDate)
				if err != nil {
					return nil, utility.WrapError(err, "failed to parse time")
				}

				diff := currentTime.Sub(previousTime).Hours()
				// reset prayer counter if time between now and weekly prayer date is greater than
				// 7 days and select intercessor
				hoursInDay := 24.0
				daysInWeek := 7.0

				if (diff / hoursInDay) > daysInWeek {
					intr.PrayerCount = 1
					intr.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
					intercessors = append(intercessors, intr)
					allPhones.RemovePhone(intr.Phone)
					if err := intr.Put(ddbClnt); err != nil {
						return nil, err
					}
				} else if (diff / hoursInDay) < daysInWeek {
					allPhones.RemovePhone(intr.Phone)
				}
			}
		}
	}

	return intercessors, nil
}

func queuePrayer(msg messaging.TextMessage, mem object.Member, ddbClnt db.DDBConnecter,
	smsClnt messaging.TextSender) error {
	pryr := object.Prayer{}
	// random ID is generated here since queued Prayers do not have an intercessor assigned
	// to them
	id, err := utility.GenerateID()
	if err != nil {
		return err
	}

	pryr.IntercessorPhone, pryr.Request, pryr.Requestor = id, msg.Body, mem

	if err := pryr.Put(ddbClnt, true); err != nil {
		return err
	}

	if err := mem.SendMessage(smsClnt, messaging.MsgPrayerQueued); err != nil {
		return err
	}

	return nil
}

func completePrayer(mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	pryr := object.Prayer{IntercessorPhone: mem.Phone}
	if err := pryr.Get(ddbClnt, false); err != nil {
		return err
	}

	if pryr.Request == "" {
		// this means that the get prayer did not return an active prayer
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

	if err := pryr.Delete(ddbClnt, false); err != nil {
		return err
	}

	return nil
}
