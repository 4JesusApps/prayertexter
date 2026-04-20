package domain

import (
	"log/slog"
	"math/rand/v2"
	"slices"
)

type BlockedPhones struct {
	Key    string
	Phones []string
}

type IntercessorPhones struct {
	Key    string
	Phones []string
}

func (b *BlockedPhones) AddPhone(phone string) {
	if slices.Contains(b.Phones, phone) {
		return
	}
	b.Phones = append(b.Phones, phone)
}

func (b *BlockedPhones) RemovePhone(phone string) {
	removeItem(&b.Phones, phone)
}

func (i *IntercessorPhones) AddPhone(phone string) {
	if slices.Contains(i.Phones, phone) {
		return
	}
	i.Phones = append(i.Phones, phone)
}

func (i *IntercessorPhones) RemovePhone(phone string) {
	removeItem(&i.Phones, phone)
}

func (i *IntercessorPhones) GenRandPhones(intercessorsPerPrayer int) []string {
	if len(i.Phones) == 0 {
		slog.Warn("unable to generate phones, phone list is empty")
		return nil
	}

	if len(i.Phones) <= intercessorsPerPrayer {
		result := make([]string, len(i.Phones))
		copy(result, i.Phones)
		return result
	}

	var selectedPhones []string
	for len(selectedPhones) < intercessorsPerPrayer {
		phone := i.Phones[rand.IntN(len(i.Phones))] //nolint:gosec // rand is fine here, not used for security
		if slices.Contains(selectedPhones, phone) {
			continue
		}
		selectedPhones = append(selectedPhones, phone)
	}

	return selectedPhones
}

func removeItem[T comparable](items *[]T, target T) {
	slice := *items
	var newItems []T
	for _, v := range slice {
		if v != target {
			newItems = append(newItems, v)
		}
	}
	*items = newItems
}
