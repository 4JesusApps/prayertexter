// Package model contains pure domain types for the prayertexter application.
// These types have no I/O dependencies — they hold data, constants, and pure logic only.
package model

// A Member represents someone who has signed up or is in the process of signing up for prayertexter.
type Member struct {
	Administrator    bool
	Intercessor      bool
	Name             string
	Phone            string
	PrayerCount      int
	SetupStage       int
	SetupStatus      string
	WeeklyPrayerDate string
	WeeklyPrayerLimit int
}

// MemberKey is the DynamoDB partition key field name for the Member table.
const MemberKey = "Phone"

// Values used during the Member sign up process.
const (
	SetupInProgress = "IN PROGRESS"
	SetupComplete   = "COMPLETE"
	SignUpStepOne   = 1
	SignUpStepTwo   = 2
	SignUpStepThree = 3
	SignUpStepFinal = 99
)

// IsActive reports whether a Member was found in the database.
// An empty SetupStatus means the member does not exist.
func (m *Member) IsActive() bool {
	return m.SetupStatus != ""
}
