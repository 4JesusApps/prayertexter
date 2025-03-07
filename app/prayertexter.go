package prayertexter

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

func MainFlow(msg TextMessage, ddbClnt DDBConnecter, smsClnt TextSender) error {
	currTime := time.Now().Format(time.RFC3339)
	id, err := generateID()
	if err != nil {
		slog.Error("failure during pre-flow stages", "error", err)
		return err
	}

	state := State{}
	state.Status, state.TimeStart, state.ID, state.Message = "IN PROGRESS", currTime, id, msg
	if err := state.update(ddbClnt, false); err != nil {
		slog.Error("failure during pre-flow stages", "error", err)
		return err
	}

	mem := Member{Phone: msg.Phone}
	if err := mem.get(ddbClnt); err != nil {
		slog.Error("failure during pre-flow stages", "error", err)
		return err
	}

	// HELP FLOW
	// this responds with contact info and is a requirement to get sent to to anyone regardless
	// whether they are a member or not
	if strings.ToLower(msg.Body) == "help" {
		state.Stage = "HELP"
		if err := state.update(ddbClnt, false); err != nil {
			slog.Error("failure during help flow", "error", err)
			return err
		}
		if err1 := mem.sendMessage(smsClnt, msgHelp); err1 != nil {
			state.Error = err1.Error()
			state.Status = "FAILED"
			if err2 := state.update(ddbClnt, false); err2 != nil {
				slog.Error("failure during help flow", "error", err)
				return err2
			}

			slog.Error("failure during help flow", "error", err)
			return err1
		}

		// CANCEL FLOW
		// this removes member from database
	} else if strings.ToLower(msg.Body) == "cancel" || strings.ToLower(msg.Body) == "stop" {
		state.Stage = "MEMBER DELETE"
		if err := state.update(ddbClnt, false); err != nil {
			slog.Error("failure during cancel flow", "error", err)
			return err
		}
		if err1 := memberDelete(mem, ddbClnt, smsClnt); err1 != nil {
			state.Error = err1.Error()
			state.Status = "FAILED"
			if err2 := state.update(ddbClnt, false); err2 != nil {
				slog.Error("failure during cancel flow", "error", err)
				return err2
			}

			slog.Error("failure during cancel flow", "error", err)
			return err1
		}

		// SIGN UP FLOW
		// this is the initial sign up process
	} else if strings.ToLower(msg.Body) == "pray" || mem.SetupStatus == "in-progress" {
		state.Stage = "SIGN UP"
		if err := state.update(ddbClnt, false); err != nil {
			slog.Error("failure during sign up flow", "error", err)
			return err
		}
		if err1 := signUp(msg, mem, ddbClnt, smsClnt); err1 != nil {
			state.Error = err1.Error()
			state.Status = "FAILED"
			if err2 := state.update(ddbClnt, false); err2 != nil {
				slog.Error("failure during sign up flow", "error", err)
				return err2
			}

			slog.Error("failure during sign up flow", "error", err)
			return err1
		}

		// DROP MESSAGE FLOW
		// this will drop all messages if they do not meet any of the previous criteria. This serves
		// as a catch all to drop any messages of non members
	} else if mem.SetupStatus == "" {
		state.Stage = "DROP MESSAGE"
		if err := state.update(ddbClnt, false); err != nil {
			slog.Error("failure during drop message flow", "error", err)
			return err
		}

		slog.Warn("non registered user, dropping message", "member", mem.Phone, "msg", msg.Body)

		// PRAYER CONFIRMATION FLOW
		// this is when intercessors pray for a prayer request and send back the confirmation that
		// they prayed. This will let the prayer requestor know that their prayer was prayed for
	} else if strings.ToLower(msg.Body) == "prayed" {
		state.Stage = "COMPLETE PRAYER"
		if err := state.update(ddbClnt, false); err != nil {
			slog.Error("failure during prayer confirmation flow", "error", err)
			return err
		}
		if err1 := completePrayer(mem, ddbClnt, smsClnt); err1 != nil {
			state.Error = err1.Error()
			state.Status = "FAILED"
			if err2 := state.update(ddbClnt, false); err2 != nil {
				slog.Error("failure during prayer confirmation flow", "error", err)
				return err2
			}

			slog.Error("failure during prayer confirmation flow", "error", err)
			return err1
		}

		// PRAYER REQUEST FLOW
		// this is for members sending in prayer requests. It assigns prayers to intercessors
	} else if mem.SetupStatus == "completed" {
		state.Stage = "PRAYER REQUEST"
		if err := state.update(ddbClnt, false); err != nil {
			slog.Error("failure during prayer request flow", "error", err)
			return err
		}
		if err1 := prayerRequest(msg, mem, ddbClnt, smsClnt); err1 != nil {
			state.Error = err1.Error()
			state.Status = "FAILED"
			if err2 := state.update(ddbClnt, false); err2 != nil {
				slog.Error("failure during prayer request flow", "error", err)
				return err2
			}

			slog.Error("failure during prayer request flow", "error", err)
			return err1
		}
	}

	if err := state.update(ddbClnt, true); err != nil {
		slog.Error("failure during flow completion", "error", err)
		return err
	}

	return nil
}

