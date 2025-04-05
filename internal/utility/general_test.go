package utility_test

import (
	"reflect"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/utility"
)

func TestGenerateID(t *testing.T) {
	t.Run("generate id and confirm basic details", func(t *testing.T) {
		id, err := utility.GenerateID()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(id) != 32 {
			t.Errorf("expected string of 32 length, got %v", id)
		}
	})
}

func TestRemoveItem(t *testing.T) {
	t.Run("remove integer from slice", func(t *testing.T) {
		testRemoveItem(t, []int{1, 2, 3, 4, 2}, 2, []int{1, 3, 4})
	})

	t.Run("remove string from slice", func(t *testing.T) {
		testRemoveItem(t, []string{"apple", "banana", "cherry", "banana"}, "banana", []string{"apple", "cherry"})
	})

	t.Run("remove integer from empty slice", func(t *testing.T) {
		testRemoveItem(t, []int{}, 42, []int{})
	})

	t.Run("remove State from slice", func(t *testing.T) {
		states := []object.State{
			{
				Error: "sample error text",
				Message: messaging.TextMessage{
					Body:  "sample text message 1",
					Phone: "+11234567890",
				},
				ID:        "67f8ce776cc147c2b8700af909639ba2",
				Stage:     "HELP",
				Status:    "FAILED",
				TimeStart: "2025-02-16T23:54:01Z",
			},
			{
				Error: "",
				Message: messaging.TextMessage{
					Body:  "sample text message 2",
					Phone: "+19987654321",
				},
				ID:        "19ee2955d41d08325e1a97cbba1e544b",
				Stage:     "MEMBER DELETE",
				Status:    "IN PROGRESS",
				TimeStart: "2025-02-16T23:57:01Z",
			},
		}
		testRemoveItem(t, states, states[0], states[1:])
	})
}

func testRemoveItem[T comparable](t *testing.T, items []T, target T, expected []T) {
	utility.RemoveItem(&items, target)
	if items == nil {
		items = []T{}
	}
	if !reflect.DeepEqual(items, expected) {
		t.Errorf("for target %v, expected %v, got %v", target, expected, items)
	}
}
