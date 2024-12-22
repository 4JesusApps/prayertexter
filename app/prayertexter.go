package prayertexter

import (
	"log/slog"
	"strconv"
	"strings"
	"time"
)

func MainFlow(txt TextMessage, clnt DDBConnecter, sndr TextSender) error {
	mem := Member{Phone: txt.Phone}
	mem.get(clnt)

	if strings.ToLower(txt.Body) == "pray" || mem.SetupStatus == "in-progress" {
		if err := signUpFlow(txt, mem, clnt, sndr); err != nil {
			return err
		}
	} else if strings.ToLower(txt.Body) == "cancel" || strings.ToLower(txt.Body) == "stop" {
		if err := memberDelete(mem, clnt, sndr); err != nil {
			return err
		}
	} else if mem.SetupStatus == "completed" {
		if err := prayerRequest(txt, mem, clnt, sndr); err != nil {
			return err
		}
	} else if mem.SetupStatus == "" {
		slog.Warn("non registered user, dropping message", "member", mem.Phone)
	}

	return nil
}

func signUpFlow(txt TextMessage, mem Member, clnt DDBConnecter, sndr TextSender) error {
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
		if err := signUpWrongInput(mem, sndr); err != nil {
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
	if err := mem.sendMessage(sndr, msgNameRequest); err != nil {
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
	if err := mem.sendMessage(sndr, msgMemberTypeRequest); err != nil {
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
	if err := mem.sendMessage(sndr, msgMemberTypeRequest); err != nil {
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
	if err := mem.sendMessage(sndr, msgPrayerInstructions); err != nil {
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
	if err := mem.sendMessage(sndr, msgPrayerNumRequest); err != nil {
		slog.Error("message send failed during sign up stage 3")
		return err
	}

	return nil
}

func signUpFinalIntercessorMessage(mem Member, clnt DDBConnecter, sndr TextSender, txt TextMessage) error {
	num, err := strconv.Atoi(txt.Body)
	if err != nil {
		return signUpWrongInput(mem, sndr)
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
	if err := mem.sendMessage(sndr, msgIntercessorInstructions); err != nil {
		slog.Error("message send failed during sign up final intercessor message - instructions")
		return err
	}

	return nil
}

func signUpWrongInput(mem Member, sndr TextSender) error {
	slog.Warn("wrong input received during sign up", "member", mem.Phone)
	if err := mem.sendMessage(sndr, msgWrongInput); err != nil {
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
	if err := mem.sendMessage(sndr, msgRemoveUser); err != nil {
		slog.Error("message send failed during cancellation")
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

		for _, phn := range randPhones {
			intr := Member{Phone: phn}
			if err := intr.get(clnt); err != nil {
				slog.Error("get intercessor failed during find intercessors")
				return nil, err
			}

			if intr.PrayerCount < intr.WeeklyPrayerLimit {
				intercessors = append(intercessors, intr)
				intr.PrayerCount += 1
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
					intercessors = append(intercessors, intr)
					intr.PrayerCount = 1
					allPhones.delPhone(intr.Phone)
					intr.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
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

func prayerRequest(txt TextMessage, mem Member, clnt DDBConnecter, sndr TextSender) error {
	const (
		prayerIntro        = "Hello! Please pray for this person:\n"
		prayerConfirmation = "Your prayer request has been sent out!"
	)

	intercessors, err := findIntercessors(clnt)
	if err != nil {
		slog.Error("failed to find intercessors during prayer request")
		return err
	}

	for _, intr := range intercessors {
		pryr := Prayer{
			Intercessor:      intr,
			IntercessorPhone: intr.Phone,
			Request:          txt.Body,
			Requestor:        mem,
		}
		if err := pryr.put(clnt); err != nil {
			slog.Error("failed to put prayer during prayer request")
			return err
		}
		intr.sendMessage(sndr, prayerIntro+pryr.Request)
	}

	if err := mem.sendMessage(sndr, prayerConfirmation); err != nil {
		slog.Error("message send failed during prayer request")
		return err
	}

	return nil
}
