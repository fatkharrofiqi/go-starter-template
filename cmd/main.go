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
	app := config.NewFiber(cfg)

	config.Bootstrap(&config.BootstrapConfig{
		DB:        db,
		App:       app,
		Log:       log,
		Viper:     cfg,
		Validator: validator,
	})

	webPort := cfg.GetInt("web.port")
	err := app.Listen(fmt.Sprintf(":%d", webPort))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
