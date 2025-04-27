/*
Package statecontroller is the main package for the statecontroller helper application. This package contains all of the
main application logic for statecontroller such as all of the jobs that run on a reoccurring basis.
*/
package statecontroller

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/prayertexter"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

// RunJobs will run all of the main statecontroller functions that are meant to be ran as scheduled jobs.
func RunJobs(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) {
	config.InitConfig()

	const (
		assignQueuedPrayersJob = "Assign Queued Prayers"
		remindActivePrayersJob = "Remind Intercessors with Active Prayers"
	)

	if err := AssignQueuedPrayers(ctx, ddbClnt, smsClnt); err != nil {
		utility.LogError(ctx, err, "failed job", "job", assignQueuedPrayersJob)
	} else {
		slog.InfoContext(ctx, "finished job", "job", assignQueuedPrayersJob)
	}

	if err := RemindActiveIntercessors(ctx, ddbClnt, smsClnt); err != nil {
		utility.LogError(ctx, err, "failed job", "job", remindActivePrayersJob)
	} else {
		slog.InfoContext(ctx, "finished job", "job", remindActivePrayersJob)
	}
}

// AssignQueuedPrayers gets all prayers in the queued prayers table if any. It will then attempt to assign each prayer
// to intercessors if there are any available. If a prayer is assigned successfully, it sends the prayer request to the
// intercessors as well as sending a confirmation message to the prayer requestor.
func AssignQueuedPrayers(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	prayers, err := getAllPrayers(ctx, ddbClnt, true)
	if err != nil {
		return utility.WrapError(err, "failed to get queued prayers")
	}

	for _, pryr := range prayers {
		var intercessors []object.Member
		intercessors, err = prayertexter.FindIntercessors(ctx, ddbClnt, pryr.Requestor.Phone)
		if err != nil && errors.Is(err, utility.ErrNoAvailableIntercessors) {
			slog.WarnContext(ctx, "no intercessors available, exiting job")
			break
		} else if err != nil {
			return utility.WrapError(err, "failed to find intercessors")
		}

		for _, intr := range intercessors {
			if err = prayertexter.AssignPrayer(ctx, ddbClnt, smsClnt, pryr, intr); err != nil {
				return utility.WrapError(err, "failed to assign prayer")
			}
		}

		if err = pryr.Delete(ctx, ddbClnt, true); err != nil {
			return err
		}

		if err = pryr.Requestor.SendMessage(ctx, smsClnt, messaging.MsgPrayerAssigned); err != nil {
			return err
		}
	}

	return nil
}

func getAllPrayers(ctx context.Context, ddbClnt db.DDBConnecter, queue bool) ([]object.Prayer, error) {
	table := object.GetPrayerTable(queue)
	return db.GetAllObjects[object.Prayer](ctx, ddbClnt, table)
}

func RemindActiveIntercessors(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	prayerReminderHours := viper.GetInt(object.PrayerReminderHoursConfigPath)

	prayers, err := getAllPrayers(ctx, ddbClnt, false)
	if err != nil {
		return utility.WrapError(err, "failed to get active prayers")
	}

	currentTime := time.Now()
	for _, pryr := range prayers {
		// Set date initially on all active prayers. This is always empty for newly active prayers.
		if pryr.ReminderDate == "" {
			pryr.ReminderDate = currentTime.Format(time.RFC3339)
			if err = pryr.Put(ctx, ddbClnt, false); err != nil {
				return err
			}
			continue
		}

		var previousTime time.Time
		previousTime, err = time.Parse(time.RFC3339, pryr.ReminderDate)
		if err != nil {
			return utility.WrapError(err, "failed to parse time")
		}
		diffTime := currentTime.Sub(previousTime).Hours()
		if diffTime > float64(prayerReminderHours) {
			pryr.ReminderCount++
			pryr.ReminderDate = currentTime.Format(time.RFC3339)
			if err = pryr.Put(ctx, ddbClnt, false); err != nil {
				return err
			}

			msg := strings.Replace(messaging.MsgPrayerReminder, "PLACEHOLDER", pryr.Requestor.Name, 1)
			msg = msg + pryr.Request + "\n\n" + messaging.MsgPrayed
			if err = pryr.Intercessor.SendMessage(ctx, smsClnt, msg); err != nil {
				return err
			}
		}
	}

	return nil
}
