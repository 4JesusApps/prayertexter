package model

// A Prayer represents a prayer request.
type Prayer struct {
	Intercessor      Member
	IntercessorPhone string
	ReminderCount    int
	ReminderDate     string
	Request          string
	Requestor        Member
}

// PrayerKey is the DynamoDB partition key field name for prayer tables.
const PrayerKey = "IntercessorPhone"

// IsActive reports whether a Prayer was found in the database.
// An empty Request means the prayer does not exist.
func (p *Prayer) IsActive() bool {
	return p.Request != ""
}
