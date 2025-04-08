package utility

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// Logging Retryer implements the aws.Retryer interface. The only purpose of this is so AWS retries are logged within
// this application. If not for this, AWS retries would happen silently. This should improve debugging and give insight
// into AWS retry attempts.
type LoggingRetryer struct {
	delegate aws.Retryer
}

func (r *LoggingRetryer) IsErrorRetryable(err error) bool {
	return r.delegate.IsErrorRetryable(err)
}

func (r *LoggingRetryer) MaxAttempts() int {
	return r.delegate.MaxAttempts()
}

func (r *LoggingRetryer) GetRetryToken(ctx context.Context, opErr error) (releaseToken func(error) error, err error) {
	return r.delegate.GetRetryToken(ctx, opErr)
}

func (r *LoggingRetryer) GetInitialToken() (releaseToken func(error) error) {
	return r.delegate.GetInitialToken()
}

func (r *LoggingRetryer) RetryDelay(attempt int, opErr error) (time.Duration, error) {
	delay, calcErr := r.delegate.RetryDelay(attempt, opErr)
	slog.Warn("AWS retry", "attempt", attempt, "error", opErr, "delay", delay)
	return delay, calcErr
}
