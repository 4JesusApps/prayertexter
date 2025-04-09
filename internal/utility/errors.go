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

// LogAndWrapError will log, wrap, and return an error and is used for high level functions where most logging is done.
// If the error is nil, this will return nil as well.
func LogAndWrapError(err error, message string, attrs ...any) error {
	if err == nil {
		return nil
	}
	slog.Error(message, append(attrs, "error", err)...)
	return fmt.Errorf("%s: %w", message, err)
}

// WrapError will wrap and return an error and is used when logging is not needed (lower level functions where error is
// passed up the chain). If the error is nil, this will return nil as well.
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
