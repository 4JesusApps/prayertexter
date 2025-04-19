package object

import (
	"context"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

// A Member represents someone who has signed up or is in the process of signing up for prayertexter.
type Member struct {
	// Intercessor shows whether the Member is an intercessor.
	Intercessor bool
	// Name is the Members name.
	Name string
	// Phone is the Members phone number.
	Phone string
	// PrayerCount is the current number of prayers that the Member has prayed for in a week long period.
	// This is only used if Member is an intercessor.
	PrayerCount int
	// SetupStage is the Members current setup stage that they are on.
	// SetupStage refers to the initial Member sign up process over text message.
	// This is used to track the Member sign up progress and is set to 99 when sign up is complete.
	// This is only used if Member is an intercessor.
	SetupStage int
	// SetupStatus describes whether the Member has started or completed the sign up process.
	SetupStatus string
	// WeeklyPrayerDate keeps track of the date in order to track weekly number of prayer requests.
	// This is used with PrayerCount to determine if an intercessor is able to receive a prayer request.
	// This is only used if Member is an intercessor.
	WeeklyPrayerDate string
	// WeeklyPrayerLimit is the max number of prayers that the Member has agreed to receive and pray for per week.
	// This is only used if Member is an intercessor.
	WeeklyPrayerLimit int
}

// Default values for configuration that has been exposed to be used with the config package.
const (
	DefaultMemberTable    = "Member"
	MemberTableConfigPath = "conf.aws.db.member.table"
)

// MemberKey is the Member object key used to interact with dynamodb tables.
const MemberKey = "Phone"

// Values used during the Member sign up process will be included here.
const (
	MemberSetupInProgress = "IN PROGRESS"
	MemberSetupComplete   = "COMPLETE"
	MemberSignUpStepOne   = 1
	MemberSignUpStepTwo   = 2
	MemberSignUpStepThree = 3
	MemberSignUpStepFinal = 99
)

// Get gets a Member from dynamodb. If it does not exist, the current instance of Member will not change.
func (m *Member) Get(ctx context.Context, ddbClnt db.DDBConnecter) error {
	table := viper.GetString(MemberTableConfigPath)
	mem, err := db.GetDdbObject[Member](ctx, ddbClnt, MemberKey, m.Phone, table)
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

// Put saves a Member to dynamodb.
func (m *Member) Put(ctx context.Context, ddbClnt db.DDBConnecter) error {
	table := viper.GetString(MemberTableConfigPath)
	return db.PutDdbObject(ctx, ddbClnt, table, m)
}

// Delete deletes a Member from dynamodb. If it does not exist, it will not return an error.
func (m *Member) Delete(ctx context.Context, ddbClnt db.DDBConnecter) error {
	table := viper.GetString(MemberTableConfigPath)
	return db.DelDdbItem(ctx, ddbClnt, MemberKey, m.Phone, table)
}

// SendMessage sends a text message to a Member.
func (m *Member) SendMessage(ctx context.Context, smsClnt messaging.TextSender, body string) error {
	message := messaging.TextMessage{
		Body:  body,
		Phone: m.Phone,
	}

	return messaging.SendText(ctx, smsClnt, message)
}

// IsMemberActive reports whether a Member is found (active) in dynamodb.
func IsMemberActive(ctx context.Context, ddbClnt db.DDBConnecter, phone string) (bool, error) {
	mem := Member{Phone: phone}
	if err := mem.Get(ctx, ddbClnt); err != nil {
		return false, utility.WrapError(err, "failed to check if Member is active")
	}

	// Empty string means get Member did not return an Member. Dynamodb get requests return empty data if the key does
	// not exist inside the database.
	if mem.SetupStatus == "" {
		return false, nil
	}

	return true, nil
}
