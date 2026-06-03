package conf

import (
	"context"
	"strings"

	"github.com/spf13/viper"
	"github.com/swiggy-private/gocommons/log"
)

var configuration Configuration

func GetConfig() Configuration {
	return configuration
}

type Databricks struct {
	AnalyticsHost  string `mapstructure:"analyticshost"`
	AnalyticsToken string `mapstructure:"analyticstoken"`
}

type OpsGenie struct {
	APIKey      string `mapstructure:"apikey"`
	URL         string `mapstructure:"url"`
	InfraOncall string `mapstructure:"infra_oncall"`
	CodeOncall  string `mapstructure:"code_oncall"`
}

type Server struct {
	Port int `mapstructure:"port"`
}

type Configuration struct {
	Databricks Databricks `mapstructure:"databricks"`
	OpsGenie   OpsGenie   `mapstructure:"opsgenie"`
	Server     Server     `mapstructure:"server"`
}

func Initialize(path string) error {
	log.Infow(context.Background(), "initializing config")

	viper.AutomaticEnv()
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		log.Errorw(context.Background(), "error reading config file", "err", err)
		return err
	}

	if err := viper.Unmarshal(&configuration); err != nil {
		log.Errorw(context.Background(), "error unmarshalling config", "err", err)
		return err
	}

	log.Infow(context.Background(), "config initialized successfully")
	return nil
}