func signUp(msg TextMessage, mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	if strings.ToLower(msg.Body) == "pray" {
		if err := signUpStageOne(mem, ddbClnt, smsClnt); err != nil {
			return fmt.Errorf("signUpStageOne: %w", err)
		}
	} else if msg.Body != "2" && mem.SetupStage == 1 {
		if err := signUpStageTwoA(mem, ddbClnt, smsClnt, msg); err != nil {
			return fmt.Errorf("signUpStageTwoA: %w", err)
		}
	} else if msg.Body == "2" && mem.SetupStage == 1 {
		if err := signUpStageTwoB(mem, ddbClnt, smsClnt); err != nil {
			return fmt.Errorf("signUpStageTwoB: %w", err)
		}
	} else if msg.Body == "1" && mem.SetupStage == 2 {
		if err := signUpFinalPrayerMessage(mem, ddbClnt, smsClnt); err != nil {
			return fmt.Errorf("signUpFinalPrayerMessage: %w", err)
		}
	} else if msg.Body == "2" && mem.SetupStage == 2 {
		if err := signUpStageThree(mem, ddbClnt, smsClnt); err != nil {
			return fmt.Errorf("signUpStageThree: %w", err)
		}
	} else if mem.SetupStage == 3 {
		if err := signUpFinalIntercessorMessage(mem, ddbClnt, smsClnt, msg); err != nil {
			return fmt.Errorf("signUpFinalIntercessorMessage: %w", err)
		}
	} else {
		if err := signUpWrongInput(mem, smsClnt); err != nil {
			return fmt.Errorf("signUpWrongInput: %w", err)
		}
	}

	return nil
}

func signUpStageOne(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	mem.SetupStatus = "in-progress"
	mem.SetupStage = 1
	if err := mem.put(ddbClnt); err != nil {
		return err
	}

	if err := mem.sendMessage(smsClnt, msgNameRequest); err != nil {
		return err
	}

	return nil
}

func signUpStageTwoA(mem Member, ddbClnt DDBConnecter, smsClnt TextSender, msg TextMessage) error {
	mem.SetupStage = 2
	mem.Name = msg.Body
	if err := mem.put(ddbClnt); err != nil {
		return err
	}

	if err := mem.sendMessage(smsClnt, msgMemberTypeRequest); err != nil {
		return err
	}

	return nil
}

func signUpStageTwoB(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	mem.SetupStage = 2
	mem.Name = "Anonymous"
	if err := mem.put(ddbClnt); err != nil {
		return err
	}

	if err := mem.sendMessage(smsClnt, msgMemberTypeRequest); err != nil {
		return err
	}

	return nil
}

