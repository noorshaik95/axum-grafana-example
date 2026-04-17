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
COPY services/user-auth-service/go.mod services/user-auth-service/go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Generate protobuf code
RUN mkdir -p api/proto && \
    protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    -I /build/proto \
    /build/proto/user.proto \
    /build/proto/auth.proto && \
    mkdir -p api/proto/authpb && \
    mv -f user.pb.go user_grpc.pb.go api/proto/ 2>/dev/null || true && \
    mv -f auth.pb.go auth_grpc.pb.go api/proto/authpb/ 2>/dev/null || true

# Copy source code
COPY services/user-auth-service/ .

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

# Expose gRPC port
EXPOSE 50051

# Run the service
CMD ["./server"]
