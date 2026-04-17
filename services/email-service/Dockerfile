FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy proto files
COPY proto/ ./proto/

# Copy service files
COPY services/email-service/ ./services/email-service/

WORKDIR /app/services/email-service

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /email-service ./cmd/server/

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /email-service .

# Copy migrations
COPY --from=builder /app/services/email-service/migrations ./migrations

# Expose ports: HTTP 8086, gRPC 50056
EXPOSE 8086 50056

CMD ["./email-service"]
