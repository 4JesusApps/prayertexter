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
	resp, err := getItem(clnt, stateTrackerAttribute, stateTrackerKey, stateTrackerTable)
	if err != nil {
		return err
	}

	if err := attributevalue.UnmarshalMap(resp.Item, &st); err != nil {
		slog.Error("unmarshal failed for get StateTracker")
		return err
	}

	return nil
}

func (st *StateTracker) put(clnt DDBConnecter) error {
	st.Key = stateTrackerKey

	data, err := attributevalue.MarshalMap(st)
	if err != nil {
		slog.Error("marshal failed for put StateTracker")
		return err
	}

	if err := putItem(clnt, stateTrackerTable, data); err != nil {
		return err
	}

	return nil
}

func (state *State) save(clnt DDBConnecter) error {
	st := StateTracker{}
	if err := st.get(clnt); err != nil {
		return err
	}

	st.States = append(st.States, *state)
	if err := st.put(clnt); err != nil {
		return err
	}

	return nil
}
