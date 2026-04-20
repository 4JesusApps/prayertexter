package awscfg

import (
	"context"
	"os"
	"time"

	"github.com/4JesusApps/prayertexter/internal/apperr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	defaultRegion     = "us-west-1"
	defaultMaxRetry   = 5
	defaultMaxBackoff = 10
)

func GetAwsConfig(ctx context.Context) (aws.Config, error) {
	region := defaultRegion
	if r := os.Getenv("PRAY_CONF_AWS_REGION"); r != "" {
		region = r
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region),
		config.WithRetryer(func() aws.Retryer {
			retryer := retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = defaultMaxRetry
				o.MaxBackoff = time.Duration(defaultMaxBackoff) * time.Second
			})
			return &loggingRetryer{delegate: retryer}
		}))

	return cfg, apperr.WrapError(err, "failed to get aws config")
}
