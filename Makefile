.PHONY: migrateup migratedown migrateschema run build install

# Variables to construct DSN from config.yml using yq
DB_DSN = $(shell yq e '.database.dsn' config.yml)
# Construct the DSN
DSN = "${DB_DSN}"

install:
	go mod tidy

migrateup:
	migrate -path db/migration -database $(DSN) -verbose up

migratedown:
	migrate -path db/migration -database $(DSN) -verbose down

migrateschema:
	@if [ -z "$(name)" ]; then \
		echo "Error: 'name' variable is required. Usage: make migrateschema name=<migration_name>"; \
		exit 1; \
	fi
	migrate create -ext sql -dir db/migration -seq $(name)

run:
	go run cmd/main.go

build:
	go build -o main cmd/main.go