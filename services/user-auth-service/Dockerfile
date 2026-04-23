FROM golang:1.25-alpine AS builder

RUN apk add --no-cache protobuf protobuf-dev git make

WORKDIR /build

COPY proto /build/proto
COPY libs /libs

COPY services/user-auth-service/go.mod services/user-auth-service/go.sum* ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
    go mod download

RUN mkdir -p api/proto api/proto/authpb && \
    protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    -I /build/proto \
    /build/proto/user.proto \
    /build/proto/auth.proto && \
    mv -f user.pb.go user_grpc.pb.go api/proto/ 2>/dev/null || true && \
    mv -f auth.pb.go auth_grpc.pb.go api/proto/authpb/ 2>/dev/null || true

COPY services/user-auth-service/ .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /build/server .
COPY --from=builder /build/migrations ./migrations

USER appuser

EXPOSE 50051

CMD ["./server"]
