package main

import (
	"go-starter-template/db/seeder"
	"go-starter-template/internal/config/database"
	"go-starter-template/internal/config/env"
	"go-starter-template/internal/config/logger"
)

func main() {
	config := env.NewConfig()
	log := logger.NewLogger(config)
	db := database.NewDatabase(log, config)

	seeder.Seed(db)
}
