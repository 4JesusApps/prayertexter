package utility

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func GetAwsConfig() (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-1"))

	return cfg, WrapError(err, "failed to get aws config")
}

func IsAwsLocal() bool {
	if local := os.Getenv("AWS_SAM_LOCAL"); local == "true" {
		return true
	}

	return false
}
