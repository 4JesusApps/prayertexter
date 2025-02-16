package prayertexter

import (
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
)

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

	data, err := attributevalue.MarshalMap(st)
	if err != nil {
		slog.Error("marshal failed for put StateTracker")
		return err
	}

	if err := putDdbItem(clnt, stateTrackerTable, data); err != nil {
		return err
	}

	return nil
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
