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
	web := web.NewFiber(config)
	log := logger.NewLogger(config)
	redis := redis.NewRedis(log, config)
    sqlDB := database.NewSQLDatabase(log, config)
	monitoring := monitor.NewMonitoring(log, config)
	validation := validation.NewValidation()
	defer monitoring.Shutdown()

    server := app.NewApp(log, config, sqlDB, web, validation, redis)
	server.Run()
}
