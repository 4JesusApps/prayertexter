package utility

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateID returns a random string.
func GenerateID() (string, error) {
	size := 16
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", WrapError(err, "failed generate ID")
	}

	return hex.EncodeToString(bytes), nil
}