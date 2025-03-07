package prayertexter

import (
	"slices"
	"testing"
)

var i = IntercessorPhones{
	Key: intercessorPhonesKey,
	Phones: []string{
		"+11111111111",
		"+12222222222",
		"+13333333333",
		"+14444444444",
		"+15555555555",
	},
}

func TestAddPhone(t *testing.T) {
	newPhone := "+1666-666-6666"
	i.addPhone(newPhone)
	if !slices.Contains(i.Phones, newPhone) {
		t.Errorf("expected slice to contain %v, got %v", newPhone, i.Phones)
	}
}

func TestRemovePhone(t *testing.T) {
	removePhone := "+13333333333"
	i.removePhone(removePhone)
	if slices.Contains(i.Phones, removePhone) {
		t.Errorf("expected slice to not contain %v, got %v", removePhone, i.Phones)
	}
}

func TestGenRandPhones(t *testing.T) {
	phones := i.genRandPhones()
	if len(phones) != numIntercessorsPerPrayer {
		t.Errorf("expected number of phones to be %v, got %v", numIntercessorsPerPrayer, len(phones))
	}

	if checkDuplicates(phones) {
		t.Errorf("expected phone list to not contain duplicates, got %v", phones)
	}

	// this test verifies that genRandPhones can return # of phones less than
	// numIntercessorsPerPrayer if there are not enough available phones in the slice
	for len(i.Phones) > numIntercessorsPerPrayer-1 {
		i.Phones = i.Phones[:len(i.Phones)-1]
	}
	phones = i.genRandPhones()
	if len(phones) != numIntercessorsPerPrayer-1 {
		t.Errorf("expected phone list to be len %v, got len: %v phones: %v", numIntercessorsPerPrayer-1, len(phones), phones)
	}

	if checkDuplicates(phones) {
		t.Errorf("expected phone list to not contain duplicates, got %v", phones)
	}

	i.Phones = []string{}
	if phones = i.genRandPhones(); phones != nil {
		t.Errorf("expected nil return when phone slice is empty, got %v", phones)
	}
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
