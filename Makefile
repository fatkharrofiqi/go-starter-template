.PHONY: migrateup migratedown migrateschema run build install seed

# Variables to construct DSN from config.yml using yq
DB_DSN = $(shell yq e '.database.dsn' config.yml)
# Construct the DSN
DSN = "${DB_DSN}"

install:
	go mod tidy

migrateup:
	migrate -path db/migration -database $(DSN) -verbose up

STEP ?= 1
migratedown:
	migrate -path db/migration -database $(DSN) -verbose down ${STEP}

migrateschema:
	@if [ -z "$(name)" ]; then \
		echo "Error: 'name' variable is required. Usage: make migrateschema name=<migration_name>"; \
		exit 1; \
	fi
	migrate create -ext sql -dir db/migration -seq $(name)

run:
	go run cmd/app/main.go

build:
	go build -o main cmd/app/main.go

seed:
	go run cmd/seed/main.go