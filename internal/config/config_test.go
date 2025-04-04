package config_test

import (
	"fmt"
	"testing"

	"github.com/mshort55/prayertexter/internal/config"
	"github.com/spf13/viper"
)

func TestInitConfig(t *testing.T) {
	config.InitConfig()

	fmt.Println(viper.GetInt("conf.aws.db.timeout"))
}
