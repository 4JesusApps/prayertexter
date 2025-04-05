package object

import (
	"log/slog"
	"math/rand/v2"
	"slices"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

type IntercessorPhones struct {
	Key    string
	Phones []string
}

const (
	DefaultIntercessorPhonesTable    = "General"
	intercessorPhonesTableConfigPath = "conf.aws.db.intercessorphones.table"

	IntercessorPhonesAttribute = "Key"
	IntercessorPhonesKey       = "IntercessorPhones"
)

func (i *IntercessorPhones) Get(ddbClnt db.DDBConnecter) error {
	table := viper.GetString(intercessorPhonesTableConfigPath)
	intr, err := db.GetDdbObject[IntercessorPhones](ddbClnt, IntercessorPhonesAttribute, IntercessorPhonesKey, table)

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

func (i *IntercessorPhones) Put(ddbClnt db.DDBConnecter) error {
	table := viper.GetString(intercessorPhonesTableConfigPath)
	i.Key = IntercessorPhonesKey

	return db.PutDdbObject(ddbClnt, table, i)
}

func (i *IntercessorPhones) AddPhone(phone string) {
	i.Phones = append(i.Phones, phone)
}

func (i *IntercessorPhones) RemovePhone(phone string) {
	utility.RemoveItem(&i.Phones, phone)
}

func (i *IntercessorPhones) GenRandPhones() []string {
	var selectedPhones []string

	if len(i.Phones) == 0 {
		slog.Warn("unable to generate phones; phone list is empty")
		return nil
	}

	intercessorsPerPrayer := viper.GetInt(IntercessorsPerPrayerConfigPath)

	// This is needed so it can return some/one phones even if it is less than the set # of intercessors for each
	// prayer.
	if len(i.Phones) <= intercessorsPerPrayer {
		selectedPhones = append(selectedPhones, i.Phones...)
		return selectedPhones
	}

	for len(selectedPhones) < intercessorsPerPrayer {
		phone := i.Phones[rand.IntN(len(i.Phones))] // # - false positive
		if slices.Contains(selectedPhones, phone) {
			continue
		}
		selectedPhones = append(selectedPhones, phone)
	}

	return selectedPhones
}
