package config_test

import (
	"testing"

	"github.com/spf13/viper"

	"github.com/4JesusApps/prayertexter/internal/config"
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
			{"AWS.Retry", cfg.AWS.Retry, 5},
			{"AWS.Backoff", cfg.AWS.Backoff, 10},
			{"AWS.DB.Timeout", cfg.AWS.DB.Timeout, 60},
			{"AWS.DB.MemberTable", cfg.AWS.DB.MemberTable, "Member"},
			{"AWS.DB.ActivePrayerTable", cfg.AWS.DB.ActivePrayerTable, "ActivePrayer"},
			{"AWS.DB.QueuedPrayerTable", cfg.AWS.DB.QueuedPrayerTable, "QueuedPrayer"},
			{"AWS.DB.IntercessorPhonesTable", cfg.AWS.DB.IntercessorPhonesTable, "General"},
			{"AWS.DB.BlockedPhonesTable", cfg.AWS.DB.BlockedPhonesTable, "General"},
			{"SMS.PhonePool", cfg.SMS.PhonePool, "dummy"},
			{"SMS.Timeout", cfg.SMS.Timeout, 60},
			{"Prayer.IntercessorsPerPrayer", cfg.Prayer.IntercessorsPerPrayer, 2},
			{"Prayer.ReminderHours", cfg.Prayer.ReminderHours, 3},
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

func TestDefaultConfigValues(t *testing.T) {
	t.Run("verify that all default config values get passed through viper as expected", func(t *testing.T) {
		config.InitConfig()

		configs := map[string]any{
			"conf.aws.region":                     "us-west-1",
			"conf.aws.backoff":                    10,
			"conf.aws.retry":                      5,
			"conf.aws.db.timeout":                 60,
			"conf.aws.db.blockedphones.table":     "General",
			"conf.aws.db.intercessorphones.table": "General",
			"conf.aws.db.member.table":            "Member",
			"conf.aws.db.prayer.activetable":      "ActivePrayer",
			"conf.aws.db.prayer.queuetable":       "QueuedPrayer",
			"conf.aws.sms.phonepool":              "dummy",
			"conf.aws.sms.timeout":                60,
			"conf.intercessorsperprayer":          2,
			"conf.prayerreminderhours":            3,
		}

		var cfgVal any
		for configPath, configValue := range configs {
			switch configValue.(type) {
			case int:
				cfgVal = viper.GetInt(configPath)
			case string:
				cfgVal = viper.GetString(configPath)
			default:
				t.Errorf("expected type int or string, got %T", configValue)
				return
			}

			if cfgVal != configValue {
				t.Errorf("expected value for config path %v to be %v, got %v", configPath, configValue, cfgVal)
			}
		}
	})
}

func TestEnvironmentalVariableOverride(t *testing.T) {
	t.Run("verify environmental variables can override default config values", func(t *testing.T) {
		config.InitConfig()
		defaultPhone := viper.GetString("conf.aws.sms.phonepool")
		newPhone := "+17777777777"

		t.Setenv("PRAY_CONF_AWS_SMS_PHONEPOOL", newPhone)

		config.InitConfig()
		phone := viper.GetString("conf.aws.sms.phonepool")

		if phone == defaultPhone {
			t.Errorf("expected phones to not be equal, got %v for both", phone)
		}

		if phone != newPhone {
			t.Errorf("expected phone to be %v, got %v", newPhone, phone)
		}
	})
}
