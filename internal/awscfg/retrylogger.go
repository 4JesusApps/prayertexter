package awscfg

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type loggingRetryer struct {
	delegate aws.Retryer
}

func (r *loggingRetryer) IsErrorRetryable(err error) bool {
	return r.delegate.IsErrorRetryable(err)
}

func (r *loggingRetryer) MaxAttempts() int {
	return r.delegate.MaxAttempts()
}

func (r *loggingRetryer) GetRetryToken(ctx context.Context, opErr error) (func(error) error, error) {
	return r.delegate.GetRetryToken(ctx, opErr)
}

func (r *loggingRetryer) GetInitialToken() func(error) error {
	return r.delegate.GetInitialToken()
}

func (r *loggingRetryer) RetryDelay(attempt int, opErr error) (time.Duration, error) {
	delay, calcErr := r.delegate.RetryDelay(attempt, opErr)
	slog.Warn("AWS retry", "attempt", attempt, "error", opErr, "delay", delay)
	return delay, calcErr
}
