package prayertexter

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

func prayerRequest(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, msg messaging.TextMessage, mem object.Member, profanityChecker *messaging.ProfanityChecker) error {
	hasProfanity, err := checkIfProfanity(ctx, smsClnt, mem, msg, profanityChecker)
	if err != nil {
		return err
	} else if hasProfanity {
		return nil
	}

	isValid, err := checkIfRequestValid(ctx, smsClnt, msg, mem)
	if err != nil {
		return err
	} else if !isValid {
		return nil
	}

	handleTriggerWords(&msg, &mem)

	intercessors, err := FindIntercessors(ctx, ddbClnt, mem.Phone)
	if err != nil && errors.Is(err, utility.ErrNoAvailableIntercessors) {
		slog.WarnContext(ctx, "no intercessors available", "request", msg.Body, "requestor", msg.Phone)
		if err = queuePrayer(ctx, ddbClnt, smsClnt, msg, mem); err != nil {
			return utility.WrapError(err, "failed to queue prayer")
		}
		return nil
	} else if err != nil {
		return utility.WrapError(err, "failed to find intercessors")
	}

	for _, intr := range intercessors {
		pryr := object.Prayer{
			Request:   msg.Body,
			Requestor: mem,
		}

		if err = AssignPrayer(ctx, ddbClnt, smsClnt, pryr, intr); err != nil {
			return err
		}
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgPrayerAssigned)
}

// AssignPrayer will save a prayer object to the dynamodb active prayers table with a newly assigned intercessor. It
// will also send the intercessor a text message with the newly assigned prayer request.
func AssignPrayer(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, pryr object.Prayer, intr object.Member) error {
	pryr.Intercessor = intr
	pryr.IntercessorPhone = intr.Phone
	if err := pryr.Put(ctx, ddbClnt, false); err != nil {
		return err
	}

	msg := strings.Replace(messaging.MsgPrayerIntro, "PLACEHOLDER", pryr.Requestor.Name, 1)
	msg = msg + pryr.Request + "\n\n" + messaging.MsgPrayed
	err := pryr.Intercessor.SendMessage(ctx, smsClnt, msg)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "assigned prayer successfully")
	return nil
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
			slog.InfoContext(ctx, "there are no more intercessors left to check")
			if len(intercessors) > 0 {
				slog.InfoContext(ctx, "there is at least one intercessor found, returning this even though it is less "+
					"than the desired number of intercessors per prayer")
				return intercessors, nil
			}

			// There are not any intercessors available at all.
			return nil, utility.ErrNoAvailableIntercessors
		}

		for _, phn := range randPhones {
			// Check if we've already reached the desired number of intercessors.
			if len(intercessors) >= intercessorsPerPrayer {
				return intercessors, nil
			}

			var intr *object.Member
			intr, err = processIntercessor(ctx, ddbClnt, phn)
			if err != nil && errors.Is(err, utility.ErrIntercessorUnavailable) {
				allPhones.RemovePhone(phn)
				continue
			} else if err != nil {
				return nil, err
			}

			intercessors = append(intercessors, *intr)
			allPhones.RemovePhone(phn)
			slog.InfoContext(ctx, "found one available intercessor")
		}
	}

	return intercessors, nil
}

func getAndPreparePhones(ctx context.Context, ddbClnt db.DDBConnecter, skipPhone string) (object.IntercessorPhones, error) {
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
		var canReset bool
		canReset, err = canResetPrayerCount(intr)
		if err != nil {
			return nil, err
		}
		if canReset {
			// Reset intercessor's weekly prayer count
			intr.PrayerCount = 1
			intr.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
		} else {
			return nil, utility.ErrIntercessorUnavailable
		}
	}

	if err = intr.Put(ctx, ddbClnt); err != nil {
		return nil, err
	}
	return &intr, nil
}

func queuePrayer(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender, msg messaging.TextMessage, mem object.Member) error {
	pryr := object.Prayer{}
	// A random ID is generated since queued Prayers do not have an intercessor assigned to them. We use the ID in place
	// if the intercessors phone number until there is an available intercessor, at which time the ID will get changed
	// to the available intercessors phone number.
	id, err := utility.GenerateID()
	if err != nil {
		return err
	}

	pryr.IntercessorPhone = id
	pryr.Request = msg.Body
	pryr.Requestor = mem

	if err = pryr.Put(ctx, ddbClnt, true); err != nil {
		return err
	}

	return mem.SendMessage(ctx, smsClnt, messaging.MsgPrayerQueued)
}
