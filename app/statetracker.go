package prayertexter

type StateTracker struct {
	Key    string
	States []State
}

type State struct {
	Error     string
	Message   TextMessage
	ID        string
	Stage     string
	Status    string
	TimeStart string
}

const (
	stateTrackerAttribute = "Key"
	stateTrackerKey       = "StateTracker"
	stateTrackerTable     = "General"
)

func (st *StateTracker) get(ddbClnt DDBConnecter) error {
	sttrackr, err := getDdbObject[StateTracker](ddbClnt, stateTrackerAttribute, stateTrackerKey, stateTrackerTable)
	if err != nil {
		return err
	}

	// this is important so that the original Member object doesn't get reset to all empty struct
	// values if the Member does not exist in ddb
	if sttrackr.Key != "" {
		*st = *sttrackr
	}

	return nil
}

func (st *StateTracker) put(ddbClnt DDBConnecter) error {
	st.Key = stateTrackerKey
	return putDdbObject(ddbClnt, string(stateTrackerTable), st)
}

func (s *State) update(ddbClnt DDBConnecter, remove bool) error {
	st := StateTracker{}
	if err := st.get(ddbClnt); err != nil {
		return err
	}

	states := &st.States
	for _, state := range st.States {
		if state.ID == s.ID {
			removeItem(states, state)
		}
	}

	if !remove {
		st.States = append(st.States, *s)
	}

	if err := st.put(ddbClnt); err != nil {
		return err
	}

	return nil
}
