package object

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"

	"github.com/mshort55/prayertexter/internal/db"
)

type IntercessorPhones struct {
	Key    string
	Phones []string
}

const (
	IntercessorPhonesAttribute = "Key"
	IntercessorPhonesKey       = "IntercessorPhones"
	IntercessorPhonesTable     = "General"
	NumIntercessorsPerPrayer   = 2
)

func (i *IntercessorPhones) Get(ddbClnt db.DDBConnecter) error {
	intr, err := db.GetDdbObject[IntercessorPhones](ddbClnt, IntercessorPhonesAttribute,
		IntercessorPhonesKey, IntercessorPhonesTable)
	if err != nil {
		return fmt.Errorf("failed to get IntercessorPhones: %w", err)
	}

	// this is important so that the original IntercessorPhones object doesn't get reset to all
	// empty struct values if the IntercessorPhones does not exist in ddb
	if intr.Key != "" {
		*i = *intr
	}

	return nil
}

func (i *IntercessorPhones) Put(ddbClnt db.DDBConnecter) error {
	i.Key = IntercessorPhonesKey
	if err := db.PutDdbObject(ddbClnt, IntercessorPhonesTable, i); err != nil {
		return fmt.Errorf("failed to put IntercessorPhones: %w", err)
	}

	return nil
}

func (i *IntercessorPhones) AddPhone(phone string) {
	i.Phones = append(i.Phones, phone)
}

func (i *IntercessorPhones) RemovePhone(phone string) {
	var newPhones []string

	for _, p := range i.Phones {
		if p != phone {
			newPhones = append(newPhones, p)
		}
	}

	i.Phones = newPhones
}

func (i *IntercessorPhones) GenRandPhones() []string {
	var selectedPhones []string

	if len(i.Phones) == 0 {
		slog.Warn("unable to generate phones; phone list is empty")
		return nil
	}

	// this is needed so it can return some/one phones even if it is less than the set # of
	// intercessors for each prayer
	if len(i.Phones) <= NumIntercessorsPerPrayer {
		selectedPhones = append(selectedPhones, i.Phones...)
		return selectedPhones
	}

	for len(selectedPhones) < NumIntercessorsPerPrayer {
		phone := i.Phones[rand.IntN(len(i.Phones))] // #nosec G404 - false positive
		if slices.Contains(selectedPhones, phone) {
			continue
		}
		selectedPhones = append(selectedPhones, phone)
	}

	return selectedPhones
}
