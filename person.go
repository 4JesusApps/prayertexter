package main

import (
	"log"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

type Person struct {
	Name        string
	Phone       string
	PrayerLimit string
	SetupStage  string
	SetupStatus string
}

const (
	personAttribute = "Phone"
)

func (per Person) sendMessage(body string) {
	sendText(body, per.Phone)
}

func (per Person) get(table string) Person {
	resp := getItem(personAttribute, per.Phone, table)

	err := attributevalue.UnmarshalMap(resp.Item, &per)
	if err != nil {
		log.Fatalf("unmarshal failed for get person, %v", err)
	}

	return per
}

func (per Person) put(table string) {
	data, err := attributevalue.MarshalMap(per)
	if err != nil {
		log.Fatalf("unmarshal failed, for put person, %v", err)
	}

	putItem(table, data)
}

func (per Person) delete() {
	tables := []string{"Members", "Intercessors"}

	for _, table := range tables {
		delItem(personAttribute, per.Phone, table)
	}
}
