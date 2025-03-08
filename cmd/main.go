package main

import (
	"fmt"
	"go-starter-template/internal/config"
)

func main() {
	// Load the configuration
	cfg := config.NewViper()
	log := config.NewLogger(cfg)
	db := config.NewDatabase(cfg, log)
	validator := config.NewValidator()
	app := config.NewGin()

	config.Bootstrap(&config.BootstrapConfig{
		DB:        db,
		App:       app,
		Log:       log,
		Viper:     cfg,
		Validator: validator,
	})

	webPort := cfg.GetInt("web.port")
	if err := app.Run(fmt.Sprintf(":%d", webPort)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
