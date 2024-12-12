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

func (m *Member) sendMessage(body string) {
	sendText(body, m.Phone)
}

func (m *Member) get(clnt DDBClient) {
	resp := getItem(clnt, memberAttribute, m.Phone, memberTable)

	if err := attributevalue.UnmarshalMap(resp.Item, &m); err != nil {
		log.Fatalf("unmarshal failed for get member, %v", err)
	}
}

func (m *Member) put(clnt DDBClient) {
	data, err := attributevalue.MarshalMap(m)
	if err != nil {
		log.Fatalf("unmarshal failed for put member, %v", err)
	}

	putItem(clnt, memberTable, data)
}

func (m *Member) delete(clnt DDBClient) {
	delItem(clnt, memberAttribute, m.Phone, memberTable)
}
