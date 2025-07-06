package object

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"slices"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

// IntercessorPhones contains all of the phone numbers of active intercessors. This is kept separately to allow for
// quick lookups of all intercessor phone numbers at once, as opposed to looping through all Member objects, which would
// also add a lot of extra dynamodb get calls.
type IntercessorPhones struct {
	// Key is the dynamodb table key name used for dynamodb operations.
	Key string
	// Phones contains a slice of intercessor phone numbers.
	Phones []string
}

// Default values for configuration that has been exposed to be used with the config package.
const (
	DefaultIntercessorPhonesTable    = "General"
	IntercessorPhonesTableConfigPath = "conf.aws.db.intercessorphones.table"
)

// IntercessorPhones object key/value used to interact with dynamodb tables.
const (
	IntercessorPhonesKey      = "Key"
	IntercessorPhonesKeyValue = "IntercessorPhones"
)

// Get gets IntercessorPhones from dynamodb. If it does not exist, the current instance of IntercessorPhones will not
// change.
func (i *IntercessorPhones) Get(ctx context.Context, ddbClnt db.DDBConnecter) error {
	table := viper.GetString(IntercessorPhonesTableConfigPath)
	intr, err := db.GetDdbObject[IntercessorPhones](ctx, ddbClnt, IntercessorPhonesKey, IntercessorPhonesKeyValue,
		table)

	if err != nil {
		return err
	}

	// This is important so that the original IntercessorPhones object doesn't get reset to all empty struct values if
	// the IntercessorPhones does not exist in dynamodb.
	if intr.Key != "" {
		*i = *intr
	}

	return nil
}

// Put saves IntercessorPhones to dynamodb.
func (i *IntercessorPhones) Put(ctx context.Context, ddbClnt db.DDBConnecter) error {
	table := viper.GetString(IntercessorPhonesTableConfigPath)
	i.Key = IntercessorPhonesKeyValue

	return db.PutDdbObject(ctx, ddbClnt, table, i)
}

// AddPhone adds a phone number string to IntercessorPhones. If phone already exists, it will not add a duplicate.
func (i *IntercessorPhones) AddPhone(phone string) {
	if slices.Contains(i.Phones, phone) {
		return
	}

	i.Phones = append(i.Phones, phone)
}

// RemovePhone removes a phone number string from IntercessorPhones.
func (i *IntercessorPhones) RemovePhone(phone string) {
	utility.RemoveItem(&i.Phones, phone)
}

// GenRandPhones will return a string slice of individual intercessor phone numbers from phone numbers in
// IntercessorPhones. The number of intercessor phones returned is a configurable number.
func (i *IntercessorPhones) GenRandPhones() []string {
	var selectedPhones []string

	if len(i.Phones) == 0 {
		slog.Warn("unable to generate phones, phone list is empty")
		return nil
	}

	intercessorsPerPrayer := viper.GetInt(IntercessorsPerPrayerConfigPath)

	// This is needed so it can return some/one phones even if they are less than the desired number of intercessors per
	// prayer.
	if len(i.Phones) <= intercessorsPerPrayer {
		selectedPhones = append(selectedPhones, i.Phones...)
		return selectedPhones
	}

	for len(selectedPhones) < intercessorsPerPrayer {
		phone := i.Phones[rand.IntN(len(i.Phones))] //nolint:gosec // this is a false positive
		if slices.Contains(selectedPhones, phone) {
			continue
		}
		selectedPhones = append(selectedPhones, phone)
	}

	return selectedPhones
}
