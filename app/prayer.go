package prayertexter

type Prayer struct {
	Intercessor      Member
	IntercessorPhone string
	Request          string
	Requestor        Member
}

const (
	prayersAttribute   = "IntercessorPhone"
	activePrayersTable = "ActivePrayers"
	prayersQueueTable  = "PrayersQueue"
)

func (p *Prayer) get(clnt DDBConnecter, queue bool) error {
	// queue determines whether ActivePrayers or PrayersQueue table is used for get
	table := getPrayerTable(queue)
	pryr, err := getDdbObject[Prayer](clnt, prayersAttribute, p.IntercessorPhone, table)
	if err != nil {
		return err
	}

	// this is important so that the original prayer object doesn't get reset to all empty struct
	// values if the prayer does not exist in ddb
	if pryr.Request != "" {
		*p = *pryr
	}
	
	return nil
}

func (p *Prayer) put(clnt DDBConnecter, queue bool) error {
	// queue is only used if there are not enough intercessors available to take a prayer request
	// prayers get queued in order to save them for a time when intercessors are available
	// this will change the ddb table that the prayer is saved to
	table := getPrayerTable(queue)
	
	return putDdbObject(clnt, table, p)
}

func (p *Prayer) delete(clnt DDBConnecter, queue bool) error {
	table := getPrayerTable(queue)
	return delDdbItem(clnt, prayersAttribute, p.IntercessorPhone, table)
}

func getPrayerTable(queue bool) string {
	var table string
	if queue {
		table = prayersQueueTable
	} else {
		table = activePrayersTable
	}

	return table
}

func (p *Prayer) checkIfActive(clnt DDBConnecter) (bool, error) {
	if err := p.get(clnt, false); err != nil {
		return false, err
	}

	// empty string means get did not return a prayer
	if p.Request == "" {
		return false, nil
	} else {
		return true, nil
	}
}
