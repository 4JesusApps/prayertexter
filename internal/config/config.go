package config

import (
	"strings"

	"github.com/mshort55/prayertexter/internal/db"
	"github.com/mshort55/prayertexter/internal/messaging"
	"github.com/mshort55/prayertexter/internal/object"
	"github.com/mshort55/prayertexter/internal/utility"
	"github.com/spf13/viper"
)

func setDefaults() {
	defaults := map[string]any{
		"aws": map[string]any{
			"backoff": utility.DefaultAwsSvcMaxBackoff,
			"retry":   utility.DefaultAwsSvcRetryAttempts,
			"db": map[string]any{
				"timeout": db.DefaultTimeout,
				"intercessorphones": map[string]any{
					"table": object.DefaultIntercessorPhonesTable,
				},
				"member": map[string]any{
					"table": object.DefaultMemberTable,
				},
				"prayer": map[string]any{
					"activetable": object.DefaultActivePrayersTable,
					"queuetable":  object.DefaultQueuedPrayersTable,
				},
				"statetracker": map[string]any{
					"table": object.DefaultStateTrackerTable,
				},
			},
			"sms": map[string]any{
				"phone":   messaging.DefaultPhone,
				"timeout": messaging.DefaultTimeout,
			},
		},
		"intercessorsperprayer": object.DefaultIntercessorsPerPrayer,
	}

	viper.SetDefault("conf", defaults)
}

func InitConfig() {
	setDefaults()

	// This allows one to overwrite the default configurations with environment variables. For example, to
	// overwrite the config at path conf.aws.db.timeout, one could have this environmental variable set:
	// PRAY_CONF_AWS_DB_TIMEOUT=10. PRAY_ is prefixed, everything gets automatically capitalized, and the . delimiter
	// gets changed to the _ delimiter.
	viper.SetEnvPrefix("pray")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

}
