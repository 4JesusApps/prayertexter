package utility

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// LoggingRetryer implements the aws.Retryer interface. The only purpose of this is so AWS retries are logged within
// this application. If not for this, AWS retries would happen silently. This should improve debugging and give insight
// into AWS retry attempts.
type LoggingRetryer struct {
	delegate aws.Retryer
}

// IsErrorRetryable is a dummy method to satisfy the aws.Retryer interface. It delegates to the actual Retryer.
func (r *LoggingRetryer) IsErrorRetryable(err error) bool {
	return r.delegate.IsErrorRetryable(err)
}

// MaxAttempts is a dummy method to satisfy the aws.Retryer interface. It delegates to the actual Retryer.
func (r *LoggingRetryer) MaxAttempts() int {
	return r.delegate.MaxAttempts()
}

// GetRetryToken is a dummy method to satisfy the aws.Retryer interface. It delegates to the actual Retryer.
func (r *LoggingRetryer) GetRetryToken(ctx context.Context, opErr error) (func(error) error, error) {
	return r.delegate.GetRetryToken(ctx, opErr)
}

// GetInitialToken is a dummy method to satisfy the aws.Retryer interface. It delegates to the actual Retryer.
func (r *LoggingRetryer) GetInitialToken() func(error) error {
	return r.delegate.GetInitialToken()
}

// RetryDelay delegates to the actual aws.Retryer, while also adding logging in between so that aws retries are visible
// in application logs.
func (r *LoggingRetryer) RetryDelay(attempt int, opErr error) (time.Duration, error) {
	delay, calcErr := r.delegate.RetryDelay(attempt, opErr)
	slog.Warn("AWS retry", "attempt", attempt, "error", opErr, "delay", delay)
	return delay, calcErr
}
