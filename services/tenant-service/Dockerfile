# Build stage
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache git protobuf-dev make

# Set working directory
WORKDIR /app

# Copy proto files
COPY proto/ ./proto/

# Copy service files
COPY services/tenant-service/ ./services/tenant-service/

WORKDIR /app/services/tenant-service

# Install protoc-gen-go and protoc-gen-go-grpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Add go bin to PATH
ENV PATH="$PATH:/root/go/bin"

# Generate proto files
RUN mkdir -p api/proto
RUN protoc --go_out=./api/proto --go_opt=paths=source_relative \
    --go-grpc_out=./api/proto --go-grpc_opt=paths=source_relative \
    ../../proto/tenant.proto

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /tenant-service ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /tenant-service .

# Copy migrations
COPY --from=builder /app/services/tenant-service/migrations ./migrations

# Expose ports
EXPOSE 50053 9090

# Run the service
CMD ["./tenant-service"]
