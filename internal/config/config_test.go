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
