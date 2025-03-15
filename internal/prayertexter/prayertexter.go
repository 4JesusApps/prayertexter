package prayertexter

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/mshort55/prayertexter/internal/db"
	"github.com/mshort55/prayertexter/internal/messaging"
	"github.com/mshort55/prayertexter/internal/object"
	"github.com/mshort55/prayertexter/internal/utility"
)

func MainFlow(msg messaging.TextMessage, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	const (
		mainPreStage            = "PRE"
		mainHelpStage           = "HELP"
		mainMemberDeleteStage   = "MEMBER DELETE"
		mainSignUpStage         = "SIGN UP"
		mainDropMessageStage    = "DROP MESSAGE"
		mainCompletePrayerStage = "COMPLETE PRAYER"
		mainPrayerRequestStage  = "PRAYER REQUEST"
		mainPostStage           = "POST"

		errorPre = "failure during stage: "
	)

	currTime := time.Now().Format(time.RFC3339)
	id, err := utility.GenerateID()
	if err != nil {
		slog.Error(errorPre+mainPreStage, "error", err)
		return err
	}

	state := object.State{}
	state.Status, state.TimeStart, state.ID, state.Message = object.StateInProgress, currTime, id, msg
	if err := state.Update(ddbClnt, false); err != nil {
		slog.Error(errorPre+mainPreStage, "error", err)
		return err
	}

	mem := object.Member{Phone: msg.Phone}
	if err := mem.Get(ddbClnt); err != nil {
		slog.Error(errorPre+mainPreStage, "error", err)
		return err
	}

	switch {
	// HELP STAGE
	// this responds with contact info and is a requirement to get sent to to anyone regardless
	// whether they are a member or not
	case strings.ToLower(msg.Body) == "help":
		state.Stage = mainHelpStage
		if err := state.Update(ddbClnt, false); err != nil {
			slog.Error(errorPre+mainHelpStage, "error", err)
			return err
		}
		if err1 := mem.SendMessage(smsClnt, messaging.MsgHelp); err1 != nil {
			slog.Error(errorPre+mainHelpStage, "error", err1)

			state.Error = err1.Error()
			state.Status = object.StateFailed
			if err2 := state.Update(ddbClnt, false); err2 != nil {
				slog.Error(errorPre+mainHelpStage, "error", err2)
				return err2
			}

			return err1
		}
	// MEMBER DELETE STAGE
	// this removes member from database
	case strings.ToLower(msg.Body) == "cancel" || strings.ToLower(msg.Body) == "stop":
		state.Stage = mainMemberDeleteStage
		if err := state.Update(ddbClnt, false); err != nil {
			slog.Error(errorPre+mainMemberDeleteStage, "error", err)
			return err
		}
		if err1 := memberDelete(mem, ddbClnt, smsClnt); err1 != nil {
			slog.Error(errorPre+mainMemberDeleteStage, "error", err1)

			state.Error = err1.Error()
			state.Status = object.StateFailed
			if err2 := state.Update(ddbClnt, false); err2 != nil {
				slog.Error(errorPre+mainMemberDeleteStage, "error", err2)
				return err2
			}

			return err1
		}
	// SIGN UP STAGE
	// this is the initial sign up process
	case strings.ToLower(msg.Body) == "pray" || mem.SetupStatus == object.MemberSetupInProgress:
		state.Stage = mainSignUpStage
		if err := state.Update(ddbClnt, false); err != nil {
			slog.Error(errorPre+mainSignUpStage, "error", err)
			return err
		}
		if err1 := signUp(msg, mem, ddbClnt, smsClnt); err1 != nil {
			slog.Error(errorPre+mainSignUpStage, "error", err1)

			state.Error = err1.Error()
			state.Status = object.StateFailed
			if err2 := state.Update(ddbClnt, false); err2 != nil {
				slog.Error(errorPre+mainSignUpStage, "error", err2)
				return err2
			}

			return err1
		}
	// DROP MESSAGE STAGE
	// this will drop all messages if they do not meet any of the previous criteria. This serves
	// as a catch all to drop any messages of non members
	case mem.SetupStatus == "":
		state.Stage = mainDropMessageStage
		if err := state.Update(ddbClnt, false); err != nil {
			slog.Error(errorPre+mainDropMessageStage, "error", err)
			return err
		}

		slog.Warn("non registered user, dropping message", "member", mem.Phone, "msg", msg.Body)
	// COMPLETE PRAYER STAGE
	// this is when intercessors pray for a prayer request and send back the confirmation that
	// they prayed. This will let the prayer requestor know that their prayer was prayed for
	case strings.ToLower(msg.Body) == "prayed":
		state.Stage = mainCompletePrayerStage
		if err := state.Update(ddbClnt, false); err != nil {
			slog.Error(errorPre+mainCompletePrayerStage, "error", err)
			return err
		}
		if err1 := completePrayer(mem, ddbClnt, smsClnt); err1 != nil {
			slog.Error(errorPre+mainCompletePrayerStage, "error", err1)

			state.Error = err1.Error()
			state.Status = object.StateFailed
			if err2 := state.Update(ddbClnt, false); err2 != nil {
				slog.Error(errorPre+mainCompletePrayerStage, "error", err2)
				return err2
			}

			return err1
		}
	// PRAYER REQUEST STAGE
	// this is for members sending in prayer requests. It assigns prayers to intercessors
	case mem.SetupStatus == object.MemberSetupComplete:
		state.Stage = mainPrayerRequestStage
		if err := state.Update(ddbClnt, false); err != nil {
			slog.Error(errorPre+mainPrayerRequestStage, "error", err)
			return err
		}
		if err1 := prayerRequest(msg, mem, ddbClnt, smsClnt); err1 != nil {
			slog.Error(errorPre+mainPrayerRequestStage, "error", err1)

			state.Error = err1.Error()
			state.Status = object.StateFailed
			if err2 := state.Update(ddbClnt, false); err2 != nil {
				slog.Error(errorPre+mainPrayerRequestStage, "error", err2)
				return err2
			}

			return err1
		}
	// this should never happen and if it does then it is a bug
	default:
		err := errors.New("unexpected text message input/member status")
		slog.Error("error", "could not satisfy any required conditions", err)
		return err
	}

	if err := state.Update(ddbClnt, true); err != nil {
		slog.Error(errorPre+mainPostStage, "error", err)
		return err
	}

	return nil
}

