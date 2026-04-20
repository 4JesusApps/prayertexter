package service //nolint:testpackage // testing unexported generateID

import "testing"

func TestGenerateID(t *testing.T) {
	t.Run("generate id and confirm basic details", func(t *testing.T) {
		id, err := generateID()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(id) != 32 {
			t.Errorf("expected string of 32 length, got %v", id)
		}
	})
}
