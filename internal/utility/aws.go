package utility

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/viper"
)

// Default values for configuration that has been exposed to be used with the config package.
const (
	DefaultAwsRegion    = "us-west-1"
	AwsRegionConfigPath = "conf.aws.region"

	DefaultAwsSvcRetryAttempts    = 5
	AwsSvcRetryAttemptsConfigPath = "conf.aws.retry"

	DefaultAwsSvcMaxBackoff    = 10
	AwsSvcMaxBackoffConfigPath = "conf.aws.backoff"
)

// GetAwsConfig returns an aws configuration that can be used to interact with aws services.
func GetAwsConfig(ctx context.Context) (aws.Config, error) {
	region := viper.GetString(AwsRegionConfigPath)
	maxRetry := viper.GetInt(AwsSvcRetryAttemptsConfigPath)
	maxBackoff := viper.GetInt(AwsSvcMaxBackoffConfigPath)

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

// IsAwsLocal reports whether aws is running in a local testing environment (sam local).
func IsAwsLocal() bool {
	if local := os.Getenv("AWS_SAM_LOCAL"); local == "true" {
		return true
	}

	return false
}
