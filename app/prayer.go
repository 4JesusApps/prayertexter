package prayertexter

import (
	"log/slog"

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

func (p *Prayer) get(clnt DDBConnecter) error {
	resp, err := getItem(clnt, prayerAttribute, p.IntercessorPhone, prayerTable)
	if err != nil {
		return err
	}

	if err := attributevalue.UnmarshalMap(resp.Item, &p); err != nil {
		slog.Error("unmarshal failed for get Prayer")
		return err
	}

	return nil
}

func (p *Prayer) delete(clnt DDBConnecter) error {
	if err := delItem(clnt, prayerAttribute, p.IntercessorPhone, prayerTable); err != nil {
		return err
	}

	return nil
}

func (p *Prayer) put(clnt DDBConnecter) error {
	data, err := attributevalue.MarshalMap(p)
	if err != nil {
		slog.Error("marshal failed for put Prayer")
		return err
	}

	if err := putItem(clnt, prayerTable, data); err != nil {
		return err
	}

	return nil
}
