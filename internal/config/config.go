/*
Package config implements configurable settings that are overridable by environmental variables. These configurations
span across multiple other packages in prayertexter. If a package decides to expose a configuration, the config package
can be used for that purpose. All configurations should have a default value which is determined by the user of this
package. All configuration defaults should be kept up to date inside this package. Configuration defaults should be
constants or variables and defined inside other packages (not this one). The config package only links the default
values determined by other packages into this one in order to organize and set defaults.
*/
package config

import (
	"strings"

	"github.com/4JesusApps/prayertexter/internal/db"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/object"
	"github.com/4JesusApps/prayertexter/internal/utility"
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

// InitConfig sets the values of all exposed configurations. This creates a global viper instance which contains all
// configuration values that can be accessed throughout the entire application. It will use default values unless
// environmental variables are present for a specific configuration, in which case it will use the value set by the
// environmental variable.
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
