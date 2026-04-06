/*
Package config implements application configuration. All settings are loaded from environment variables
with the PRAY_ prefix and fall back to sensible defaults. The typed Config struct provides compile-time
safe access to configuration values. Viper is used internally for env var binding and MUST NOT be
accessed outside this package after full migration.
*/
package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	AWS    AWSConfig
	SMS    SMSConfig
	Prayer PrayerConfig
}

// AWSConfig holds AWS-related configuration.
type AWSConfig struct {
	Region  string
	Retry   int
	Backoff int
	DB      DBConfig
}

// DBConfig holds DynamoDB table names and timeout.
type DBConfig struct {
	Timeout            int
	MemberTable        string
	ActivePrayerTable  string
	QueuedPrayerTable  string
	IntercessorPhonesTable string
	BlockedPhonesTable string
}

// SMSConfig holds SMS/Pinpoint configuration.
type SMSConfig struct {
	PhonePool string
	Timeout   int
}

// PrayerConfig holds prayer-related business configuration.
type PrayerConfig struct {
	IntercessorsPerPrayer int
	ReminderHours         int
}

// initViper sets up Viper with defaults and env var binding. This populates the global Viper instance
// so that code still reading Viper directly (object CRUD methods) continues to work during transition.
func initViper() {
	defaults := map[string]any{
		"aws": map[string]any{
			"region":  "us-west-1",
			"backoff": 10, //nolint:mnd // default AWS backoff seconds
			"retry":   5,  //nolint:mnd // default AWS retry attempts
			"db": map[string]any{
				"timeout": 60, //nolint:mnd // default DB timeout seconds
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
				"timeout":   60, //nolint:mnd // default SMS timeout seconds
			},
		},
		"intercessorsperprayer": 2, //nolint:mnd // default intercessors per prayer
		"prayerreminderhours":   3, //nolint:mnd // default prayer reminder hours
	}

	viper.SetDefault("conf", defaults)
	viper.SetEnvPrefix("pray")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

// Load initializes configuration from environment variables and defaults, returning a typed Config.
// It also populates the global Viper instance for backward compatibility with code that still reads
// Viper directly.
func Load() *Config {
	initViper()

	return &Config{
		AWS: AWSConfig{
			Region:  viper.GetString("conf.aws.region"),
			Retry:   viper.GetInt("conf.aws.retry"),
			Backoff: viper.GetInt("conf.aws.backoff"),
			DB: DBConfig{
				Timeout:            viper.GetInt("conf.aws.db.timeout"),
				MemberTable:        viper.GetString("conf.aws.db.member.table"),
				ActivePrayerTable:  viper.GetString("conf.aws.db.prayer.activetable"),
				QueuedPrayerTable:  viper.GetString("conf.aws.db.prayer.queuetable"),
				IntercessorPhonesTable: viper.GetString("conf.aws.db.intercessorphones.table"),
				BlockedPhonesTable: viper.GetString("conf.aws.db.blockedphones.table"),
			},
		},
		SMS: SMSConfig{
			PhonePool: viper.GetString("conf.aws.sms.phonepool"),
			Timeout:   viper.GetInt("conf.aws.sms.timeout"),
		},
		Prayer: PrayerConfig{
			IntercessorsPerPrayer: viper.GetInt("conf.intercessorsperprayer"),
			ReminderHours:         viper.GetInt("conf.prayerreminderhours"),
		},
	}
}

// InitConfig sets up global Viper configuration for backward compatibility.
// Deprecated: Use Load() instead to get a typed Config struct.
func InitConfig() {
	initViper()
}
