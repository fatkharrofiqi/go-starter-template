package main

import (
	app "go-starter-template/internal"
	"go-starter-template/internal/config/database"
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/config/logger"
	"go-starter-template/internal/config/monitor"
	"go-starter-template/internal/config/redis"
	"go-starter-template/internal/config/validation"
	"go-starter-template/internal/config/web"
)

func main() {
	config := env.NewConfig()
	log := logger.NewLogger(config)
	web := web.NewFiber(log, config)
	redis := redis.NewRedis(log, config)
	db := database.NewDatabase(log, config)
	monitoring := monitor.NewMonitoring(log, config)
	validation := validation.NewValidation()
	defer monitoring.Shutdown()

	server := app.NewApp(log, config, db, web, validation, redis)
	server.Run()
}
