package domain

type Prayer struct {
	Intercessor       Member
	IntercessorPhone  string
	QueueID           string
	ReminderCount     int
	ReminderDate      string
	Request           string
	RequestorNotified bool
	Requestor         Member
}
