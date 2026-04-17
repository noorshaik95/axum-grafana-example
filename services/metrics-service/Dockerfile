# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy proto files
COPY proto/ ./proto/

# Copy shared Go library
COPY libs/common-go/ ./libs/common-go/

# Copy service files
COPY services/metrics-service/ ./services/metrics-service/

WORKDIR /app/services/metrics-service

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /metrics-service ./cmd/server/

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /metrics-service .

# Copy migrations
COPY --from=builder /app/services/metrics-service/migrations ./migrations

# Expose ports: HTTP 8087, gRPC 50057
EXPOSE 8087 50057

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8087/health || exit 1

# Run the service
CMD ["./metrics-service"]
