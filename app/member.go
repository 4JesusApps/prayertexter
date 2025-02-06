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

func (mem *Member) get(clnt DDBConnecter) error {
	resp, err := getItem(clnt, memberAttribute, mem.Phone, memberTable)
	if err != nil {
		return err
	}

	if err := attributevalue.UnmarshalMap(resp.Item, &mem); err != nil {
		slog.Error("unmarshal failed for get member")
		return err
	}

	return nil
}

func (mem *Member) put(clnt DDBConnecter) error {
	data, err := attributevalue.MarshalMap(mem)
	if err != nil {
		slog.Error("marshal failed for put Member")
		return err
	}

	if err := putItem(clnt, memberTable, data); err != nil {
		return err
	}

	return nil
}

func (mem *Member) delete(clnt DDBConnecter) error {
	if err := delItem(clnt, memberAttribute, mem.Phone, memberTable); err != nil {
		return err
	}

	return nil
}

func (mem *Member) sendMessage(sndr TextSender, body string) error {
	body = msgPre + body + "\n\n" + msgPost
	message := TextMessage{
		Body:  body,
		Phone: mem.Phone,
	}
	return sndr.sendText(message)
}
