package service

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/4JesusApps/prayertexter/internal/apperr"
)

func generateID() (string, error) {
	size := 16
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", apperr.WrapError(err, "failed generate ID")
	}

	return hex.EncodeToString(bytes), nil
}
