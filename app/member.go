package prayertexter

import (
	"log"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

type Member struct {
	Intercessor       bool
	Name              string
	Phone             string
	PrayerCount       int
	SetupStage        int
	SetupStatus       string
	WeeklyPrayerDate  string
	WeeklyPrayerLimit int
}

const (
	memberAttribute = "Phone"
	memberTable     = "Members"
)

func sendText(body string, recipient string) {
	log.Printf("Sending to: %v\n", recipient)
	log.Printf("Body: %v\n", body)
}

func (m Member) sendMessage(body string) {
	sendText(body, m.Phone)
}

func (m Member) get() Member {
	resp := getItem(memberAttribute, m.Phone, memberTable)

	if err := attributevalue.UnmarshalMap(resp.Item, &m); err != nil {
		log.Fatalf("unmarshal failed for get member, %v", err)
	}

	return m
}

func (m Member) put() {
	data, err := attributevalue.MarshalMap(m)
	if err != nil {
		log.Fatalf("unmarshal failed for put member, %v", err)
	}

	putItem(memberTable, data)
}

func (m Member) delete() {
	delItem(memberAttribute, m.Phone, memberTable)
}
