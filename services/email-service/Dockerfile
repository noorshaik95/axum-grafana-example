# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy proto files
COPY proto/ ./proto/

# Copy service files
COPY services/email-service/ ./services/email-service/

WORKDIR /app/services/email-service

# Download dependencies (if any)
RUN go mod download 2>/dev/null || true

# Build the application (placeholder — no main.go yet)
# TODO: Add cmd/server/main.go with gRPC server implementation
RUN echo 'package main; import ("fmt";"os";"time"); func main() { fmt.Println("email-service: placeholder — not yet implemented"); for { fmt.Fprintf(os.Stdout, "email-service running (stub)\n"); time.Sleep(30 * time.Second) } }' > main_placeholder.go && \
    CGO_ENABLED=0 GOOS=linux go build -o /email-service main_placeholder.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /email-service .

# Copy migrations
COPY --from=builder /app/services/email-service/migrations ./migrations

# Expose ports
EXPOSE 50052 8082

# Run the service
CMD ["./email-service"]
