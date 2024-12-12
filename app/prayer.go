package prayertexter

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

func (p *Prayer) get(clnt DDBClient) {
	resp := getItem(clnt, prayerAttribute, p.IntercessorPhone, prayerTable)

	if err := attributevalue.UnmarshalMap(resp.Item, &p); err != nil {
		log.Fatalf("unmarshal failed for get prayer, %v", err)
	}
}

func (p *Prayer) delete(clnt DDBClient) {
	delItem(clnt, prayerAttribute, p.IntercessorPhone, prayerTable)
}

func (p *Prayer) put(clnt DDBClient) {
	data, err := attributevalue.MarshalMap(p)
	if err != nil {
		log.Fatalf("unmarshal failed for put prayer, %v", err)
	}

	putItem(clnt, prayerTable, data)
}
