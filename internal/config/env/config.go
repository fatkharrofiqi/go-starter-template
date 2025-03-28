package env

import (
	"fmt"

	"github.com/spf13/viper"
	"gorm.io/gorm/logger"
)

type Config struct {
	App struct {
		Name string `mapstructure:"name"`
	} `mapstructure:"app"`
	Web struct {
		Port    int  `mapstructure:"port"`
		Prefork bool `mapstructure:"prefork"`
		Cors    struct {
			AllowOrigins string `mapstructure:"allow_origins"`
		} `mapstructure:"cors"`
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
		Log struct {
			Level logger.LogLevel `mapstructure:"level"`
		} `mapstructure:"log"`
	} `mapstructure:"database"`
	Monitoring struct {
		Otel struct {
			Host string `mapstructure:"host"`
		} `mapstructure:"otel"`
	} `mapstructure:"monitoring"`
}

func NewConfig() *Config {
	vp := viper.New()

	// Set configuration file details
	vp.SetConfigName("config")
	vp.SetConfigType("yml")
	vp.AddConfigPath("./../")
	vp.AddConfigPath("./")

	// Read the configuration file
	if err := vp.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error reading config file: %w", err))
	}

	// Unmarshal into the Config struct
	config := new(Config)
	if err := vp.Unmarshal(config); err != nil {
		panic(fmt.Errorf("fatal error unmarshaling config: %w", err))
	}

	return config
}
