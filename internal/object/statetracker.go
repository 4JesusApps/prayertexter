package object

import (
	"context"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

// A StateTracker contains States. This is used to track whether prayertexter operations were successful or not and used
// to retry and recover from failed operations.
type StateTracker struct {
	// Key is the dynamodb table key name used for dynamodb operations.
	Key string
	// States contain multiple State objects.
	States []State
}

// A State is the current state of a prayertexter operation. When any prayertexter operation starts, a State is created
// and is used for tracking whether that specific operation was successful or not.
type State struct {
	// Error is the error string value of a failed operation.
	Error string
	// Message is the original TextMessage that was received by prayertexter.
	Message messaging.TextMessage
	// ID is a randomly generated and unique string.
	ID string
	// Stage is the high level prayertexter stage of the operation.
	Stage string
	// Status tracks whether the operation is successful or not.
	Status string
	// TimeStart is the time when the operation is first started.
	TimeStart string
}

// Default values for configuration that has been exposed to be used with the config package.
const (
	DefaultStateTrackerTable    = "General"
	StateTrackerTableConfigPath = "conf.aws.db.statetracker.table"
)

// StateTracker object key/value used to interact with dynamodb tables.
const (
	StateTrackerKey      = "Key"
	StateTrackerKeyValue = "StateTracker"
)

// State Status field values.
const (
	StateInProgress = "IN PROGRESS"
	StateFailed     = "FAILED"
)

// Update will either add or remove a State from a StateTracker depending on the remove parameter. After adding or
// removing, it will upload the final StateTracker to dynamodb.
func (s *State) Update(ctx context.Context, ddbClnt db.DDBConnecter, remove bool) error {
	st := StateTracker{}
	if err := st.get(ctx, ddbClnt); err != nil {
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

	err := st.put(ctx, ddbClnt)

	return utility.WrapError(err, "failed state update")
}

func (st *StateTracker) get(ctx context.Context, ddbClnt db.DDBConnecter) error {
	table := viper.GetString(StateTrackerTableConfigPath)
	sttrackr, err := db.GetDdbObject[StateTracker](ctx, ddbClnt, StateTrackerKey, StateTrackerKeyValue, table)
	if err != nil {
		return err
	}

	// This is important so that the original StateTracker object doesn't get reset to all empty struct values if the
	// StateTracker does not exist in ddb.
	if sttrackr.Key != "" {
		*st = *sttrackr
	}

	return nil
}

func (st *StateTracker) put(ctx context.Context, ddbClnt db.DDBConnecter) error {
	table := viper.GetString(StateTrackerTableConfigPath)
	st.Key = StateTrackerKeyValue

	return db.PutDdbObject(ctx, ddbClnt, table, st)
}
