package object

import (
	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
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
	DefaultStateTrackerTable    = "General"
	StateTrackerTableConfigPath = "conf.aws.db.statetracker.table"

	StateTrackerAttribute = "Key"
	StateTrackerKey       = "StateTracker"

	StateInProgress = "IN PROGRESS"
	StateFailed     = "FAILED"
)

func (st *StateTracker) Get(ddbClnt db.DDBConnecter) error {
	table := viper.GetString(StateTrackerTableConfigPath)
	sttrackr, err := db.GetDdbObject[StateTracker](ddbClnt, StateTrackerAttribute, StateTrackerKey, table)
	if err != nil {
		return err
	}

	// This is important so that the original Member object doesn't get reset to all empty struct values if the Member
	// does not exist in ddb.
	if sttrackr.Key != "" {
		*st = *sttrackr
	}

	return nil
}

func (st *StateTracker) Put(ddbClnt db.DDBConnecter) error {
	table := viper.GetString(StateTrackerTableConfigPath)
	st.Key = StateTrackerKey

	return db.PutDdbObject(ddbClnt, string(table), st)
}

func (s *State) Update(ddbClnt db.DDBConnecter, remove bool) error {
	st := StateTracker{}
	if err := st.Get(ddbClnt); err != nil {
		return utility.WrapError(err, "failed state update")
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

	err := st.Put(ddbClnt)

	return utility.WrapError(err, "failed state update")
}
