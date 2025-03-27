package main

import (
	"go-starter-template/db/seeder"
	"go-starter-template/internal/config"
	"go-starter-template/internal/config/env"
)

func main() {
	cfg := env.NewConfig()
	log := config.NewLogger(cfg)
	db := config.NewDatabase(cfg, log)

	seeder.Seed(db)
}
