package prayertexter

import (
	"log/slog"
	"strconv"
	"strings"
	"time"
)

func MainFlow(msg TextMessage, ddbClnt DDBConnecter, smsClnt TextSender) error {
	currTime := time.Now().Format(time.RFC3339)
	id, err := generateID()
	if err != nil {
		return err
	}

	state := State{}
	state.Status, state.TimeStart, state.ID, state.Message = "IN PROGRESS", currTime, id, msg
	if err := state.update(ddbClnt, false); err != nil {
		return err
	}

	mem := Member{Phone: msg.Phone}
	if err := mem.get(ddbClnt); err != nil {
		return err
	}

	// help flow
	if strings.ToLower(msg.Body) == "help" {
		state.Stage = "HELP"
		if err := state.update(ddbClnt, false); err != nil {
			return err
		}
		if err1 := mem.sendMessage(ddbClnt, smsClnt, msgHelp); err1 != nil {
			state.Error = err1.Error()
			state.Status = "FAILED"
			if err2 := state.update(ddbClnt, false); err2 != nil {
				return err2
			}
			return err1
		}

		// cancel flow
	} else if strings.ToLower(msg.Body) == "cancel" || strings.ToLower(msg.Body) == "stop" {
		state.Stage = "MEMBER DELETE"
		if err := state.update(ddbClnt, false); err != nil {
			return err
		}
		if err1 := memberDelete(mem, ddbClnt, smsClnt); err1 != nil {
			state.Error = err1.Error()
			state.Status = "FAILED"
			if err2 := state.update(ddbClnt, false); err2 != nil {
				return err2
			}
			return err1
		}

		//sign up flow
	} else if strings.ToLower(msg.Body) == "pray" || mem.SetupStatus == "in-progress" {
		state.Stage = "SIGN UP"
		if err := state.update(ddbClnt, false); err != nil {
			return err
		}
		if err1 := signUp(msg, mem, ddbClnt, smsClnt); err1 != nil {
			state.Error = err1.Error()
			state.Status = "FAILED"
			if err2 := state.update(ddbClnt, false); err2 != nil {
				return err2
			}
			return err1
		}

		// drop message flow
	} else if mem.SetupStatus == "" {
		state.Stage = "DROP MESSAGE"
		if err := state.update(ddbClnt, false); err != nil {
			return err
		}
		slog.Warn("non registered user, dropping message", "member", mem.Phone)

		// prayer confirmation flow
	} else if strings.ToLower(msg.Body) == "prayed" {
		state.Stage = "COMPLETE PRAYER"
		if err := state.update(ddbClnt, false); err != nil {
			return err
		}
		if err1 := completePrayer(mem, ddbClnt, smsClnt); err1 != nil {
			state.Error = err1.Error()
			state.Status = "FAILED"
			if err2 := state.update(ddbClnt, false); err2 != nil {
				return err2
			}
			return err1
		}

		// prayer request flow
	} else if mem.SetupStatus == "completed" {
		state.Stage = "PRAYER REQUEST"
		if err := state.update(ddbClnt, false); err != nil {
			return err
		}
		if err1 := prayerRequest(msg, mem, ddbClnt, smsClnt); err1 != nil {
			state.Error = err1.Error()
			state.Status = "FAILED"
			if err2 := state.update(ddbClnt, false); err2 != nil {
				return err2
			}
			return err1
		}
	}

	if err := state.update(ddbClnt, true); err != nil {
		return err
	}

	return nil
}

func signUp(msg TextMessage, mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	if strings.ToLower(msg.Body) == "pray" {
		if err := signUpStageOne(mem, ddbClnt, smsClnt); err != nil {
			return err
		}
	} else if msg.Body != "2" && mem.SetupStage == 1 {
		if err := signUpStageTwoA(mem, ddbClnt, smsClnt, msg); err != nil {
			return err
		}
	} else if msg.Body == "2" && mem.SetupStage == 1 {
		if err := signUpStageTwoB(mem, ddbClnt, smsClnt); err != nil {
			return err
		}
	} else if msg.Body == "1" && mem.SetupStage == 2 {
		if err := signUpFinalPrayerMessage(mem, ddbClnt, smsClnt); err != nil {
			return err
		}
	} else if msg.Body == "2" && mem.SetupStage == 2 {
		if err := signUpStageThree(mem, ddbClnt, smsClnt); err != nil {
			return err
		}
	} else if mem.SetupStage == 3 {
		if err := signUpFinalIntercessorMessage(mem, ddbClnt, smsClnt, msg); err != nil {
			return err
		}
	} else {
		if err := signUpWrongInput(mem, ddbClnt, smsClnt); err != nil {
			return err
		}
	}

	return nil
}

