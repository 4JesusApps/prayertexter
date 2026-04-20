package apperr

import (
	"context"
	"fmt"
	"log/slog"
)

func LogAndWrapError(ctx context.Context, err error, message string, attrs ...any) error {
	if err == nil {
		return nil
	}
	slog.ErrorContext(ctx, message, append(attrs, "error", err)...)
	return fmt.Errorf("%s: %w", message, err)
}

func LogError(ctx context.Context, err error, message string, attrs ...any) {
	slog.ErrorContext(ctx, message, append(attrs, "error", err)...)
}

func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
