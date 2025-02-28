package prayertexter

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
)

func generateID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		slog.Error("failed to generate random bytes")
		return "", err
	}

	return hex.EncodeToString(bytes), nil
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
