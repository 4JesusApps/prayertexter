package awscfg_test

import (
	"context"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/awscfg"
	"github.com/stretchr/testify/require"
)

func TestGetAwsConfig_PlumbsValues(t *testing.T) {
	ctx := context.Background()

	cfg, err := awscfg.GetAwsConfig(ctx, "us-east-2", 7, 15)
	require.NoError(t, err)

	require.Equal(t, "us-east-2", cfg.Region)

	require.NotNil(t, cfg.Retryer)
	retryer := cfg.Retryer()
	require.Equal(t, 7, retryer.MaxAttempts())
}
