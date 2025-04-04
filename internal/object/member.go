package object

import (
	"github.com/mshort55/prayertexter/internal/db"
	"github.com/mshort55/prayertexter/internal/messaging"
	"github.com/mshort55/prayertexter/internal/utility"
	"github.com/spf13/viper"
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
	DefaultMemberTable    = "Members"
	memberTableConfigPath = "conf.aws.db.member.table"
	
	MemberAttribute       = "Phone"

	MemberSetupInProgress = "IN PROGRESS"
	MemberSetupComplete   = "COMPLETE"

	MemberSignUpStepOne   = 1
	MemberSignUpStepTwo   = 2
	MemberSignUpStepThree = 3
	MemberSignUpStepFinal = 99
)

func (m *Member) Get(ddbClnt db.DDBConnecter) error {
	table := viper.GetString(memberTableConfigPath)
	mem, err := db.GetDdbObject[Member](ddbClnt, MemberAttribute, m.Phone, table)
	if err != nil {
		return err
	}

	// This is important so that the original Member object doesn't get reset to all empty struct values if the Member
	// does not exist in dynamodb.
	if mem.Phone != "" {
		*m = *mem
	}

	return nil
}

func (m *Member) Put(ddbClnt db.DDBConnecter) error {
	table := viper.GetString(memberTableConfigPath)
	return db.PutDdbObject(ddbClnt, table, m)
}

func (m *Member) Delete(ddbClnt db.DDBConnecter) error {
	table := viper.GetString(memberTableConfigPath)
	return db.DelDdbItem(ddbClnt, MemberAttribute, m.Phone, table)
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
		return false, utility.WrapError(err, "failed to check if Member is active")
	}

	// Empty string means get Member did not return an Member. Dynamodb get requests return empty data if the key does
	// not exist inside the database.
	if mem.SetupStatus == "" {
		return false, nil
	}

	return true, nil
}
