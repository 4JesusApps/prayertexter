package prayertexter

import (
	"reflect"
	"testing"
)

func TestGenerateID(t *testing.T) {
	id, err := generateID()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(id) != 32 {
		t.Errorf("expected string of 32 length, got %v", id)
	}
}

func testRemoveItem[T comparable](t *testing.T, items []T, target T, expected []T) {
	removeItem(&items, target)
	if items == nil {
		items = []T{}
	}
	if !reflect.DeepEqual(items, expected) {
		t.Errorf("for target %v, expected %v, got %v", target, expected, items)
	}
}

func TestRemoveItem(t *testing.T) {
	// int slice test
	testRemoveItem(t, []int{1, 2, 3, 4, 2}, 2, []int{1, 3, 4})

	// string slice test
	testRemoveItem(t, []string{"apple", "banana", "cherry", "banana"}, "banana", []string{"apple", "cherry"})

	// empty slice test
	testRemoveItem(t, []int{}, 42, []int{})

	// State slice test
	states := []State{
		{
			Error: "sample error text",
			Message: TextMessage{
				Body:  "sample text message 1",
				Phone: "123-456-7890",
			},
			ID:        "67f8ce776cc147c2b8700af909639ba2",
			Stage:     "HELP",
			Status:    "FAILED",
			TimeStart: "2025-02-16T23:54:01Z",
		},
		{
			Error: "",
			Message: TextMessage{
				Body:  "sample text message 2",
				Phone: "998-765-4321",
			},
			ID:        "19ee2955d41d08325e1a97cbba1e544b",
			Stage:     "MEMBER DELETE",
			Status:    "IN PROGRESS",
			TimeStart: "2025-02-16T23:57:01Z",
		},
	}
	testRemoveItem(t, states, states[0], states[1:])
}
