.PHONY: build run test clean docker-up docker-down seed help

# Default target
help:
	@echo "RegStrava - Invoice Funding Registry"
	@echo ""
	@echo "Usage:"
	@echo "  make build       Build the application"
	@echo "  make run         Run the application locally"
	@echo "  make test        Run tests"
	@echo "  make docker-up   Start all services with Docker Compose"
	@echo "  make docker-down Stop all Docker services"
	@echo "  make seed        Seed the database with a test funder"
	@echo "  make clean       Clean build artifacts"
	@echo ""

# Build the application
build:
	go build -o regstrava ./cmd/server

# Run locally (requires PostgreSQL and Redis running)
run: build
	./regstrava

# Run tests
test:
	go test -v ./...

# Start Docker services
docker-up:
	docker-compose up -d

# Stop Docker services
docker-down:
	docker-compose down

# Start only infrastructure (postgres + redis)
infra-up:
	docker-compose up -d postgres redis

# Seed database with test data
seed:
	go run scripts/seed.go

# Clean build artifacts
clean:
	rm -f regstrava
	go clean

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Generate mocks (if needed)
generate:
	go generate ./...

# Full rebuild
rebuild: clean build
