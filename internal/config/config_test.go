package config_test

import (
	"fmt"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/spf13/viper"
)

func TestDefaults(t *testing.T) {
	t.Run("verify that all default config values get passed through viper as expected", func(t *testing.T) {
		config.InitConfig()

		fmt.Println(viper.GetInt("conf.aws.db.timeout"))
	})
}
