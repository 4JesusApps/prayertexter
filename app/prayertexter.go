package prayertexter

import (
	"log/slog"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
)

func MainFlow(state State, clnt DDBConnecter, sndr TextSender) error {
	state.Status = "IN PROGRESS"
	state.TimeStart = time.Now().Format(time.RFC3339)
	if err := state.update(clnt); err != nil {
		return err
	}

	txt := state.Message
	mem := Member{Phone: txt.Phone}
	if err := mem.get(clnt); err != nil {
		return err
	}

	// help flow
	if strings.ToLower(txt.Body) == "help" {
		state.Stage = "HELP"
		if err := state.update(clnt); err != nil {
			return err
		}
		if err1 := mem.sendMessage(clnt, sndr, msgHelp); err1 != nil {
			state.Error = err1
			state.Status = "FAILED"
			if err2 := state.update(clnt); err2 != nil {
				return err2
			}
			return err1
		}

		// cancel flow
	} else if strings.ToLower(txt.Body) == "cancel" || strings.ToLower(txt.Body) == "stop" {
		state.Stage = "MEMBER DELETE"
		if err := state.update(clnt); err != nil {
			return err
		}
		if err1 := memberDelete(mem, clnt, sndr); err1 != nil {
			state.Error = err1
			state.Status = "FAILED"
			if err2 := state.update(clnt); err2 != nil {
				return err2
			}
			return err1
		}

		//sign up flow
	} else if strings.ToLower(txt.Body) == "pray" || mem.SetupStatus == "in-progress" {
		state.Stage = "SIGN UP"
		if err := state.update(clnt); err != nil {
			return err
		}
		if err1 := signUp(txt, mem, clnt, sndr); err1 != nil {
			state.Error = err1
			state.Status = "FAILED"
			if err2 := state.update(clnt); err2 != nil {
				return err2
			}
			return err1
		}

		// drop message flow
	} else if mem.SetupStatus == "" {
		state.Stage = "DROP MESSAGE"
		if err := state.update(clnt); err != nil {
			return err
		}
		slog.Warn("non registered user, dropping message", "member", mem.Phone)

		// prayer confirmation flow
	} else if strings.ToLower(txt.Body) == "prayed" {
		state.Stage = "COMPLETE PRAYER"
		if err := state.update(clnt); err != nil {
			return err
		}
		if err1 := completePrayer(mem, clnt, sndr); err1 != nil {
			state.Error = err1
			state.Status = "FAILED"
			if err2 := state.update(clnt); err2 != nil {
				return err2
			}
			return err1
		}

		// prayer request flow
	} else if mem.SetupStatus == "completed" {
		state.Stage = "PRAYER REQUEST"
		if err := state.update(clnt); err != nil {
			return err
		}
		if err1 := prayerRequest(txt, mem, clnt, sndr); err1 != nil {
			state.Error = err1
			state.Status = "FAILED"
			if err2 := state.update(clnt); err2 != nil {
				return err2
			}
			return err1
		}
	}

	state.Status = "COMPLETED"
	state.TimeFinish = time.Now().Format(time.RFC3339)
	if err := state.update(clnt); err != nil {
		return err
	}

	return nil
}

func signUp(txt TextMessage, mem Member, clnt DDBConnecter, sndr TextSender) error {
	if strings.ToLower(txt.Body) == "pray" {
		if err := signUpStageOne(mem, clnt, sndr); err != nil {
			return err
		}
	} else if txt.Body != "2" && mem.SetupStage == 1 {
		if err := signUpStageTwoA(mem, clnt, sndr, txt); err != nil {
			return err
		}
	} else if txt.Body == "2" && mem.SetupStage == 1 {
		if err := signUpStageTwoB(mem, clnt, sndr); err != nil {
			return err
		}
	} else if txt.Body == "1" && mem.SetupStage == 2 {
		if err := signUpFinalPrayerMessage(mem, clnt, sndr); err != nil {
			return err
		}
	} else if txt.Body == "2" && mem.SetupStage == 2 {
		if err := signUpStageThree(mem, clnt, sndr); err != nil {
			return err
		}
	} else if mem.SetupStage == 3 {
		if err := signUpFinalIntercessorMessage(mem, clnt, sndr, txt); err != nil {
			return err
		}
	} else {
		if err := signUpWrongInput(mem, clnt, sndr); err != nil {
			return err
		}
	}

	return nil
}

