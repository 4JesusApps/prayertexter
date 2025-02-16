package prayertexter

type StateTracker struct {
	Key    string
	States []State
}

type State struct {
	Error      error
	Message    TextMessage
	RequestID  string
	Stage      string
	Status     string
	TimeStart  string
	TimeFinish string
}

const (
	stateTrackerAttribute = "Key"
	stateTrackerKey       = "StateTracker"
	stateTrackerTable     = "General"
)

func (st *StateTracker) get(clnt DDBConnecter) error {
	sttrckr, err := getDdbObject[StateTracker](clnt, stateTrackerAttribute, stateTrackerKey, stateTrackerTable)
	if err != nil {
		return err
	}

	*st = *sttrckr

	return nil
}

func (st *StateTracker) put(clnt DDBConnecter) error {
	st.Key = stateTrackerKey
	return putDdbObject(clnt, string(stateTrackerTable), st)
}

func (s *State) update(clnt DDBConnecter) error {
	st := StateTracker{}
	if err := st.get(clnt); err != nil {
		return err
	}

	st.States = append(st.States, *s)
	if err := st.put(clnt); err != nil {
		return err
	}

	return nil
}
