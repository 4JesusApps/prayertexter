package object

import (
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
	activePrayersTableConfigPath = "conf.aws.db.prayer.activetable"

	DefaultQueuedPrayersTable    = "QueuedPrayers"
	queuedPrayersTableConfigPath = "conf.aws.db.prayer.queuetable"

	DefaultIntercessorsPerPrayer    = 2
	IntercessorsPerPrayerConfigPath = "conf.intercessorsperprayer"

	PrayersAttribute = "IntercessorPhone"
)

func (p *Prayer) Get(ddbClnt db.DDBConnecter, queue bool) error {
	// Queue determines whether ActivePrayers or PrayersQueue table is used for dynamodb get requests.
	table := GetPrayerTable(queue)
	pryr, err := db.GetDdbObject[Prayer](ddbClnt, PrayersAttribute, p.IntercessorPhone, table)
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

func (p *Prayer) Put(ddbClnt db.DDBConnecter, queue bool) error {
	// Prayers get queued in order to save them for a time when intercessors are available. This will change the
	// dynamodb table that the prayer is saved to.
	table := GetPrayerTable(queue)

	return db.PutDdbObject(ddbClnt, table, p)
}

func (p *Prayer) Delete(ddbClnt db.DDBConnecter, queue bool) error {
	table := GetPrayerTable(queue)

	return db.DelDdbItem(ddbClnt, PrayersAttribute, p.IntercessorPhone, table)
}

func GetPrayerTable(queue bool) string {
	queuedPrayersTable := viper.GetString(queuedPrayersTableConfigPath)
	activePrayersTable := viper.GetString(activePrayersTableConfigPath)

	if queue {
		return queuedPrayersTable
	} else {
		return activePrayersTable
	}
}

func IsPrayerActive(ddbClnt db.DDBConnecter, phone string) (bool, error) {
	pryr := Prayer{IntercessorPhone: phone}
	if err := pryr.Get(ddbClnt, false); err != nil {
		return false, utility.WrapError(err, "failed to check if Prayer is active")
	}

	// Empty string means get Prayer did not return an active Prayer. Dynamodb get requests return empty data if the key
	// does not exist inside the database.
	if pryr.Request == "" {
		return false, nil
	}

	return true, nil
}