func signUpStageOne(mem Member, clnt DDBConnecter, sndr TextSender) error {
	mem.SetupStatus = "in-progress"
	mem.SetupStage = 1
	if err := mem.put(clnt); err != nil {
		slog.Error("put Member failed during sign up stage 1")
		return err
	}
	if err := mem.sendMessage(clnt, sndr, msgNameRequest); err != nil {
		slog.Error("message send failed during sign up stage 1")
		return err
	}

	return nil
}

func signUpStageTwoA(mem Member, clnt DDBConnecter, sndr TextSender, txt TextMessage) error {
	mem.SetupStage = 2
	mem.Name = txt.Body
	if err := mem.put(clnt); err != nil {
		slog.Error("put Member failed during sign up stage 2, real name")
		return err
	}
	if err := mem.sendMessage(clnt, sndr, msgMemberTypeRequest); err != nil {
		slog.Error("message send failed during sign up stage 2, real name")
		return err
	}

	return nil
}

func signUpStageTwoB(mem Member, clnt DDBConnecter, sndr TextSender) error {
	mem.SetupStage = 2
	mem.Name = "Anonymous"
	if err := mem.put(clnt); err != nil {
		slog.Error("put Member failed during sign up stage 2, anonymous")
		return err
	}
	if err := mem.sendMessage(clnt, sndr, msgMemberTypeRequest); err != nil {
		slog.Error("message send failed during sign up stage 2, anonymous")
		return err
	}

	return nil
}

func signUpFinalPrayerMessage(mem Member, clnt DDBConnecter, sndr TextSender) error {
	mem.SetupStatus = "completed"
	mem.SetupStage = 99
	mem.Intercessor = false
	if err := mem.put(clnt); err != nil {
		slog.Error("put Member failed during sign up final member message")
		return err
	}

	body := msgPrayerInstructions + "\n\n" + msgSignUpConfirmation
	if err := mem.sendMessage(clnt, sndr, body); err != nil {
		slog.Error("message send failed during sign up final member message")
		return err
	}

	return nil
}

func signUpStageThree(mem Member, clnt DDBConnecter, sndr TextSender) error {
	mem.SetupStage = 3
	mem.Intercessor = true
	if err := mem.put(clnt); err != nil {
		slog.Error("put Member failed during sign up stage 3")
		return err
	}
	if err := mem.sendMessage(clnt, sndr, msgPrayerNumRequest); err != nil {
		slog.Error("message send failed during sign up stage 3")
		return err
	}

	return nil
}

func signUpFinalIntercessorMessage(mem Member, clnt DDBConnecter, sndr TextSender, txt TextMessage) error {
	num, err := strconv.Atoi(txt.Body)
	if err != nil {
		return signUpWrongInput(mem, clnt, sndr)
	}

	phones := IntercessorPhones{}
	if err := phones.get(clnt); err != nil {
		slog.Error("get IntercessorPhones failed during sign up final intercessor message")
		return err
	}
	phones.addPhone(mem.Phone)
	if err := phones.put(clnt); err != nil {
		slog.Error("put IntercessorPhones failed during sign up final intercessor message")
		return err
	}

	mem.SetupStatus = "completed"
	mem.SetupStage = 99
	mem.WeeklyPrayerLimit = num
	mem.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
	if err := mem.put(clnt); err != nil {
		slog.Error("put Member failed during sign up final intercessor message")
		return err
	}

	body := msgPrayerInstructions + "\n\n" + msgIntercessorInstructions + "\n\n" + msgSignUpConfirmation
	if err := mem.sendMessage(clnt, sndr, body); err != nil {
		slog.Error("message send failed during sign up final intercessor message - instructions")
		return err
	}

	return nil
}

func signUpWrongInput(mem Member, clnt DDBConnecter, sndr TextSender) error {
	slog.Warn("wrong input received during sign up", "member", mem.Phone)
	if err := mem.sendMessage(clnt, sndr, msgWrongInput); err != nil {
		slog.Error("message send failed during sign up - wrong input")
		return err
	}

	return nil
}

func memberDelete(mem Member, clnt DDBConnecter, sndr TextSender) error {
	if err := mem.delete(clnt); err != nil {
		slog.Error("failed to delete member during cancellation")
		return err
	}
	if mem.Intercessor {
		phones := IntercessorPhones{}
		if err := phones.get(clnt); err != nil {
			slog.Error("failed to get phone list during cancellation")
			return err
		}
		phones.delPhone(mem.Phone)
		if err := phones.put(clnt); err != nil {
			slog.Error("failed to put phone list during cancellation")
			return err
		}
	}
	if err := mem.sendMessage(clnt, sndr, msgRemoveUser); err != nil {
		slog.Error("message send failed during cancellation")
		return err
	}

	return nil
}