func signUpStageOne(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	mem.SetupStatus = "in-progress"
	mem.SetupStage = 1
	if err := mem.put(ddbClnt); err != nil {
		slog.Error("put Member failed during sign up stage 1")
		return err
	}
	if err := mem.sendMessage(ddbClnt, smsClnt, msgNameRequest); err != nil {
		slog.Error("message send failed during sign up stage 1")
		return err
	}

	return nil
}

func signUpStageTwoA(mem Member, ddbClnt DDBConnecter, smsClnt TextSender, msg TextMessage) error {
	mem.SetupStage = 2
	mem.Name = msg.Body
	if err := mem.put(ddbClnt); err != nil {
		slog.Error("put Member failed during sign up stage 2, real name")
		return err
	}
	if err := mem.sendMessage(ddbClnt, smsClnt, msgMemberTypeRequest); err != nil {
		slog.Error("message send failed during sign up stage 2, real name")
		return err
	}

	return nil
}

func signUpStageTwoB(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	mem.SetupStage = 2
	mem.Name = "Anonymous"
	if err := mem.put(ddbClnt); err != nil {
		slog.Error("put Member failed during sign up stage 2, anonymous")
		return err
	}
	if err := mem.sendMessage(ddbClnt, smsClnt, msgMemberTypeRequest); err != nil {
		slog.Error("message send failed during sign up stage 2, anonymous")
		return err
	}

	return nil
}

func signUpFinalPrayerMessage(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	mem.SetupStatus = "completed"
	mem.SetupStage = 99
	mem.Intercessor = false
	if err := mem.put(ddbClnt); err != nil {
		slog.Error("put Member failed during sign up final member message")
		return err
	}

	body := msgPrayerInstructions + "\n\n" + msgSignUpConfirmation
	if err := mem.sendMessage(ddbClnt, smsClnt, body); err != nil {
		slog.Error("message send failed during sign up final member message")
		return err
	}

	return nil
}

func signUpStageThree(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	mem.SetupStage = 3
	mem.Intercessor = true
	if err := mem.put(ddbClnt); err != nil {
		slog.Error("put Member failed during sign up stage 3")
		return err
	}
	if err := mem.sendMessage(ddbClnt, smsClnt, msgPrayerNumRequest); err != nil {
		slog.Error("message send failed during sign up stage 3")
		return err
	}

	return nil
}

func signUpFinalIntercessorMessage(mem Member, ddbClnt DDBConnecter, smsClnt TextSender, msg TextMessage) error {
	num, err := strconv.Atoi(msg.Body)
	if err != nil {
		return signUpWrongInput(mem, ddbClnt, smsClnt)
	}

	phones := IntercessorPhones{}
	if err := phones.get(ddbClnt); err != nil {
		slog.Error("get IntercessorPhones failed during sign up final intercessor message")
		return err
	}
	phones.addPhone(mem.Phone)
	if err := phones.put(ddbClnt); err != nil {
		slog.Error("put IntercessorPhones failed during sign up final intercessor message")
		return err
	}

	mem.SetupStatus = "completed"
	mem.SetupStage = 99
	mem.WeeklyPrayerLimit = num
	mem.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
	if err := mem.put(ddbClnt); err != nil {
		slog.Error("put Member failed during sign up final intercessor message")
		return err
	}

	body := msgPrayerInstructions + "\n\n" + msgIntercessorInstructions + "\n\n" + msgSignUpConfirmation
	if err := mem.sendMessage(ddbClnt, smsClnt, body); err != nil {
		slog.Error("message send failed during sign up final intercessor message - instructions")
		return err
	}

	return nil
}

func signUpWrongInput(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	slog.Warn("wrong input received during sign up", "member", mem.Phone)
	if err := mem.sendMessage(ddbClnt, smsClnt, msgWrongInput); err != nil {
		slog.Error("message send failed during sign up - wrong input")
		return err
	}

	return nil
}

