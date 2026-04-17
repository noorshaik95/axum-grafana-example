.PHONY: proto build run test clean docker-build docker-run

# Proto generation
proto:
	@echo "Generating proto files..."
	protoc --go_out=./api/proto --go_opt=paths=source_relative \
		--go-grpc_out=./api/proto --go-grpc_opt=paths=source_relative \
		../../proto/tenant.proto
	@echo "Proto files generated successfully"

# Build
build: proto
	@echo "Building tenant-service..."
	go build -o bin/tenant-service ./cmd/server
	@echo "Build complete"

# Run
run: build
	@echo "Running tenant-service..."
	./bin/tenant-service

# Test
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Tests complete"

# Clean
clean:
	@echo "Cleaning..."
	rm -rf bin/ coverage.out coverage.html
	@echo "Clean complete"

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t tenant-service:latest -f Dockerfile ../..
	@echo "Docker image built"

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker run -p 50053:50053 -p 9093:9090 tenant-service:latest

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Code formatted"

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run
	@echo "Linting complete"
