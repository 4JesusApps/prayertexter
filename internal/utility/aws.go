package utility

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	DefaultAwsRegion           = "us-west-1"
	DefaultAwsSvcRetryAttempts = 5
	DefaultAwsSvcMaxBackoff    = 10
)

func GetAwsConfig(ctx context.Context) (aws.Config, error) {
	region := DefaultAwsRegion
	maxRetry := DefaultAwsSvcRetryAttempts
	maxBackoff := DefaultAwsSvcMaxBackoff

	if r := os.Getenv("PRAY_CONF_AWS_REGION"); r != "" {
		region = r
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region),
		config.WithRetryer(func() aws.Retryer {
			retryer := retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = maxRetry
				o.MaxBackoff = time.Duration(maxBackoff) * time.Second
			})
			return &LoggingRetryer{delegate: retryer}
		}))

	return cfg, WrapError(err, "failed to get aws config")
}

func IsAwsLocal() bool {
	return os.Getenv("AWS_SAM_LOCAL") == "true"
}