func memberDelete(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	if err := mem.delete(ddbClnt); err != nil {
		slog.Error("failed to delete member during cancellation")
		return err
	}
	if mem.Intercessor {
		phones := IntercessorPhones{}
		if err := phones.get(ddbClnt); err != nil {
			slog.Error("failed to get phone list during cancellation")
			return err
		}
		phones.removePhone(mem.Phone)
		if err := phones.put(ddbClnt); err != nil {
			slog.Error("failed to put phone list during cancellation")
			return err
		}

		// if Member has an active Prayer, then we need to move it to the prayer queue
		// so that the Prayer can get sent to someone else
		isActive, err := isPrayerActive(ddbClnt, mem.Phone)
		if err != nil {
			slog.Error("failed to check if Prayer is active during cancellation")
			return err
		} else if isActive {
			pryr := Prayer{IntercessorPhone: mem.Phone}
			if err := pryr.get(ddbClnt, false); err != nil {
				slog.Error("failed to get Prayer during cancellation")
				return err
			}

			if err := pryr.delete(ddbClnt, false); err != nil {
				slog.Error("failed to delete Prayer during cancellation")
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
				slog.Error("failed to put Prayer during cancellation")
				return err
			}
		}
	}

	if err := mem.sendMessage(ddbClnt, smsClnt, msgRemoveUser); err != nil {
		slog.Error("message send failed during cancellation")
		return err
	}

	return nil
}

func prayerRequest(msg TextMessage, mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	profanity := msg.checkProfanity()
	if profanity != "" {
		msg := strings.Replace(msgProfanityFound, "PLACEHOLDER", profanity, 1)
		mem.sendMessage(ddbClnt, smsClnt, msg)
		return nil
	}

	intercessors, err := findIntercessors(ddbClnt, mem.Phone)
	if err != nil {
		slog.Error("failed to find intercessors during prayer request")
		return err
	} else if intercessors == nil {
		if err := queuePrayer(msg, mem, ddbClnt, smsClnt); err != nil {
			slog.Error("failed to complete queueing prayer request")
			return err
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
			slog.Error("failed to put prayer during prayer request")
			return err
		}

		msg := strings.Replace(msgPrayerIntro, "PLACEHOLDER", mem.Name, 1)
		if err := intr.sendMessage(ddbClnt, smsClnt, msg+pryr.Request); err != nil {
			slog.Error("message send to intercessor failed during prayer request")
			return err
		}
	}

	if err := mem.sendMessage(ddbClnt, smsClnt, msgPrayerSentOut); err != nil {
		slog.Error("message send to member failed during prayer request")
		return err
	}

	return nil
}

func findIntercessors(ddbClnt DDBConnecter, skipPhone string) ([]Member, error) {
	var intercessors []Member

	allPhones := IntercessorPhones{}
	if err := allPhones.get(ddbClnt); err != nil {
		slog.Error("get phone list failed during find intercessors")
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
				slog.Error("get intercessor failed during find intercessors")
				return nil, err
			}

			isActive, err := isPrayerActive(ddbClnt, intr.Phone)
			if err != nil {
				slog.Error("check if active prayer failed during find intercessors")
				return nil, err
			}
			if isActive {
				// this means that intercessor already has 1 active prayer and cannot be used for
				// another 1. there is a limitation of 1 active prayer at a time per intercessor
				allPhones.removePhone(intr.Phone)
				continue
			}

			if intr.PrayerCount < intr.WeeklyPrayerLimit {
				intr.PrayerCount += 1
				intercessors = append(intercessors, intr)
				allPhones.removePhone(intr.Phone)
				if err := intr.put(ddbClnt); err != nil {
					slog.Error("put intercessor failed during find intercessors - +1 count")
					return nil, err
				}
			} else if intr.PrayerCount >= intr.WeeklyPrayerLimit {
				currentTime := time.Now()
				previousTime, err := time.Parse(time.RFC3339, intr.WeeklyPrayerDate)
				if err != nil {
					slog.Error("date parse failed during find intercessors")
					return nil, err
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
						slog.Error("put intercessor failed during find intercessors - count reset")
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

	if err := mem.sendMessage(ddbClnt, smsClnt, msgPrayerQueued); err != nil {
		return err
	}

	return nil
}

func completePrayer(mem Member, ddbClnt DDBConnecter, smsClnt TextSender) error {
	pryr := Prayer{IntercessorPhone: mem.Phone}
	if err := pryr.get(ddbClnt, false); err != nil {
		slog.Error("get prayer failed during complete prayer stage")
		return err
	}

	if pryr.Request == "" {
		// this means that the get prayer did not return an active prayer
		mem.sendMessage(ddbClnt, smsClnt, msgNoActivePrayer)
		return nil
	}

	mem.sendMessage(ddbClnt, smsClnt, msgPrayerThankYou)

	msg := strings.Replace(msgPrayerConfirmation, "PLACEHOLDER", mem.Name, 1)
	pryr.Requestor.sendMessage(ddbClnt, smsClnt, msg)

	if err := pryr.delete(ddbClnt, false); err != nil {
		slog.Error("delete prayer failed during complete prayer stage")
		return err
	}

	return nil
}
