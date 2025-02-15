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
	mem, err := getDdbObject[Member](clnt, memberAttribute, m.Phone, memberTable)
	if err != nil {
		return err
	}

	// this is important so that the original member object doesn't get reset to all empty struct 
	// values if the member does not exist in ddb
	if mem.Phone != "" {
		*m = *mem
	}
	
	return nil
}

func (m *Member) put(clnt DDBConnecter) error {
	data, err := attributevalue.MarshalMap(m)
	if err != nil {
		slog.Error("marshal failed for put Member")
		return err
	}

	if err := putDdbItem(clnt, memberTable, data); err != nil {
		return err
	}

	return nil
}

func (m *Member) delete(clnt DDBConnecter) error {
	if err := delDdbItem(clnt, memberAttribute, m.Phone, memberTable); err != nil {
		return err
	}

	return nil
}

func (m *Member) sendMessage(clnt DDBConnecter, sndr TextSender, body string) error {
	body = msgPre + body + "\n\n" + msgPost
	message := TextMessage{
		Body:  body,
		Phone: m.Phone,
	}

	return sndr.sendText(clnt, message)
}

func (m *Member) checkIfActive(clnt DDBConnecter) (bool, error) {
	if err := m.get(clnt); err != nil {
		return false, err
	}

	// empty string means get did not return a member
	if m.SetupStatus == "" {
		return false, nil
	} else {
		return true, nil
	}
}
