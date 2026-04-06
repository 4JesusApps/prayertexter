package model

import (
	"log/slog"
	"math/rand/v2"
	"slices"
)

// IntercessorPhones contains all of the phone numbers of active intercessors.
type IntercessorPhones struct {
	Key    string
	Phones []string
}

// DynamoDB key constants for IntercessorPhones.
const (
	IntercessorPhonesKey      = "Key"
	IntercessorPhonesKeyValue = "IntercessorPhones"
)

// AddPhone adds a phone number. If phone already exists, it will not add a duplicate.
func (i *IntercessorPhones) AddPhone(phone string) {
	if slices.Contains(i.Phones, phone) {
		return
	}

	i.Phones = append(i.Phones, phone)
}

// RemovePhone removes a phone number from the list.
func (i *IntercessorPhones) RemovePhone(phone string) {
	RemoveItem(&i.Phones, phone)
}

// GenRandPhones returns a random selection of phone numbers up to count.
// Returns nil if the phone list is empty.
func (i *IntercessorPhones) GenRandPhones(count int) []string {
	var selectedPhones []string

	if len(i.Phones) == 0 {
		slog.Warn("unable to generate phones, phone list is empty")
		return nil
	}

	if len(i.Phones) <= count {
		selectedPhones = append(selectedPhones, i.Phones...)
		return selectedPhones
	}

	for len(selectedPhones) < count {
		phone := i.Phones[rand.IntN(len(i.Phones))] //nolint:gosec // this is a false positive
		if slices.Contains(selectedPhones, phone) {
			continue
		}
		selectedPhones = append(selectedPhones, phone)
	}

	return selectedPhones
}

// BlockedPhones contains all of the phone numbers that are blocked.
type BlockedPhones struct {
	Key    string
	Phones []string
}

// DynamoDB key constants for BlockedPhones.
const (
	BlockedPhonesKey      = "Key"
	BlockedPhonesKeyValue = "BlockedPhones"
)

// AddPhone adds a phone number. If phone already exists, it will not add a duplicate.
func (b *BlockedPhones) AddPhone(phone string) {
	if slices.Contains(b.Phones, phone) {
		return
	}

	b.Phones = append(b.Phones, phone)
}

// RemovePhone removes a phone number from the list.
func (b *BlockedPhones) RemovePhone(phone string) {
	RemoveItem(&b.Phones, phone)
}
