package awscfg

import (
	"context"
	"time"

	"github.com/4JesusApps/prayertexter/internal/apperr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
)

func GetAwsConfig(ctx context.Context, region string, maxRetry, maxBackoffSeconds int) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region),
		config.WithRetryer(func() aws.Retryer {
			retryer := retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = maxRetry
				o.MaxBackoff = time.Duration(maxBackoffSeconds) * time.Second
			})
			return &loggingRetryer{delegate: retryer}
		}))

	return cfg, apperr.WrapError(err, "failed to get aws config")
}
