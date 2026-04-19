package domain

type Prayer struct {
	Intercessor      Member
	IntercessorPhone string
	ReminderCount    int
	ReminderDate     string
	Request          string
	Requestor        Member
}
