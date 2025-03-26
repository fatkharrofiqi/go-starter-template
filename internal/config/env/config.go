package env

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	App struct {
		Name string `mapstructure:"name"`
	} `mapstructure:"app"`
	Web struct {
		Port    int  `mapstructure:"port"`
		Prefork bool `mapstructure:"prefork"`
	} `mapstructure:"web"`
	JWT struct {
		Secret        string `mapstructure:"secret"`
		RefreshSecret string `mapstructure:"refresh_secret"`
	} `mapstructure:"jwt"`
	Log struct {
		Level int `mapstructure:"level"`
	} `mapstructure:"log"`
	Database struct {
		DSN  string `mapstructure:"dsn"`
		Pool struct {
			Idle     int `mapstructure:"idle"`
			Max      int `mapstructure:"max"`
			Lifetime int `mapstructure:"lifetime"`
		} `mapstructure:"pool"`
	} `mapstructure:"database"`
}

func NewConfig() *Config {
	config := viper.New()

	// Set configuration file details
	config.SetConfigName("config")
	config.SetConfigType("yml")
	config.AddConfigPath("./../")
	config.AddConfigPath("./")

	// Read the configuration file
	if err := config.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error reading config file: %w", err))
	}

	// Unmarshal into the Config struct
	cfg := new(Config)
	if err := config.Unmarshal(cfg); err != nil {
		panic(fmt.Errorf("fatal error unmarshaling config: %w", err))
	}

	return cfg
}
