package model

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const idSize = 16

// GenerateID returns a cryptographically random hex string.
func GenerateID() (string, error) {
	bytes := make([]byte, idSize)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}
