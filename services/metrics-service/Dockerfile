# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy proto files
COPY proto/ ./proto/

# Copy service files
COPY services/metrics-service/ ./services/metrics-service/

WORKDIR /app/services/metrics-service

# Download dependencies (if any)
RUN go mod download 2>/dev/null || true

# Build the application (placeholder — no main.go yet)
# TODO: Add cmd/server/main.go with gRPC server implementation
RUN echo 'package main; import ("fmt";"os";"time"); func main() { fmt.Println("metrics-service: placeholder — not yet implemented"); for { fmt.Fprintf(os.Stdout, "metrics-service running (stub)\n"); time.Sleep(30 * time.Second) } }' > main_placeholder.go && \
    CGO_ENABLED=0 GOOS=linux go build -o /metrics-service main_placeholder.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /metrics-service .

# Copy migrations
COPY --from=builder /app/services/metrics-service/migrations ./migrations

# Expose ports
EXPOSE 50054 8084 9090

# Run the service
CMD ["./metrics-service"]
