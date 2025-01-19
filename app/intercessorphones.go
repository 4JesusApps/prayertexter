package prayertexter

import (
	"log/slog"
	"math/rand"
	"slices"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

type IntercessorPhones struct {
	Name   string
	Phones []string
}

const (
	intercessorPhonesAttribute = "Name"
	intercessorPhonesKey       = "IntercessorPhones"
	intercessorPhonesTable     = "General"
	numIntercessorsPerPrayer   = 2
)

func (i *IntercessorPhones) get(clnt DDBConnecter) error {
	resp, err := getItem(clnt, intercessorPhonesAttribute, intercessorPhonesKey,
		intercessorPhonesTable)
	if err != nil {
		return err
	}

	if err := attributevalue.UnmarshalMap(resp.Item, &i); err != nil {
		slog.Error("unmarshal failed for get IntercessorPhones")
		return err
	}

	return nil
}

func (i *IntercessorPhones) put(clnt DDBConnecter) error {
	i.Name = intercessorPhonesKey

	data, err := attributevalue.MarshalMap(i)
	if err != nil {
		slog.Error("marshal failed for put IntercessorPhones")
		return err
	}

	if err := putItem(clnt, intercessorPhonesTable, data); err != nil {
		return err
	}

	return nil
}

func (i *IntercessorPhones) addPhone(phone string) {
	i.Phones = append(i.Phones, phone)
}

func (i *IntercessorPhones) delPhone(phone string) {
	var newPhones []string

	for _, p := range i.Phones {
		if p != phone {
			newPhones = append(newPhones, p)
		}
	}

	i.Phones = newPhones
}

func (i *IntercessorPhones) genRandPhones() ([]string) {
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
		phone := i.Phones[rand.Intn(len(i.Phones))]
		if slices.Contains(selectedPhones, phone) {
			continue
		}
		selectedPhones = append(selectedPhones, phone)
	}

	return selectedPhones
}
