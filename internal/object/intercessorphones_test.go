package object_test

import (
	"slices"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/object"
)

func TestIntercessorPhones(t *testing.T) {
	i := object.IntercessorPhones{
		Key: object.IntercessorPhonesKey,
		Phones: []string{
			"+11111111111",
			"+12222222222",
			"+13333333333",
			"+14444444444",
			"+15555555555",
		},
	}

	t.Run("Test AddPhone: adds new phone to slice", func(t *testing.T) {
		testAddPhone(t, &i)
	})

	t.Run("Test RemovePhone", func(t *testing.T) {
		testRemovePhone(t, &i)
	})

	t.Run("Test GenRandPhones", func(t *testing.T) {
		testGenRandPhones(t, &i)
	})
}

func testAddPhone(t *testing.T, i *object.IntercessorPhones) {
	t.Run("Test AddPhone: adds new phone to slice", func(t *testing.T) {
		newPhone := "+16666666666"
		i.AddPhone(newPhone)
		if !slices.Contains(i.Phones, newPhone) {
			t.Errorf("expected slice to contain %v, got %v", newPhone, i.Phones)
		}
	})
}

func testRemovePhone(t *testing.T, i *object.IntercessorPhones) {
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
}

func testGenRandPhones(t *testing.T, i *object.IntercessorPhones) {
	config.InitConfig()
	t.Run("returns correct number of phones (NumIntercessorsPerPrayer) when enough phones are in slice",
		func(t *testing.T) {
			phones := i.GenRandPhones()
			if len(phones) != object.DefaultIntercessorsPerPrayer {
				t.Errorf("expected number of phones to be %v, got %v",
					object.DefaultIntercessorsPerPrayer, len(phones))
			}

			if checkDuplicates(phones) {
				t.Errorf("expected phone list to not contain duplicates, got %v", phones)
			}
		})

	t.Run("returns fewer phones there are not enough to satisfy NumIntercessorsPerPrayer", func(t *testing.T) {
		// This reduces phone list to less than NumIntercessorsPerPrayer.
		for len(i.Phones) > object.DefaultIntercessorsPerPrayer-1 {
			i.Phones = i.Phones[:len(i.Phones)-1]
		}

		phones := i.GenRandPhones()
		if len(phones) != object.DefaultIntercessorsPerPrayer-1 {
			t.Errorf("expected phone list to be len %v, got len: %v phones: %v",
				object.DefaultIntercessorsPerPrayer-1, len(phones), phones)
		}

		if checkDuplicates(phones) {
			t.Errorf("expected phone list to not contain duplicates, got %v", phones)
		}
	})

	t.Run("returns nil when no phones available", func(t *testing.T) {
		i.Phones = []string{}
		phones := i.GenRandPhones()
		if phones != nil {
			t.Errorf("expected nil return when phone slice is empty, got %v", phones)
		}
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
