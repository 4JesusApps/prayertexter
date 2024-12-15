package prayertexter

import (
	"log/slog"

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

func (m *Member) get(clnt DDBConnecter) error {
	resp, err := getItem(clnt, memberAttribute, m.Phone, memberTable)
	if err != nil {
		slog.Error("get Member failed")
		return err
	}

	if err := attributevalue.UnmarshalMap(resp.Item, &m); err != nil {
		slog.Error("unmarshal failed for get member")
		return err
	}

	return nil
}

func (m *Member) put(clnt DDBConnecter) error {
	data, err := attributevalue.MarshalMap(m)
	if err != nil {
		slog.Error("marshal failed for put Member")
		return err
	}

	if err := putItem(clnt, memberTable, data); err != nil {
		slog.Error("put Member failed")
		return err
	}

	return nil
}

func (m *Member) delete(clnt DDBConnecter) error {
	if err := delItem(clnt, memberAttribute, m.Phone, memberTable); err != nil {
		slog.Error("delete Member failed")
		return err
	}

	return nil
}

func (m *Member) sendMessage(sndr TextSender, body string) error {
	message := TextMessage{
		Body:  body,
		Phone: m.Phone,
	}
	return sndr.SendText(message)
}
