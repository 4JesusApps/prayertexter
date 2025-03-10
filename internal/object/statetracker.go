package object

import (
	"fmt"

	"github.com/mshort55/prayertexter/internal/db"
	"github.com/mshort55/prayertexter/internal/messaging"
	"github.com/mshort55/prayertexter/internal/utility"
)

type StateTracker struct {
	Key    string
	States []State
}

type State struct {
	Error     string
	Message   messaging.TextMessage
	ID        string
	Stage     string
	Status    string
	TimeStart string
}

const (
	StateTrackerAttribute = "Key"
	StateTrackerKey       = "StateTracker"
	StateTrackerTable     = "General"
)

func (st *StateTracker) Get(ddbClnt db.DDBConnecter) error {
	sttrackr, err := db.GetDdbObject[StateTracker](ddbClnt, StateTrackerAttribute, StateTrackerKey, StateTrackerTable)
	if err != nil {
		return fmt.Errorf("StateTracker get: %w", err)
	}

	// this is important so that the original Member object doesn't get reset to all empty struct
	// values if the Member does not exist in ddb
	if sttrackr.Key != "" {
		*st = *sttrackr
	}

	return nil
}

func (st *StateTracker) Put(ddbClnt db.DDBConnecter) error {
	st.Key = StateTrackerKey
	if err := db.PutDdbObject(ddbClnt, string(StateTrackerTable), st); err != nil {
		return fmt.Errorf("StateTracker put: %w", err)
	}

	return nil
}

func (s *State) Update(ddbClnt db.DDBConnecter, remove bool) error {
	st := StateTracker{}
	if err := st.Get(ddbClnt); err != nil {
		return fmt.Errorf("State update: %w", err)
	}

	states := &st.States
	for _, state := range st.States {
		if state.ID == s.ID {
			utility.RemoveItem(states, state)
		}
	}

	if !remove {
		st.States = append(st.States, *s)
	}

	if err := st.Put(ddbClnt); err != nil {
		return fmt.Errorf("State update: %w", err)
	}

	return nil
}
