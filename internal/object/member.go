package object

import (
	"fmt"
	"log/slog"

	"github.com/mshort55/prayertexter/internal/db"
	"github.com/mshort55/prayertexter/internal/messaging"
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
	MemberAttribute = "Phone"
	MemberTable     = "Members"
)

func (m *Member) Get(ddbClnt db.DDBConnecter) error {
	mem, err := db.GetDdbObject[Member](ddbClnt, MemberAttribute, m.Phone, MemberTable)
	if err != nil {
		return fmt.Errorf("failed to get Member: %w", err)
	}

	// this is important so that the original Member object doesn't get reset to all empty struct
	// values if the Member does not exist in ddb
	if mem.Phone != "" {
		*m = *mem
	}

	return nil
}

func (m *Member) Put(ddbClnt db.DDBConnecter) error {
	if err := db.PutDdbObject(ddbClnt, MemberTable, m); err != nil {
		return fmt.Errorf("failed to put Member: %w", err)
	}

	return nil
}

func (m *Member) Delete(ddbClnt db.DDBConnecter) error {
	if err := db.DelDdbItem(ddbClnt, MemberAttribute, m.Phone, MemberTable); err != nil {
		return fmt.Errorf("failed to delete Member: %w", err)
	}

	return nil
}

func (m *Member) SendMessage(smsClnt messaging.TextSender, body string) error {
	message := messaging.TextMessage{
		Body:  body,
		Phone: m.Phone,
	}

	if err := messaging.SendText(smsClnt, message); err != nil {
		slog.Error("sendMessage failed", "recipient", m.Phone, "msg", body, "error", err)
		return fmt.Errorf("failed to send text to Member: %w", err)
	}

	return nil
}

func IsMemberActive(ddbClnt db.DDBConnecter, phone string) (bool, error) {
	mem := Member{Phone: phone}
	if err := mem.Get(ddbClnt); err != nil {
		// returning false but it really should be nil due to error
		return false, fmt.Errorf("failed to check if Member is active: %w", err)
	}

	// empty string means get Member did not return an Member. Dynamodb get requests
	// return empty data if the key does not exist inside the database
	if mem.SetupStatus == "" {
		return false, nil
	}

	return true, nil
}
