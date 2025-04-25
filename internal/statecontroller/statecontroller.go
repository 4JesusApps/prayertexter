/*
Package statecontroller is the main package for the statecontroller helper application. This package contains all of the
main application logic for statecontroller such as all of the jobs that run on a reoccurring basis.
*/
package statecontroller

import (
	"context"
	"errors"
	"log/slog"

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
	)

	if err := AssignQueuedPrayers(ctx, ddbClnt, smsClnt); err != nil {
		utility.LogError(ctx, err, "failed job", "job", assignQueuedPrayersJob)
	} else {
		slog.InfoContext(ctx, "finished job", "job", assignQueuedPrayersJob)
	}
}

// AssignQueuedPrayers gets all prayers in the queued prayers table if any. It will then attempt to assign each prayer
// to intercessors if there are any available. If a prayer is assigned successfully, it sends the prayer request to the
// intercessors as well as sending a confirmation message to the prayer requestor.
func AssignQueuedPrayers(ctx context.Context, ddbClnt db.DDBConnecter, smsClnt messaging.TextSender) error {
	prayers, err := getQueuedPrayers(ctx, ddbClnt)
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

func getQueuedPrayers(ctx context.Context, ddbClnt db.DDBConnecter) ([]object.Prayer, error) {
	table := viper.GetString(object.QueuedPrayersTableConfigPath)
	return db.GetAllObjects[object.Prayer](ctx, ddbClnt, table)
}
