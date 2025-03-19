package object

import (
	"github.com/mshort55/prayertexter/internal/db"
	"github.com/mshort55/prayertexter/internal/messaging"
	"github.com/mshort55/prayertexter/internal/utility"
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

	MemberSetupInProgress = "IN PROGRESS"
	MemberSetupComplete   = "COMPLETE"

	MemberSignUpStepOne   = 1
	MemberSignUpStepTwo   = 2
	MemberSignUpStepThree = 3
	MemberSignUpStepFinal = 99
)

func (m *Member) Get(ddbClnt db.DDBConnecter) error {
	mem, err := db.GetDdbObject[Member](ddbClnt, MemberAttribute, m.Phone, MemberTable)
	if err != nil {
		return err
	}

	// this is important so that the original Member object doesn't get reset to all empty struct
	// values if the Member does not exist in ddb
	if mem.Phone != "" {
		*m = *mem
	}

	return nil
}

func (m *Member) Put(ddbClnt db.DDBConnecter) error {
	return db.PutDdbObject(ddbClnt, MemberTable, m)
}

func (m *Member) Delete(ddbClnt db.DDBConnecter) error {
	return db.DelDdbItem(ddbClnt, MemberAttribute, m.Phone, MemberTable)
}

func (m *Member) SendMessage(smsClnt messaging.TextSender, body string) error {
	message := messaging.TextMessage{
		Body:  body,
		Phone: m.Phone,
	}

	return messaging.SendText(smsClnt, message)
}

func IsMemberActive(ddbClnt db.DDBConnecter, phone string) (bool, error) {
	mem := Member{Phone: phone}
	if err := mem.Get(ddbClnt); err != nil {
		return *new(bool), utility.WrapError(err, "failed to check if Member is active")
	}

	// empty string means get Member did not return an Member. Dynamodb get requests
	// return empty data if the key does not exist inside the database
	if mem.SetupStatus == "" {
		return false, nil
	}

	return true, nil
}
