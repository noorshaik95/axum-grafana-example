FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git protobuf-dev make

WORKDIR /app

COPY proto/ ./proto/
COPY libs/common-go /app/libs/common-go

WORKDIR /app/services/tenant-service

COPY services/tenant-service/go.mod .
COPY services/tenant-service/go.sum* .

# Install proto tools and download deps in one cached step.
# Pinned versions match assignment-grading and user-auth services.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
    go mod download

ENV PATH="$PATH:/root/go/bin"

COPY services/tenant-service/ .

# Generate proto files
RUN mkdir -p api/proto && \
    protoc --go_out=./api/proto --go_opt=paths=source_relative \
    --go-grpc_out=./api/proto --go-grpc_opt=paths=source_relative \
    -I /app/proto \
    /app/proto/tenant.proto

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /tenant-service ./cmd/server

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /tenant-service .
COPY --from=builder /app/services/tenant-service/migrations ./migrations

USER appuser

EXPOSE 50053 9090

CMD ["./tenant-service"]
