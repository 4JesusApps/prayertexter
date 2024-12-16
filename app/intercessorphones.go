package prayertexter

import (
	"log/slog"
	"math/rand"

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
	resp, err := getItem(clnt, intercessorPhonesAttribute, intercessorPhonesKey, intercessorPhonesTable)
	if err != nil {
		slog.Error("get IntercessorPhones failed")
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
		slog.Error("put IntercessorPhones failed")
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

func (i *IntercessorPhones) genRandPhones() []string {
	var phones []string

	for len(phones) < numIntercessorsPerPrayer {
		p := i.Phones[rand.Intn(len(i.Phones))]
		phones = append(phones, p)
	}

	return phones
}
