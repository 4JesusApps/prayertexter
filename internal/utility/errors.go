package utility

import (
	"fmt"
	"log/slog"
)

type constError string

func (err constError) Error() string {
	return string(err)
}

// Errors raised by FindIntercessors.
const (
	ErrNoAvailableIntercessors = constError("no available intercessors")
	ErrIntercessorUnavailable  = constError("intercessor unavailable")
)

// Use this for high level functions where you want to log and wrap the error.
func LogAndWrapError(err error, message string, attrs ...any) error {
	if err == nil {
		return nil
	}
	slog.Error(message, append(attrs, "error", err)...)
	return fmt.Errorf("%s: %w", message, err)
}

// Use this for internal function where logging will happen at a higher level.
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
