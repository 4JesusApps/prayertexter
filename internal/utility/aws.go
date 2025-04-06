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

const (
	DefaultAwsSvcRetryAttempts    = 5
	AwsSvcRetryAttemptsConfigPath = "conf.aws.retry"

	DefaultAwsSvcMaxBackoff    = 10
	AwsSvcMaxBackoffConfigPath = "conf.aws.backoff"
)

func GetAwsConfig() (aws.Config, error) {
	maxRetry := viper.GetInt(AwsSvcRetryAttemptsConfigPath)
	maxBackoff := viper.GetInt(AwsSvcMaxBackoffConfigPath)

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion("us-west-1"),
		config.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = maxRetry
				o.MaxBackoff = time.Duration(maxBackoff) * time.Second
			})
		}))

	return cfg, WrapError(err, "failed to get aws config")
}

func IsAwsLocal() bool {
	if local := os.Getenv("AWS_SAM_LOCAL"); local == "true" {
		return true
	}

	return false
}
