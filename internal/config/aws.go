package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
)

// GetAwsConfig returns an AWS configuration using the provided AWSConfig settings.
func GetAwsConfig(ctx context.Context, cfg *AWSConfig) (aws.Config, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.Region),
		awsconfig.WithRetryer(func() aws.Retryer {
			retryer := retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = cfg.Retry
				o.MaxBackoff = time.Duration(cfg.Backoff) * time.Second
			})
			return &LoggingRetryer{delegate: retryer}
		}))

	if err != nil {
		return awsCfg, fmt.Errorf("failed to get aws config: %w", err)
	}

	return awsCfg, nil
}

// IsAwsLocal reports whether AWS is running in a local testing environment (SAM local).
func IsAwsLocal() bool {
	return os.Getenv("AWS_SAM_LOCAL") == "true"
}
