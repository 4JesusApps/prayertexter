package object

import (
	"context"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

// A Prayer represents a prayer request.
type Prayer struct {
	// Intercessor is the current intercessor member assigned to this prayer.
	Intercessor Member
	// IntercessorPhone is the phone number of the currently assigned intercessor.
	IntercessorPhone string
	// Request is the prayer request content.
	Request string
	// Requestor is the member who sent in the prayer request.
	Requestor Member
}

// Default values for configuration that has been exposed to be used with the config package.
const (
	DefaultActivePrayersTable    = "ActivePrayer"
	ActivePrayersTableConfigPath = "conf.aws.db.prayer.activetable"

	DefaultQueuedPrayersTable    = "QueuedPrayer"
	QueuedPrayersTableConfigPath = "conf.aws.db.prayer.queuetable"

	DefaultIntercessorsPerPrayer    = 2
	IntercessorsPerPrayerConfigPath = "conf.intercessorsperprayer"
)

// PrayerKey is the Prayer object key used to interact with dynamodb tables.
const PrayerKey = "IntercessorPhone"

// Get gets a Prayer from dynamodb. If it does not exist, the current instance of Prayer will not change.
func (p *Prayer) Get(ctx context.Context, ddbClnt db.DDBConnecter, queue bool) error {
	// Queue determines whether ActivePrayers or PrayersQueue table is used for dynamodb get requests.
	table := GetPrayerTable(queue)
	pryr, err := db.GetDdbObject[Prayer](ctx, ddbClnt, PrayerKey, p.IntercessorPhone, table)
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

// Put saves a Prayer to dynamodb.
func (p *Prayer) Put(ctx context.Context, ddbClnt db.DDBConnecter, queue bool) error {
	// Prayers can get queued in order to save them for a time when intercessors are available. This will change the
	// dynamodb table that the prayer is saved to.
	table := GetPrayerTable(queue)

	return db.PutDdbObject(ctx, ddbClnt, table, p)
}

// Delete deletes a Prayer from dynamodb. If it does not exist, it will not return an error.
func (p *Prayer) Delete(ctx context.Context, ddbClnt db.DDBConnecter, queue bool) error {
	table := GetPrayerTable(queue)

	return db.DelDdbItem(ctx, ddbClnt, PrayerKey, p.IntercessorPhone, table)
}

// GetPrayerTable returns either the active or queued prayer table depending on the parameter queue.
func GetPrayerTable(queue bool) string {
	queuedPrayersTable := viper.GetString(QueuedPrayersTableConfigPath)
	activePrayersTable := viper.GetString(ActivePrayersTableConfigPath)

	if queue {
		return queuedPrayersTable
	}

	return activePrayersTable
}

// IsPrayerActive reports whether a Prayer is found (active) in dynamodb.
func IsPrayerActive(ctx context.Context, ddbClnt db.DDBConnecter, phone string) (bool, error) {
	pryr := Prayer{IntercessorPhone: phone}
	if err := pryr.Get(ctx, ddbClnt, false); err != nil {
		return false, utility.WrapError(err, "failed to check if Prayer is active")
	}

	// Empty string means get Prayer did not return an active Prayer. Dynamodb get requests return empty data if the key
	// does not exist inside the table.
	if pryr.Request == "" {
		return false, nil
	}

	return true, nil
}
