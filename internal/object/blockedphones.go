package object

import (
	"context"
	"slices"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

// BlockedPhones contains all of the phone numbers that are blocked from sending or receiving messages.
type BlockedPhones struct {
	// Key is the dynamodb table key name used for dynamodb operations.
	Key string
	// Phones contains a slice of blocked phone numbers.
	Phones []string
}

// Default values for configuration that has been exposed to be used with the config package.
const (
	DefaultBlockedPhonesTable    = "General"
	BlockedPhonesTableConfigPath = "conf.aws.db.blockedphones.table"
)

// BlockedPhones object key/value used to interact with dynamodb tables.
const (
	BlockedPhonesKey      = "Key"
	BlockedPhonesKeyValue = "BlockedPhones"
)

// Get gets BlockedPhones from dynamodb. If it does not exist, the current instance of BlockedPhones will not
// change.
func (b *BlockedPhones) Get(ctx context.Context, ddbClnt db.DDBConnecter) error {
	table := viper.GetString(BlockedPhonesTableConfigPath)
	blocked, err := db.GetDdbObject[BlockedPhones](ctx, ddbClnt, BlockedPhonesKey, BlockedPhonesKeyValue,
		table)

	if err != nil {
		return err
	}

	// This is important so that the original BlockedPhones object doesn't get reset to all empty struct values if
	// the BlockedPhones does not exist in dynamodb.
	if blocked.Key != "" {
		*b = *blocked
	}

	return nil
}

// Put saves BlockedPhones to dynamodb.
func (b *BlockedPhones) Put(ctx context.Context, ddbClnt db.DDBConnecter) error {
	table := viper.GetString(BlockedPhonesTableConfigPath)
	b.Key = BlockedPhonesKeyValue

	return db.PutDdbObject(ctx, ddbClnt, table, b)
}

// AddPhone adds a phone number string to BlockedPhones. If phone already exists, it will not add a duplicate.
func (b *BlockedPhones) AddPhone(phone string) {
	if slices.Contains(b.Phones, phone) {
		return
	}

	b.Phones = append(b.Phones, phone)
}

// RemovePhone removes a phone number string from BlockedPhones.
func (b *BlockedPhones) RemovePhone(phone string) {
	utility.RemoveItem(&b.Phones, phone)
}
