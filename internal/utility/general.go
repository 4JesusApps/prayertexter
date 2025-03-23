package utility

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateID() (string, error) {
	size := 16
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", WrapError(err, "failed generate ID")
	}

	return hex.EncodeToString(bytes), nil
}

func RemoveItem[T comparable](items *[]T, target T) {
	slice := *items
	var newItems []T

	for _, v := range slice {
		if v != target {
			newItems = append(newItems, v)
		}
	}

	*items = newItems
}
