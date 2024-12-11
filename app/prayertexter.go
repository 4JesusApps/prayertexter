package prayertexter

import (
	"log"
	"strconv"
	"strings"
	"time"
)


type TextMessage struct {
	Body  string `json:"body"`
	Phone string `json:"phone-number"`
}


func signUp(txt TextMessage, mem Member) {
	const (
		nameRequest             = "Text your name, or 2 to stay anonymous"
		memberTypeRequest       = "Text 1 for prayer request, or 2 to be added to the intercessors list (to pray for others)"
		prayerInstructions      = "You are now signed up to send prayer requests! Please send them directly to this number."
		prayerNumRequest        = "Send the max number of prayer texts you are willing to receive and pray for per week."
		intercessorInstructions = "You are now signed up to receive prayer requests. Please try to pray for the requests ASAP. Once you are done praying, send 'prayed' back to this number for confirmation."
		wrongInput              = "Wrong input received during sign up process. Please try again."
	)

	if strings.ToLower(txt.Body) == "pray" {
		// stage 1
		mem.SetupStatus = "in-progress"
		mem.SetupStage = 1
		mem.put()
		mem.sendMessage(nameRequest)
	} else if txt.Body != "2" && mem.SetupStage == 1 {
		// stage 2 name request
		mem.SetupStage = 2
		mem.Name = txt.Body
		mem.put()
		mem.sendMessage(memberTypeRequest)
	} else if txt.Body == "2" && mem.SetupStage == 1 {
		// stage 2 name request
		mem.SetupStage = 2
		mem.Name = "Anonymous"
		mem.put()
		mem.sendMessage(memberTypeRequest)
	} else if txt.Body == "1" && mem.SetupStage == 2 {
		// final message for member sign up
		mem.SetupStatus = "completed"
		mem.SetupStage = 99
		mem.Intercessor = false
		mem.put()
		mem.sendMessage(prayerInstructions)
	} else if txt.Body == "2" && mem.SetupStage == 2 {
		// stage 3 intercessor sign up
		mem.SetupStage = 3
		mem.Intercessor = true
		mem.put()
		mem.sendMessage(prayerNumRequest)
	} else if mem.SetupStage == 3 {
		// final message for intercessor sign up
		if num, err := strconv.Atoi(txt.Body); err == nil {
			phones := IntercessorPhones{}.get()
			phones = phones.addPhone(mem.Phone)
			phones.put()

			mem.SetupStatus = "completed"
			mem.SetupStage = 99
			mem.WeeklyPrayerLimit = num
			mem.put()
			mem.sendMessage(intercessorInstructions)
		} else {
			mem.sendMessage(wrongInput)
		}
	} else {
		// catch all response for incorrect input
		mem.sendMessage(wrongInput)
	}
}

func findIntercessors() []Member {
	var intercessors []Member

	for len(intercessors) < numIntercessorsPerPrayer {
		allPhones := IntercessorPhones{}.get()
		randPhones := allPhones.genRandPhones()

		for _, phn := range randPhones {
			intr := Member{Phone: phn}.get()

			if intr.PrayerCount < intr.WeeklyPrayerLimit {
				intercessors = append(intercessors, intr)
				intr.PrayerCount += 1
				allPhones = allPhones.delPhone(intr.Phone)
				intr.put()

				if intr.WeeklyPrayerDate == "" {
					intr.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
					intr.put()
				}
			} else if intr.PrayerCount >= intr.WeeklyPrayerLimit {
				currentTime := time.Now()
				previousTime, err := time.Parse(time.RFC3339, intr.WeeklyPrayerDate)
				if err != nil {
					log.Fatalf("date parse failed, %v", err)
				}

				diff := currentTime.Sub(previousTime).Hours()
				// reset prayer counter if time between now and weekly prayer date is greater than
				// 7 days
				if (diff / 24) > 7 {
					intercessors = append(intercessors, intr)
					intr.PrayerCount = 1
					allPhones = allPhones.delPhone(intr.Phone)
					intr.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
					intr.put()
				} else if (diff / 24) < 7 {
					allPhones = allPhones.delPhone(intr.Phone)
				}
			}
		}
	}

	return intercessors
}

func prayerRequest(txt TextMessage, mem Member) {
	const (
		prayerIntro        = "Hello! Please pray for this person:\n"
		prayerConfirmation = "Your prayer request has been sent out!"
	)

	intercessors := findIntercessors()

	for _, intr := range intercessors {
		pryr := Prayer{
			Intercessor:      intr,
			IntercessorPhone: intr.Phone,
			Request:          txt.Body,
			Requestor:        mem,
		}
		pryr.put()
		intr.sendMessage(prayerIntro+pryr.Request)
	}

	mem.sendMessage(prayerConfirmation)
}

func MainFlow(txt TextMessage) {
	const (
		removeUser = "You have been removed from prayer texter. If you ever want to sign back up, text the word pray to this number."
	)

	

	mem := Member{Phone: txt.Phone}.get()

	if strings.ToLower(txt.Body) == "pray" || mem.SetupStatus == "in-progress" {
		signUp(txt, mem)
	} else if strings.ToLower(txt.Body) == "cancel" || strings.ToLower(txt.Body) == "stop" {
		mem.delete()
		if mem.Intercessor {
			phones := IntercessorPhones{}.get()
			phones = phones.delPhone(mem.Phone)
			phones.put()
		}
		mem.sendMessage(removeUser)
	} else if mem.SetupStatus == "completed" {
		prayerRequest(txt, mem)
	} else if mem.SetupStatus == "" {
		log.Printf("%v is not a registered user, dropping message", mem.Phone)
	}
}