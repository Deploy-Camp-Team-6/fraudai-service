.PHONY: help build run test coverage lint clean

BINARY_NAME=go-api

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build the binary"
	@echo "  run         Build and run the binary"
	@echo "  test        Run tests"
	@echo "  coverage    Run tests with coverage"
	@echo "  lint        Run linter"
	@echo "  clean       Clean up build artifacts"
	@echo "  docker      Build docker image"
	@echo "  db-up       Start docker-compose db"
	@echo "  db-down     Stop docker-compose db"
	@echo "  db-migrate  Run database migrations"
	@echo "  sqlc        Generate sqlc code"


build:
	@echo "Building binary..."
	@go build -o $(BINARY_NAME) ./cmd/api

run: build
	@echo "Running binary..."
	@./$(BINARY_NAME)

test:
	@echo "Running tests..."
	@go test -v ./...

coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

lint:
	@echo "Running linter..."
	@golangci-lint run

clean:
	@echo "Cleaning up..."
	@go clean
	@rm -f $(BINARY_NAME) coverage.out

docker:
	@echo "Building docker image..."
	@docker build -t $(BINARY_NAME) .

db-up:
	@echo "Starting database..."
	@docker-compose -f docker-compose.dev.yml up -d postgres redis

db-down:
	@echo "Stopping database..."
	@docker-compose -f docker-compose.dev.yml down

db-migrate:
	@echo "Running database migrations..."
	@migrate -path ./migrations -database "$$PG_DSN" up

sqlc:
	@echo "Generating sqlc code..."
	@sqlc generate
