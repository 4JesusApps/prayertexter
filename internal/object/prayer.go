package object

import (
	"context"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

type Prayer struct {
	Intercessor      Member
	IntercessorPhone string
	Request          string
	Requestor        Member
}

const (
	DefaultActivePrayersTable    = "ActivePrayers"
	ActivePrayersTableConfigPath = "conf.aws.db.prayer.activetable"

	DefaultQueuedPrayersTable    = "QueuedPrayers"
	QueuedPrayersTableConfigPath = "conf.aws.db.prayer.queuetable"

	DefaultIntercessorsPerPrayer    = 2
	IntercessorsPerPrayerConfigPath = "conf.intercessorsperprayer"

	PrayersAttribute = "IntercessorPhone"
)

func (p *Prayer) Get(ctx context.Context, ddbClnt db.DDBConnecter, queue bool) error {
	// Queue determines whether ActivePrayers or PrayersQueue table is used for dynamodb get requests.
	table := GetPrayerTable(queue)
	pryr, err := db.GetDdbObject[Prayer](ctx, ddbClnt, PrayersAttribute, p.IntercessorPhone, table)
	if err != nil {
		return err
	}

	// This is important so that the original Prayer object doesn't get reset to all empty struct values if the Prayer
	// does not exist in dynamodb.
	if pryr.IntercessorPhone != "" {
		*p = *pryr
	}

	return nil
}

func (p *Prayer) Put(ctx context.Context, ddbClnt db.DDBConnecter, queue bool) error {
	// Prayers get queued in order to save them for a time when intercessors are available. This will change the
	// dynamodb table that the prayer is saved to.
	table := GetPrayerTable(queue)

	return db.PutDdbObject(ctx, ddbClnt, table, p)
}

func (p *Prayer) Delete(ctx context.Context, ddbClnt db.DDBConnecter, queue bool) error {
	table := GetPrayerTable(queue)

	return db.DelDdbItem(ctx, ddbClnt, PrayersAttribute, p.IntercessorPhone, table)
}

func GetPrayerTable(queue bool) string {
	queuedPrayersTable := viper.GetString(QueuedPrayersTableConfigPath)
	activePrayersTable := viper.GetString(ActivePrayersTableConfigPath)

	if queue {
		return queuedPrayersTable
	} else {
		return activePrayersTable
	}
}

func IsPrayerActive(ctx context.Context, ddbClnt db.DDBConnecter, phone string) (bool, error) {
	pryr := Prayer{IntercessorPhone: phone}
	if err := pryr.Get(ctx, ddbClnt, false); err != nil {
		return false, utility.WrapError(err, "failed to check if Prayer is active")
	}

	// Empty string means get Prayer did not return an active Prayer. Dynamodb get requests return empty data if the key
	// does not exist inside the database.
	if pryr.Request == "" {
		return false, nil
	}

	return true, nil
}
