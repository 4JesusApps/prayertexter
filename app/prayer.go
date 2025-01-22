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
	activePrayersAttribute = "IntercessorPhone"
	activePrayersTable     = "ActivePrayers"
	prayersQueueAttribute  = "RequestorPhone"
	prayersQueueTable      = "PrayersQueue"
)

func (p *Prayer) get(clnt DDBConnecter, queue bool) error {
	table := getPrayerTable(queue)
	resp, err := getItem(clnt, activePrayersAttribute, p.IntercessorPhone, table)
	if err != nil {
		return err
	}

	if err := attributevalue.UnmarshalMap(resp.Item, &p); err != nil {
		slog.Error("unmarshal failed for get Prayer")
		return err
	}

	return nil
}

func (p *Prayer) delete(clnt DDBConnecter, queue bool) error {
	table := getPrayerTable(queue)
	if err := delItem(clnt, activePrayersAttribute, p.IntercessorPhone, table); err != nil {
		return err
	}

	return nil
}

func (p *Prayer) put(clnt DDBConnecter, queue bool) error {
	// queue is only used if there are not enough intercessors available to take a prayer request
	// prayers get queued in order to save them for a time when intercessors are available
	// this will change the ddb table that the prayer is saved to
	data, err := attributevalue.MarshalMap(p)
	if err != nil {
		slog.Error("marshal failed for put Prayer")
		return err
	}

	table := getPrayerTable(queue)
	if err := putItem(clnt, table, data); err != nil {
		return err
	}

	return nil
}

func getPrayerTable(queue bool) string {
	var table string
	if queue {
		table = prayersQueueTable
	} else {
		table = activePrayersTable
	}

	return table
}