func signUpFinalPrayerMessage(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	mem.SetupStatus = "completed"
	mem.SetupStage = 99
	mem.Intercessor = false
	if err := mem.put(ddbClnt); err != nil {
		return err
	}

	body := msgPrayerInstructions + "\n\n" + msgSignUpConfirmation
	if err := mem.sendMessage(smsClnt, body); err != nil {
		return err
	}

	return nil
}

func signUpStageThree(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	mem.SetupStage = 3
	mem.Intercessor = true
	if err := mem.put(ddbClnt); err != nil {
		return err
	}

	if err := mem.sendMessage(smsClnt, msgPrayerNumRequest); err != nil {
		return err
	}

	return nil
}

func signUpFinalIntercessorMessage(mem Member, ddbClnt DDBConnecter, smsClnt TextSender, msg TextMessage) error {
	num, err := strconv.Atoi(msg.Body)
	if err != nil {
		return signUpWrongInput(mem, smsClnt)
	}

	phones := IntercessorPhones{}
	if err := phones.get(ddbClnt); err != nil {
		return err
	}

	phones.addPhone(mem.Phone)
	if err := phones.put(ddbClnt); err != nil {
		return err
	}

	mem.SetupStatus = "completed"
	mem.SetupStage = 99
	mem.WeeklyPrayerLimit = num
	mem.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
	if err := mem.put(ddbClnt); err != nil {
		return err
	}

	body := msgPrayerInstructions + "\n\n" + msgIntercessorInstructions + "\n\n" + msgSignUpConfirmation
	if err := mem.sendMessage(smsClnt, body); err != nil {
		return err
	}

	return nil
}

func signUpWrongInput(mem Member, smsClnt TextSender) error {
	slog.Warn("wrong input received during sign up", "member", mem.Phone)

	if err := mem.sendMessage(smsClnt, msgWrongInput); err != nil {
		return err
	}

	return nil
}

func memberDelete(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	if err := mem.delete(ddbClnt); err != nil {
		return err
	}
	if mem.Intercessor {
		phones := IntercessorPhones{}
		if err := phones.get(ddbClnt); err != nil {
			return err
		}
		phones.removePhone(mem.Phone)
		if err := phones.put(ddbClnt); err != nil {
			return err
		}

		// if Member has an active Prayer, then we need to move it to the prayer queue
		// so that the Prayer can get sent to someone else
		isActive, err := isPrayerActive(ddbClnt, mem.Phone)
		if err != nil {
			return err
		} else if isActive {
			pryr := Prayer{IntercessorPhone: mem.Phone}
			if err := pryr.get(ddbClnt, false); err != nil {
				return err
			}

			if err := pryr.delete(ddbClnt, false); err != nil {
				return err
			}

			// random ID is generated here since queued Prayers do not have an intercessor assigned
			// to them
			id, err := generateID()
			if err != nil {
				return err
			}
			pryr.IntercessorPhone, pryr.Intercessor = id, Member{}

			if err := pryr.put(ddbClnt, true); err != nil {
				return err
			}
		}
	}

	if err := mem.sendMessage(smsClnt, msgRemoveUser); err != nil {
		return err
	}

	return nil
}

func prayerRequest(msg TextMessage, mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	profanity := msg.checkProfanity()
	if profanity != "" {
		msg := strings.Replace(msgProfanityFound, "PLACEHOLDER", profanity, 1)
		if err := mem.sendMessage(smsClnt, msg); err != nil {
			return err
		}
		return nil
	}

	intercessors, err := findIntercessors(ddbClnt, mem.Phone)
	if err != nil {
		return fmt.Errorf("findIntercessors: %w", err)
	} else if intercessors == nil {
		if err := queuePrayer(msg, mem, ddbClnt, smsClnt); err != nil {
			return fmt.Errorf("queuePrayer: %w", err)
		}

		return nil
	}

	for _, intr := range intercessors {
		pryr := Prayer{
			Intercessor:      intr,
			IntercessorPhone: intr.Phone,
			Request:          msg.Body,
			Requestor:        mem,
		}
		if err := pryr.put(ddbClnt, false); err != nil {
			return err
		}

		msg := strings.Replace(msgPrayerIntro, "PLACEHOLDER", mem.Name, 1)
		if err := intr.sendMessage(smsClnt, msg+pryr.Request); err != nil {
			return err
		}
	}

	if err := mem.sendMessage(smsClnt, msgPrayerSentOut); err != nil {
		return err
	}

	return nil
}

