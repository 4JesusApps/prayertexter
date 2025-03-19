package object

import (
	"github.com/mshort55/prayertexter/internal/db"
	"github.com/mshort55/prayertexter/internal/utility"
)

type Prayer struct {
	Intercessor      Member
	IntercessorPhone string
	Request          string
	Requestor        Member
}

const (
	PrayersAttribute   = "IntercessorPhone"
	ActivePrayersTable = "ActivePrayers"
	QueuedPrayersTable = "QueuedPrayers"
)

func (p *Prayer) Get(ddbClnt db.DDBConnecter, queue bool) error {
	// queue determines whether ActivePrayers or PrayersQueue table is used for get
	table := GetPrayerTable(queue)
	pryr, err := db.GetDdbObject[Prayer](ddbClnt, PrayersAttribute, p.IntercessorPhone, table)
	if err != nil {
		return err
	}

	// this is important so that the original Prayer object doesn't get reset to all empty struct
	// values if the Prayer does not exist in ddb
	if pryr.IntercessorPhone != "" {
		*p = *pryr
	}

	return nil
}

func (p *Prayer) Put(ddbClnt db.DDBConnecter, queue bool) error {
	// queue is only used if there are not enough intercessors available to take a prayer request
	// prayers get queued in order to save them for a time when intercessors are available
	// this will change the ddb table that the prayer is saved to
	table := GetPrayerTable(queue)

	return db.PutDdbObject(ddbClnt, table, p)
}

func (p *Prayer) Delete(ddbClnt db.DDBConnecter, queue bool) error {
	table := GetPrayerTable(queue)

	return db.DelDdbItem(ddbClnt, PrayersAttribute, p.IntercessorPhone, table)
}

func GetPrayerTable(queue bool) string {
	var table string
	if queue {
		table = QueuedPrayersTable
	} else {
		table = ActivePrayersTable
	}

	return table
}

func IsPrayerActive(ddbClnt db.DDBConnecter, phone string) (bool, error) {
	pryr := Prayer{IntercessorPhone: phone}
	if err := pryr.Get(ddbClnt, false); err != nil {
		return *new(bool), utility.WrapError(err, "failed to check if Prayer is active")
	}

	// empty string means get Prayer did not return an active Prayer. Dynamodb get requests
	// return empty data if the key does not exist inside the database
	if pryr.Request == "" {
		return false, nil
	}

	return true, nil
}
