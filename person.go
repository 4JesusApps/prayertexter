package main

import (
	"log"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

type Person struct {
	Name        string
	Phone       string
	PrayerCount	int
	PrayerLimit int
	SetupStage  int
	SetupStatus string
}

const (
	personAttribute = "Phone"
)

func (p Person) sendMessage(body string) {
	sendText(body, p.Phone)
}

func (p Person) get(table string) Person {
	resp := getItem(personAttribute, p.Phone, table)

	err := attributevalue.UnmarshalMap(resp.Item, &p)
	if err != nil {
		log.Fatalf("unmarshal failed for get person, %v", err)
	}

	return p
}

func (p Person) put(table string) {
	data, err := attributevalue.MarshalMap(p)
	if err != nil {
		log.Fatalf("unmarshal failed for put person, %v", err)
	}

	putItem(table, data)
}

func (p Person) delete() {
	tables := []string{"Members", "Intercessors"}

	for _, table := range tables {
		delItem(personAttribute, p.Phone, table)
	}
}
