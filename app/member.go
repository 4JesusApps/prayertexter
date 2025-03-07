package prayertexter

import (
	"fmt"
	"log/slog"
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

func (m *Member) get(ddbClnt DDBConnecter) error {
	mem, err := getDdbObject[Member](ddbClnt, memberAttribute, m.Phone, memberTable)
	if err != nil {
		return fmt.Errorf("Member get: %w", err)
	}

	// this is important so that the original Member object doesn't get reset to all empty struct
	// values if the Member does not exist in ddb
	if mem.Phone != "" {
		*m = *mem
	}

	return nil
}

func (m *Member) put(ddbClnt DDBConnecter) error {
	if err := putDdbObject(ddbClnt, memberTable, m); err != nil {
		return fmt.Errorf("Member put: %w", err)
	}

	return nil
}

func (m *Member) delete(ddbClnt DDBConnecter) error {
	if err := delDdbItem(ddbClnt, memberAttribute, m.Phone, memberTable); err != nil {
		return fmt.Errorf("Member delete: %w", err)
	}

	return nil
}

func (m *Member) sendMessage(smsClnt TextSender, body string) error {
	message := TextMessage{
		Body:  body,
		Phone: m.Phone,
	}

	if err := sendText(smsClnt, message); err != nil {
		slog.Error("sendMessage failed", "recipient", m.Phone, "msg", body, "error", err)
		return fmt.Errorf("Member sendText: %w", err)
	}

	return nil
}

func isMemberActive(ddbClnt DDBConnecter, phone string) (bool, error) {
	mem := Member{Phone: phone}
	if err := mem.get(ddbClnt); err != nil {
		// returning false but it really should be nil due to error
		return false, fmt.Errorf("isMemberActive: %w", err)
	}

	// empty string means get Member did not return an Member. Dynamodb get requests
	// return empty data if the key does not exist inside the database
	if mem.SetupStatus == "" {
		return false, nil
	}

	return true, nil
}