func findIntercessors(ddbClnt DDBConnecter, skipPhone string) ([]Member, error) {
	var intercessors []Member

	allPhones := IntercessorPhones{}
	if err := allPhones.get(ddbClnt); err != nil {
		return nil, err
	}

	// this will remove the member's (prayer requestor) phone number from the intercessor phone
	// list so they don't get assigned to pray for their own prayer request
	removeItem(&allPhones.Phones, skipPhone)

	for len(intercessors) < numIntercessorsPerPrayer {
		randPhones := allPhones.genRandPhones()
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
			intr := Member{Phone: phn}
			if err := intr.get(ddbClnt); err != nil {
				return nil, err
			}

			isActive, err := isPrayerActive(ddbClnt, intr.Phone)
			if err != nil {
				return nil, err
			}
			if isActive {
				// this means that intercessor already has 1 active prayer and cannot be used for
				// another 1. there is a limitation of 1 active prayer at a time per intercessor
				allPhones.removePhone(intr.Phone)
				continue
			}

			if intr.PrayerCount < intr.WeeklyPrayerLimit {
				intr.PrayerCount ++
				intercessors = append(intercessors, intr)
				allPhones.removePhone(intr.Phone)
				if err := intr.put(ddbClnt); err != nil {
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
				if (diff / 24) > 7 {
					intr.PrayerCount = 1
					intr.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
					intercessors = append(intercessors, intr)
					allPhones.removePhone(intr.Phone)
					if err := intr.put(ddbClnt); err != nil {
						return nil, err
					}
				} else if (diff / 24) < 7 {
					allPhones.removePhone(intr.Phone)
				}
			}
		}
	}

	return intercessors, nil
}

func queuePrayer(msg TextMessage, mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	pryr := Prayer{}
	// random ID is generated here since queued Prayers do not have an intercessor assigned
	// to them
	id, err := generateID()
	if err != nil {
		return err
	}

	pryr.IntercessorPhone, pryr.Request, pryr.Requestor = id, msg.Body, mem

	if err := pryr.put(ddbClnt, true); err != nil {
		return err
	}

	if err := mem.sendMessage(smsClnt, msgPrayerQueued); err != nil {
		return err
	}

	return nil
}

func completePrayer(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	pryr := Prayer{IntercessorPhone: mem.Phone}
	if err := pryr.get(ddbClnt, false); err != nil {
		return err
	}

	if pryr.Request == "" {
		// this means that the get prayer did not return an active prayer
		if err := mem.sendMessage(smsClnt, msgNoActivePrayer); err != nil {
			return err
		}
		return nil
	}

	if err := mem.sendMessage(smsClnt, msgPrayerThankYou); err != nil {
		return err
	}

	msg := strings.Replace(msgPrayerConfirmation, "PLACEHOLDER", mem.Name, 1)

	isActive, err := isMemberActive(ddbClnt, pryr.Requestor.Phone)
	if err != nil {
		return err
	}

	if isActive {
		if err := pryr.Requestor.sendMessage(smsClnt, msg); err != nil {
			return err
		}
	} else {
		slog.Warn("Skip sending message, member is not active", "recipient", pryr.Requestor.Phone, "body", msg)
	}

	if err := pryr.delete(ddbClnt, false); err != nil {
		return err
	}

	return nil
}
