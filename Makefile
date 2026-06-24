.PHONY: build test lint docker-up docker-down migrate-up migrate-down dev clean

APP_NAME := webhook-api
DOCKER_COMPOSE := docker compose
MIGRATE_DIR := file://migrations
DB_URL := postgres://webhook:webhook@localhost:5432/webhook?sslmode=disable

build:
    go build -o bin/api ./cmd/api
    go build -o bin/migrate ./cmd/migrate

test:
    go test -race -count=1 ./...

test-coverage:
    go test -race -coverprofile=coverage.txt -covermode=atomic ./...
    go tool cover -html=coverage.txt -o coverage.html

lint:
    golangci-lint run ./...

docker-up:
    $(DOCKER_COMPOSE) up -d

docker-down:
    $(DOCKER_COMPOSE) down

docker-logs:
    $(DOCKER_COMPOSE) logs -f

migrate-up:
    go run ./cmd/migrate -direction up -db-url $(DB_URL) -migrations $(MIGRATE_DIR)

migrate-down:
    go run ./cmd/migrate -direction down -db-url $(DB_URL) -migrations $(MIGRATE_DIR)

dev:
    go run ./cmd/api

clean:
    rm -rf bin/ coverage.txt coverage.html

wait-db:
    @echo "Waiting for PostgreSQL..."
    @until pg_isready -h localhost -p 5432 -U webhook > /dev/null 2>&1; do \
        sleep 1; \
    done
    @echo "PostgreSQL is ready"

setup: docker-up wait-db migrate-up
    @echo "Local environment ready. Run 'make dev' to start the API."