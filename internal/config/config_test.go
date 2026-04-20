package config_test

import (
	"testing"

	"github.com/4JesusApps/prayertexter/internal/config"
)

func TestLoad(t *testing.T) {
	t.Run("verify that Load returns a Config with expected default values", func(t *testing.T) {
		cfg := config.Load()

		if cfg.AWS.Region != "us-west-1" {
			t.Errorf("expected region us-west-1, got %v", cfg.AWS.Region)
		}
		if cfg.AWS.Backoff != 10 {
			t.Errorf("expected backoff 10, got %v", cfg.AWS.Backoff)
		}
		if cfg.AWS.Retry != 5 {
			t.Errorf("expected retry 5, got %v", cfg.AWS.Retry)
		}
		if cfg.AWS.DB.Timeout != 60 {
			t.Errorf("expected db timeout 60, got %v", cfg.AWS.DB.Timeout)
		}
		if cfg.AWS.DB.MemberTable != "Member" {
			t.Errorf("expected member table Member, got %v", cfg.AWS.DB.MemberTable)
		}
		if cfg.AWS.DB.ActivePrayerTable != "ActivePrayer" {
			t.Errorf("expected active prayer table ActivePrayer, got %v", cfg.AWS.DB.ActivePrayerTable)
		}
		if cfg.AWS.DB.QueuedPrayerTable != "QueuedPrayer" {
			t.Errorf("expected queued prayer table QueuedPrayer, got %v", cfg.AWS.DB.QueuedPrayerTable)
		}
		if cfg.AWS.DB.BlockedPhonesTable != "General" {
			t.Errorf("expected blocked phones table General, got %v", cfg.AWS.DB.BlockedPhonesTable)
		}
		if cfg.AWS.DB.IntercessorPhonesTable != "General" {
			t.Errorf("expected intercessor phones table General, got %v", cfg.AWS.DB.IntercessorPhonesTable)
		}
		if cfg.AWS.SMS.PhonePool != "dummy" {
			t.Errorf("expected phone pool dummy, got %v", cfg.AWS.SMS.PhonePool)
		}
		if cfg.AWS.SMS.Timeout != 60 {
			t.Errorf("expected sms timeout 60, got %v", cfg.AWS.SMS.Timeout)
		}
		if cfg.IntercessorsPerPrayer != 2 {
			t.Errorf("expected intercessors per prayer 2, got %v", cfg.IntercessorsPerPrayer)
		}
		if cfg.PrayerReminderHours != 3 {
			t.Errorf("expected prayer reminder hours 3, got %v", cfg.PrayerReminderHours)
		}
	})
}

func TestEnvironmentalVariableOverride(t *testing.T) {
	t.Run("verify environmental variables can override default config values", func(t *testing.T) {
		defaultCfg := config.Load()
		defaultPhone := defaultCfg.AWS.SMS.PhonePool

		newPhone := "+17777777777"
		t.Setenv("PRAY_CONF_AWS_SMS_PHONEPOOL", newPhone)

		cfg := config.Load()
		phone := cfg.AWS.SMS.PhonePool

		if phone == defaultPhone {
			t.Errorf("expected phones to not be equal, got %v for both", phone)
		}

		if phone != newPhone {
			t.Errorf("expected phone to be %v, got %v", newPhone, phone)
		}
	})
}
