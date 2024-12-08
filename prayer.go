package main

import (
	"log"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

type Prayer struct {
	Intercessor      Member
	IntercessorPhone string
	Request          string
	Requestor        Member
}

const (
	prayerAttribute = "IntercessorPhone"
	prayerTable     = "ActivePrayers"
)

func (p Prayer) get() Prayer {
	resp := getItem(prayerAttribute, p.IntercessorPhone, prayerTable)

	if err := attributevalue.UnmarshalMap(resp.Item, &p); err != nil {
		log.Fatalf("unmarshal failed for get prayer, %v", err)
	}

	return p
}

func (p Prayer) delete() {
	delItem(prayerAttribute, p.IntercessorPhone, prayerTable)
}

func (p Prayer) put() {
	data, err := attributevalue.MarshalMap(p)
	if err != nil {
		log.Fatalf("unmarshal failed for put prayer, %v", err)
	}

	putItem(prayerTable, data)
}
