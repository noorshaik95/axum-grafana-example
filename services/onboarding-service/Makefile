.PHONY: help test test-unit test-integration test-coverage lint fmt build clean docker-build run-server run-worker

# Default target
help:
	@echo "Available targets:"
	@echo "  test              - Run all tests (unit + integration)"
	@echo "  test-unit         - Run unit tests only"
	@echo "  test-integration  - Run integration tests"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  lint              - Run linters"
	@echo "  fmt               - Format code"
	@echo "  build             - Build binaries"
	@echo "  clean             - Clean build artifacts"
	@echo "  docker-build      - Build Docker images"
	@echo "  run-server        - Run server locally"
	@echo "  run-worker        - Run worker locally"

# Test targets
test: test-unit test-integration
	@echo "✅ All tests passed"

test-unit:
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "✅ Unit tests passed"

test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...
	@echo "✅ Integration tests passed"

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'
	@echo "Coverage report generated: coverage.html"

# Linting
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run --timeout 5m
	@echo "✅ Linting passed"

fmt:
	@echo "Formatting code..."
	@gofmt -w .
	@echo "✅ Code formatted"

# Build targets
build: build-server build-worker
	@echo "✅ Build completed"

build-server:
	@echo "Building server..."
	@mkdir -p bin
	@go build -o bin/onboarding-server ./cmd/server/main.go
	@echo "✅ Server built: bin/onboarding-server"

build-worker:
	@echo "Building worker..."
	@mkdir -p bin
	@go build -o bin/onboarding-worker ./cmd/worker/main.go
	@echo "✅ Worker built: bin/onboarding-worker"

# Docker targets
docker-build:
	@echo "Building Docker images..."
	@docker build -f Dockerfile -t onboarding-service:latest ../..
	@docker build -f Dockerfile.worker -t onboarding-worker:latest ../..
	@echo "✅ Docker images built"

# Run targets
run-server:
	@echo "Starting onboarding server..."
	@go run ./cmd/server/main.go

run-worker:
	@echo "Starting onboarding worker..."
	@go run ./cmd/worker/main.go

# Clean targets
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "✅ Clean completed"

# Development targets
dev-setup:
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "✅ Development environment ready"

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies downloaded"

# Database targets
db-migrate:
	@echo "Running database migrations..."
	@go run ./migrations/migrate.go
	@echo "✅ Migrations completed"
