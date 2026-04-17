# Build stage
FROM golang:1.25-alpine AS builder

# Install protoc and required tools
RUN apk add --no-cache protobuf protobuf-dev git make

# Install Go protobuf plugins (pinned versions compatible with Go 1.23 and grpc v1.62)
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

WORKDIR /build

# Copy proto files from workspace root
COPY proto /build/proto

# Copy shared libs directory to parent of build directory so ../../libs/common-go resolves correctly
COPY libs /libs

# Copy go mod files
COPY services/assignment-grading-service/go.mod ./

# Copy source code first (needed for go mod tidy to detect all imports)
COPY services/assignment-grading-service/ .

# Generate go.sum and download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod tidy && go mod download

# Generate protobuf code
RUN mkdir -p api/proto && \
    protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    -I /build/proto \
    /build/proto/assignment.proto && \
    mv -f assignment.pb.go assignment_grpc.pb.go api/proto/ 2>/dev/null || true

# Build the application
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/server .
COPY --from=builder /build/migrations ./migrations

# Create directory for file storage
RUN mkdir -p /app/storage

# Expose gRPC port
EXPOSE 50053

# Run the service
CMD ["./server"]
