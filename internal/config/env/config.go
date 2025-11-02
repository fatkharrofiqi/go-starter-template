package env

import (
    "fmt"
    "time"

    "github.com/spf13/viper"
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
		Secret                 string        `mapstructure:"secret"`
		CsrfSecret             string        `mapstructure:"csrf_secret"`
		RefreshSecret          string        `mapstructure:"refresh_secret"`
		CsrfTokenExpiration    time.Duration `mapstructure:"csrf_token_expiration"`
		AccessTokenExpiration  time.Duration `mapstructure:"access_token_expiration"`
		RefreshTokenExpiration time.Duration `mapstructure:"refresh_token_expiration"`
	} `mapstructure:"jwt"`
	Redis struct {
		Address  string `mapstructure:"address"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
		Pool     struct {
			Size        int   `mapstructure:"size"`
			MinIdle     int   `mapstructure:"min_idle"`
			MaxIdle     int   `mapstructure:"max_idle"`
			Lifetime    int64 `mapstructure:"lifetime"`
			IdleTimeout int64 `mapstructure:"idle_timeout"`
		} `mapstructure:"pool"`
	} `mapstructure:"redis"`
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
            Level int `mapstructure:"level"`
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

func (c *Config) GetAccessSecret() string {
    return c.JWT.Secret
}

func (c *Config) GetRefreshSecret() string {
    return c.JWT.RefreshSecret
}

func (c *Config) GetCsrfSecret() string {
	return c.JWT.CsrfSecret
}

func (c *Config) GetAccessTokenExpiration() time.Duration {
	return c.JWT.AccessTokenExpiration * time.Second
}

func (c *Config) GetRefreshTokenExpiration() time.Duration {
	return c.JWT.RefreshTokenExpiration * time.Second
}

func (c *Config) GetCsrfTokenExpiration() time.Duration {
	return c.JWT.CsrfTokenExpiration * time.Second
}