func prayerRequest(txt TextMessage, mem Member, clnt DDBConnecter, sndr TextSender) error {
	profanity := txt.checkProfanity()
	if profanity != "" {
		msg := strings.Replace(msgProfanityFound, "PLACEHOLDER", profanity, 1)
		mem.sendMessage(clnt, sndr, msg)
		return nil
	}

	intercessors, err := findIntercessors(clnt)
	if err != nil {
		slog.Error("failed to find intercessors during prayer request")
		return err
	} else if intercessors == nil {
		if err := queuePrayer(txt, mem, clnt, sndr); err != nil {
			slog.Error("failed to complete queueing prayer request")
			return err
		}

		return nil
	}

	for _, intr := range intercessors {
		pryr := Prayer{
			Intercessor:      intr,
			IntercessorPhone: intr.Phone,
			Request:          txt.Body,
			Requestor:        mem,
		}
		if err := pryr.put(clnt, false); err != nil {
			slog.Error("failed to put prayer during prayer request")
			return err
		}

		msg := strings.Replace(msgPrayerIntro, "PLACEHOLDER", mem.Name, 1)
		if err := intr.sendMessage(clnt, sndr, msg+pryr.Request); err != nil {
			slog.Error("message send to intercessor failed during prayer request")
			return err
		}
	}

	if err := mem.sendMessage(clnt, sndr, msgPrayerSentOut); err != nil {
		slog.Error("message send to member failed during prayer request")
		return err
	}

	return nil
}

func findIntercessors(clnt DDBConnecter) ([]Member, error) {
	var intercessors []Member

	allPhones := IntercessorPhones{}
	if err := allPhones.get(clnt); err != nil {
		slog.Error("get phone list failed during find intercessors")
		return nil, err
	}

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
			if err := intr.get(clnt); err != nil {
				slog.Error("get intercessor failed during find intercessors")
				return nil, err
			}

			pryr := Prayer{IntercessorPhone: intr.Phone}
			isActive, err := pryr.checkIfActive(clnt)
			if err != nil {
				slog.Error("check if active prayer failed during find intercessors")
				return nil, err
			}
			if isActive {
				// this means that intercessor already has 1 active prayer and cannot be used for
				// another 1. there is a limitation of 1 active prayer at a time per intercessor
				allPhones.delPhone(intr.Phone)
				continue
			}

			if intr.PrayerCount < intr.WeeklyPrayerLimit {
				intr.PrayerCount += 1
				intercessors = append(intercessors, intr)
				allPhones.delPhone(intr.Phone)
				if err := intr.put(clnt); err != nil {
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
					allPhones.delPhone(intr.Phone)
					if err := intr.put(clnt); err != nil {
						slog.Error("put intercessor failed during find intercessors - count reset")
						return nil, err
					}
				} else if (diff / 24) < 7 {
					allPhones.delPhone(intr.Phone)
				}
			}
		}
	}

	return intercessors, nil
}

func queuePrayer(txt TextMessage, mem Member, clnt DDBConnecter, sndr TextSender) error {
	pryr := Prayer{}
	isRandom := false

	for !isRandom {
		// a random number is generated and checked in the queued prayers table on the very
		// unlikely chance that a prayer has the same key
		randNum := rand.IntN(9999999999)
		pryr.IntercessorPhone = strconv.Itoa(randNum)
		if err := pryr.get(clnt, true); err != nil {
			return err
		}
		if pryr.Request == "" {
			isRandom = true
		}
	}

	pryr.Request = txt.Body
	pryr.Requestor = mem
	if err := pryr.put(clnt, true); err != nil {
		return err
	}

	if err := mem.sendMessage(clnt, sndr, msgPrayerQueued); err != nil {
		return err
	}

	return nil
}

func completePrayer(mem Member, clnt DDBConnecter, sndr TextSender) error {
	pryr := Prayer{IntercessorPhone: mem.Phone}
	if err := pryr.get(clnt, false); err != nil {
		slog.Error("get prayer failed during complete prayer stage")
		return err
	}

	if pryr.Request == "" {
		// this means that the get prayer did not return an active prayer
		mem.sendMessage(clnt, sndr, msgNoActivePrayer)
		return nil
	}

	mem.sendMessage(clnt, sndr, msgPrayerThankYou)

	msg := strings.Replace(msgPrayerConfirmation, "PLACEHOLDER", mem.Name, 1)
	pryr.Requestor.sendMessage(clnt, sndr, msg)

	if err := pryr.delete(clnt, false); err != nil {
		slog.Error("delete prayer failed during complete prayer stage")
		return err
	}

	return nil
}