func signUp(msg messaging.TextMessage, mem object.Member, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	switch {
	case strings.ToLower(msg.Body) == "pray":
		if err := signUpStageOne(mem, ddbClnt, smsClnt); err != nil {
			return fmt.Errorf("signUpStageOne: %w", err)
		}
	case msg.Body != "2" && mem.SetupStage == object.MemberSignUpStepOne:
		if err := signUpStageTwoA(mem, ddbClnt, smsClnt, msg); err != nil {
			return fmt.Errorf("signUpStageTwoA: %w", err)
		}
	case msg.Body == "2" && mem.SetupStage == object.MemberSignUpStepOne:
		if err := signUpStageTwoB(mem, ddbClnt, smsClnt); err != nil {
			return fmt.Errorf("signUpStageTwoB: %w", err)
		}
	case msg.Body == "1" && mem.SetupStage == object.MemberSignUpStepTwo:
		if err := signUpFinalPrayerMessage(mem, ddbClnt, smsClnt); err != nil {
			return fmt.Errorf("signUpFinalPrayerMessage: %w", err)
		}
	case msg.Body == "2" && mem.SetupStage == object.MemberSignUpStepTwo:
		if err := signUpStageThree(mem, ddbClnt, smsClnt); err != nil {
			return fmt.Errorf("signUpStageThree: %w", err)
		}
	case mem.SetupStage == object.MemberSignUpStepThree:
		if err := signUpFinalIntercessorMessage(mem, ddbClnt, smsClnt, msg); err != nil {
			return fmt.Errorf("signUpFinalIntercessorMessage: %w", err)
		}
	default:
		if err := signUpWrongInput(mem, smsClnt); err != nil {
			return fmt.Errorf("signUpWrongInput: %w", err)
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
		return fmt.Errorf("findIntercessors: %w", err)
	} else if intercessors == nil {
		if err := queuePrayer(msg, mem, ddbClnt, smsClnt); err != nil {
			return fmt.Errorf("queuePrayer: %w", err)
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
					return nil, fmt.Errorf("time.Parse: %w", err)
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
