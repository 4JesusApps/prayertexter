package object_test

import (
	"slices"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/object"
)

func TestBlockedPhones(t *testing.T) {
	b := object.BlockedPhones{
		Key: object.BlockedPhonesKeyValue,
		Phones: []string{
			"+11111111111",
			"+12222222222",
			"+13333333333",
			"+14444444444",
			"+15555555555",
		},
	}

	t.Run("Test AddPhone", func(t *testing.T) {
		testBlockedPhonesAddPhone(t, &b)
	})

	t.Run("Test RemovePhone", func(t *testing.T) {
		testBlockedPhonesRemovePhone(t, &b)
	})
}

func testBlockedPhonesAddPhone(t *testing.T, b *object.BlockedPhones) {
	t.Run("adds new phone to slice", func(t *testing.T) {
		newPhone := "+16666666666"
		b.AddPhone(newPhone)
		if !slices.Contains(b.Phones, newPhone) {
			t.Errorf("expected slice to contain %v, got %v", newPhone, b.Phones)
		}
	})

	t.Run("does not add duplicate phone", func(t *testing.T) {
		initialPhones := make([]string, len(b.Phones))
		copy(initialPhones, b.Phones)

		existingPhone := b.Phones[0]
		b.AddPhone(existingPhone)
		if !slices.Equal(b.Phones, initialPhones) {
			t.Errorf("expected phones to not change when adding duplicate, got %v", b.Phones)
		}
	})
}

func testBlockedPhonesRemovePhone(t *testing.T, b *object.BlockedPhones) {
	t.Run("removes existing phone from slice", func(t *testing.T) {
		removePhone := "+13333333333"
		b.RemovePhone(removePhone)
		if slices.Contains(b.Phones, removePhone) {
			t.Errorf("expected slice to not contain %v, got %v", removePhone, b.Phones)
		}
	})

	t.Run("removes non existing phone; slice should not change", func(t *testing.T) {
		initialPhones := make([]string, len(b.Phones))
		copy(initialPhones, b.Phones)

		nonExistentPhone := "+19999999999"
		b.RemovePhone(nonExistentPhone)
		if !slices.Equal(b.Phones, initialPhones) {
			t.Errorf("expected phones to not change, got %v", b.Phones)
		}
	})
}
