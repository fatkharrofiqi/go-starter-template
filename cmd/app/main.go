package main

import (
	"fmt"
	"go-starter-template/internal/config"
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/config/monitoring"
	"go-starter-template/internal/config/validation"
)

func main() {
	// Load the configuration
	cfg := env.NewConfig()
	log := config.NewLogger(cfg)
	db := config.NewDatabase(cfg, log)
	validation := validation.NewValidation()
	app := config.NewFiber(cfg, log)
	trace := monitoring.NewMonitoring(log, cfg)
	defer trace.Shutdown()

	config.Bootstrap(&config.BootstrapConfig{
		DB:         db,
		App:        app,
		Log:        log,
		Config:     cfg,
		Validation: validation,
		Monitoring: trace,
	})

	webPort := cfg.Web.Port
	err := app.Listen(fmt.Sprintf(":%d", webPort))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
