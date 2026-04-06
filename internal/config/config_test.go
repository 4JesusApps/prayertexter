package config_test

import (
	"testing"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

func TestLoad(t *testing.T) {
	t.Run("verify typed config struct has correct defaults", func(t *testing.T) {
		cfg := config.Load()

		checks := []struct {
			name     string
			got      any
			expected any
		}{
			{"AWS.Region", cfg.AWS.Region, "us-west-1"},
			{"AWS.Retry", cfg.AWS.Retry, 5},  //nolint:mnd // test default
			{"AWS.Backoff", cfg.AWS.Backoff, 10}, //nolint:mnd // test default
			{"AWS.DB.Timeout", cfg.AWS.DB.Timeout, 60}, //nolint:mnd // test default
			{"AWS.DB.MemberTable", cfg.AWS.DB.MemberTable, "Member"},
			{"AWS.DB.ActivePrayerTable", cfg.AWS.DB.ActivePrayerTable, "ActivePrayer"},
			{"AWS.DB.QueuedPrayerTable", cfg.AWS.DB.QueuedPrayerTable, "QueuedPrayer"},
			{"AWS.DB.IntercessorPhonesTable", cfg.AWS.DB.IntercessorPhonesTable, "General"},
			{"AWS.DB.BlockedPhonesTable", cfg.AWS.DB.BlockedPhonesTable, "General"},
			{"SMS.PhonePool", cfg.SMS.PhonePool, "dummy"},
			{"SMS.Timeout", cfg.SMS.Timeout, 60}, //nolint:mnd // test default
			{"Prayer.IntercessorsPerPrayer", cfg.Prayer.IntercessorsPerPrayer, 2}, //nolint:mnd // test default
			{"Prayer.ReminderHours", cfg.Prayer.ReminderHours, 3}, //nolint:mnd // test default
		}

		for _, c := range checks {
			if c.got != c.expected {
				t.Errorf("%s: expected %v, got %v", c.name, c.expected, c.got)
			}
		}
	})

	t.Run("verify environment variable override works with Load", func(t *testing.T) {
		newPhone := "+17777777777"
		t.Setenv("PRAY_CONF_AWS_SMS_PHONEPOOL", newPhone)

		cfg := config.Load()

		if cfg.SMS.PhonePool != newPhone {
			t.Errorf("expected phone pool %v, got %v", newPhone, cfg.SMS.PhonePool)
		}
	})
}

// TestDefaultConfigValues verifies backward compatibility: InitConfig still populates global Viper.
func TestDefaultConfigValues(t *testing.T) {
	t.Run("verify that all default config values get passed through viper as expected", func(t *testing.T) {
		config.InitConfig()

		configs := map[string]any{
			utility.AwsRegionConfigPath:             utility.DefaultAwsRegion,
			utility.AwsSvcMaxBackoffConfigPath:      utility.DefaultAwsSvcMaxBackoff,
			utility.AwsSvcRetryAttemptsConfigPath:   utility.DefaultAwsSvcRetryAttempts,
			db.TimeoutConfigPath:                    db.DefaultTimeout,
			object.BlockedPhonesTableConfigPath:     object.DefaultBlockedPhonesTable,
			object.IntercessorPhonesTableConfigPath: object.DefaultIntercessorPhonesTable,
			object.MemberTableConfigPath:            object.DefaultMemberTable,
			object.ActivePrayersTableConfigPath:     object.DefaultActivePrayersTable,
			object.QueuedPrayersTableConfigPath:     object.DefaultQueuedPrayersTable,
			messaging.PhonePoolConfigPath:           messaging.DefaultPhonePool,
			messaging.TimeoutConfigPath:             messaging.DefaultTimeout,
			object.IntercessorsPerPrayerConfigPath:  object.DefaultIntercessorsPerPrayer,
			object.PrayerReminderHoursConfigPath:    object.DefaultPrayerReminderHours,
		}

		var config any
		for configPath, configValue := range configs {
			switch configValue.(type) {
			case int:
				config = viper.GetInt(configPath)
			case string:
				config = viper.GetString(configPath)
			default:
				t.Errorf("expected type int or string, got %T", configValue)
				return
			}

			if config != configValue {
				t.Errorf("expected value for config path %v to be %v, got %v", configPath, configValue, config)
			}
		}
	})
}

func TestEnvironmentalVariableOverride(t *testing.T) {
	t.Run("verify environmental variables can override default config values", func(t *testing.T) {
		config.InitConfig()
		defaultPhone := viper.GetString(messaging.PhonePoolConfigPath)
		newPhone := "+17777777777"

		t.Setenv("PRAY_CONF_AWS_SMS_PHONEPOOL", newPhone)

		config.InitConfig()
		phone := viper.GetString(messaging.PhonePoolConfigPath)

		if phone == defaultPhone {
			t.Errorf("expected phones to not be equal, got %v for both", phone)
		}

		if phone != newPhone {
			t.Errorf("expected phone to be %v, got %v", newPhone, phone)
		}
	})
}
