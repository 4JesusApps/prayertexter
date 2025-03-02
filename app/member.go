package prayertexter

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

	// this is important so that the original Member object doesn't get reset to all empty struct
	// values if the Member does not exist in ddb
	if mem.Phone != "" {
		*m = *mem
	}

	return nil
}

func (m *Member) put(clnt DDBConnecter) error {
	return putDdbObject(clnt, memberTable, m)
}

func (m *Member) delete(clnt DDBConnecter) error {
	return delDdbItem(clnt, memberAttribute, m.Phone, memberTable)
}

func (m *Member) sendMessage(clnt DDBConnecter, sndr TextSender, body string) error {
	body = msgPre + body + "\n\n" + msgPost
	message := TextMessage{
		Body:  body,
		Phone: m.Phone,
	}

	return sndr.sendText(clnt, message)
}

func isMemberActive(clnt DDBConnecter, phone string) (bool, error) {
	mem := Member{Phone: phone}
	if err := mem.get(clnt); err != nil {
		// returning false but it really should be nil due to error
		return false, err
	}

	// empty string means get Member did not return an Member. Dynamodb get requests 
	// return empty data if the key does not exist inside the database
	if mem.SetupStatus == "" {
		return false, nil
	} else {
		return true, nil
	}
}
