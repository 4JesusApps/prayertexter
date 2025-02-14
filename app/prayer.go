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

func (pryr *Prayer) get(clnt DDBConnecter, queue bool) error {
	// queue determines whether ActivePrayers or PrayersQueue table is used for get
	table := getPrayerTable(queue)
	resp, err := getItem(clnt, activePrayersAttribute, pryr.IntercessorPhone, table)
	if err != nil {
		return err
	}

	if err := attributevalue.UnmarshalMap(resp.Item, &pryr); err != nil {
		slog.Error("unmarshal failed for get Prayer")
		return err
	}

	return nil
}

func (pryr *Prayer) delete(clnt DDBConnecter, queue bool) error {
	table := getPrayerTable(queue)
	if err := delItem(clnt, activePrayersAttribute, pryr.IntercessorPhone, table); err != nil {
		return err
	}

	return nil
}

func (pryr *Prayer) put(clnt DDBConnecter, queue bool) error {
	// queue is only used if there are not enough intercessors available to take a prayer request
	// prayers get queued in order to save them for a time when intercessors are available
	// this will change the ddb table that the prayer is saved to
	data, err := attributevalue.MarshalMap(pryr)
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

func (pryr *Prayer) checkIfActive(clnt DDBConnecter) (bool, error) {
	if err := pryr.get(clnt, false); err != nil {
		return false, err
	}

	// empty string means get did not return a prayer
	if pryr.Request == "" {
		return false, nil
	} else {
		return true, nil
	}
}
