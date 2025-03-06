package prayertexter

import (
	"log/slog"
	"math/rand/v2"
	"slices"
)

type IntercessorPhones struct {
	Key    string
	Phones []string
}

const (
	intercessorPhonesAttribute = "Key"
	intercessorPhonesKey       = "IntercessorPhones"
	intercessorPhonesTable     = "General"
	numIntercessorsPerPrayer   = 2
)

func (i *IntercessorPhones) get(ddbClnt DDBConnecter) error {
	intr, err := getDdbObject[IntercessorPhones](ddbClnt, intercessorPhonesAttribute, intercessorPhonesKey, intercessorPhonesTable)
	if err != nil {
		return err
	}

	// this is important so that the original IntercessorPhones object doesn't get reset to all 
	// empty struct values if the IntercessorPhones does not exist in ddb
	if intr.Key != "" {
		*i = *intr
	}

	return nil
}

func (i *IntercessorPhones) put(ddbClnt DDBConnecter) error {
	i.Key = intercessorPhonesKey
	return putDdbObject(ddbClnt, intercessorPhonesTable, i)
}

func (i *IntercessorPhones) addPhone(phone string) {
	i.Phones = append(i.Phones, phone)
}

func (i *IntercessorPhones) removePhone(phone string) {
	var newPhones []string

	for _, p := range i.Phones {
		if p != phone {
			newPhones = append(newPhones, p)
		}
	}

	i.Phones = newPhones
}

func (i *IntercessorPhones) genRandPhones() []string {
	var selectedPhones []string

	if len(i.Phones) == 0 {
		slog.Warn("unable to generate phones; phone list is empty")
		return nil
	}

	// this is needed so it can return some/one phones even if it is less than the set # of
	// intercessors for each prayer
	if len(i.Phones) <= numIntercessorsPerPrayer {
		selectedPhones = append(selectedPhones, i.Phones...)
		return selectedPhones
	}

	for len(selectedPhones) < numIntercessorsPerPrayer {
		phone := i.Phones[rand.IntN(len(i.Phones))]
		if slices.Contains(selectedPhones, phone) {
			continue
		}
		selectedPhones = append(selectedPhones, phone)
	}

	return selectedPhones
}
