package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	AWS                   AWSConfig
	IntercessorsPerPrayer int
	PrayerReminderHours   int
}

type AWSConfig struct {
	Region  string
	Backoff int
	Retry   int
	DB      DBConfig
	SMS     SMSConfig
}

type DBConfig struct {
	Timeout                int
	MemberTable            string
	ActivePrayerTable      string
	QueuedPrayerTable      string
	BlockedPhonesTable     string
	IntercessorPhonesTable string
}

type SMSConfig struct {
	PhonePool string
	Timeout   int
}

// Load initializes Viper and returns a Config struct.
// Viper is fully contained here — no other package should import it.
func Load() Config {
	initViper()

	return Config{
		AWS: AWSConfig{
			Region:  viper.GetString("conf.aws.region"),
			Backoff: viper.GetInt("conf.aws.backoff"),
			Retry:   viper.GetInt("conf.aws.retry"),
			DB: DBConfig{
				Timeout:                viper.GetInt("conf.aws.db.timeout"),
				MemberTable:            viper.GetString("conf.aws.db.member.table"),
				ActivePrayerTable:      viper.GetString("conf.aws.db.prayer.activetable"),
				QueuedPrayerTable:      viper.GetString("conf.aws.db.prayer.queuetable"),
				BlockedPhonesTable:     viper.GetString("conf.aws.db.blockedphones.table"),
				IntercessorPhonesTable: viper.GetString("conf.aws.db.intercessorphones.table"),
			},
			SMS: SMSConfig{
				PhonePool: viper.GetString("conf.aws.sms.phonepool"),
				Timeout:   viper.GetInt("conf.aws.sms.timeout"),
			},
		},
		IntercessorsPerPrayer: viper.GetInt("conf.intercessorsperprayer"),
		PrayerReminderHours:   viper.GetInt("conf.prayerreminderhours"),
	}
}

func initViper() {
	defaults := map[string]any{
		"aws": map[string]any{
			"region":  "us-west-1",
			"backoff": 10,
			"retry":   5,
			"db": map[string]any{
				"timeout": 60,
				"blockedphones": map[string]any{
					"table": "General",
				},
				"intercessorphones": map[string]any{
					"table": "General",
				},
				"member": map[string]any{
					"table": "Member",
				},
				"prayer": map[string]any{
					"activetable": "ActivePrayer",
					"queuetable":  "QueuedPrayer",
				},
			},
			"sms": map[string]any{
				"phonepool": "dummy",
				"timeout":   60,
			},
		},
		"intercessorsperprayer": 2,
		"prayerreminderhours":   3,
	}

	viper.SetDefault("conf", defaults)
	viper.SetEnvPrefix("pray")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

// InitConfig is the legacy config initializer. Kept for backward compatibility
// during refactor. Will be removed when old packages are deleted.
func InitConfig() {
	initViper()
}
