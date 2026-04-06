package model_test

import (
	"slices"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/model"
)

//nolint:gocognit // test function with multiple subtests
func TestIntercessorPhones(t *testing.T) {
	i := model.IntercessorPhones{
		Key: model.IntercessorPhonesKeyValue,
		Phones: []string{
			"+11111111111",
			"+12222222222",
			"+13333333333",
			"+14444444444",
			"+15555555555",
		},
	}

	t.Run("Test AddPhone: adds new phone to slice", func(t *testing.T) {
		newPhone := "+16666666666"
		i.AddPhone(newPhone)
		if !slices.Contains(i.Phones, newPhone) {
			t.Errorf("expected slice to contain %v, got %v", newPhone, i.Phones)
		}
	})

	t.Run("Test RemovePhone", func(t *testing.T) {
		t.Run("removes existing phone from slice", func(t *testing.T) {
			removePhone := "+13333333333"
			i.RemovePhone(removePhone)
			if slices.Contains(i.Phones, removePhone) {
				t.Errorf("expected slice to not contain %v, got %v", removePhone, i.Phones)
			}
		})

		t.Run("removes non existing phone; slice should not change", func(t *testing.T) {
			initialPhones := make([]string, len(i.Phones))
			copy(initialPhones, i.Phones)

			nonExistentPhone := "+19999999999"
			i.RemovePhone(nonExistentPhone)

			if !slices.Equal(i.Phones, initialPhones) {
				t.Errorf("expected phones to not change, got %v", i.Phones)
			}
		})
	})

	t.Run("Test GenRandPhones", func(t *testing.T) {
		const defaultIntercessorsPerPrayer = 2

		t.Run("returns correct number of phones when enough phones are in slice", func(t *testing.T) {
			phones := i.GenRandPhones(defaultIntercessorsPerPrayer)
			if len(phones) != defaultIntercessorsPerPrayer {
				t.Errorf("expected number of phones to be %v, got %v",
					defaultIntercessorsPerPrayer, len(phones))
			}

			if checkDuplicates(phones) {
				t.Errorf("expected phone list to not contain duplicates, got %v", phones)
			}
		})

		t.Run("returns fewer phones when not enough to satisfy count", func(t *testing.T) {
			for len(i.Phones) > defaultIntercessorsPerPrayer-1 {
				i.Phones = i.Phones[:len(i.Phones)-1]
			}

			phones := i.GenRandPhones(defaultIntercessorsPerPrayer)
			if len(phones) != defaultIntercessorsPerPrayer-1 {
				t.Errorf("expected phone list to be len %v, got len: %v phones: %v",
					defaultIntercessorsPerPrayer-1, len(phones), phones)
			}

			if checkDuplicates(phones) {
				t.Errorf("expected phone list to not contain duplicates, got %v", phones)
			}
		})

		t.Run("returns nil when no phones available", func(t *testing.T) {
			i.Phones = []string{}
			phones := i.GenRandPhones(defaultIntercessorsPerPrayer)
			if phones != nil {
				t.Errorf("expected nil return when phone slice is empty, got %v", phones)
			}
		})
	})
}

func TestBlockedPhones(t *testing.T) {
	b := model.BlockedPhones{
		Key: model.BlockedPhonesKeyValue,
		Phones: []string{
			"+11111111111",
			"+12222222222",
			"+13333333333",
			"+14444444444",
			"+15555555555",
		},
	}

	t.Run("Test AddPhone", func(t *testing.T) {
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
	})

	t.Run("Test RemovePhone", func(t *testing.T) {
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
	})
}

func checkDuplicates(slice []string) bool {
	seen := make(map[string]bool)
	for _, item := range slice {
		if _, ok := seen[item]; ok {
			return true
		}
		seen[item] = true
	}
	return false
}
