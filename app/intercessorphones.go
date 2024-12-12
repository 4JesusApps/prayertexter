package prayertexter

import (
	"log"
	"math/rand"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

type IntercessorPhones struct {
	Name   string
	Phones []string
}

const (
	IntercessorPhonesAttribute = "Name"
	IntercessorPhonesKey       = "IntercessorPhones"
	IntercessorPhonesTable     = "General"
	numIntercessorsPerPrayer   = 2
)

func (i *IntercessorPhones) get(clnt DDBClient) {
	resp := getItem(clnt, IntercessorPhonesAttribute, IntercessorPhonesKey, IntercessorPhonesTable)

	if err := attributevalue.UnmarshalMap(resp.Item, &i); err != nil {
		log.Fatalf("unmarshal failed for get intercessor phones, %v", err)
	}
}

func (i *IntercessorPhones) put(clnt DDBClient) {
	i.Name = IntercessorPhonesKey

	data, err := attributevalue.MarshalMap(i)
	if err != nil {
		log.Fatalf("marshal failed for put intercessor phones, %v", err)
	}

	putItem(clnt, IntercessorPhonesTable, data)
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
