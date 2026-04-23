FROM golang:1.25-alpine AS builder

RUN apk add --no-cache protobuf protobuf-dev git make

WORKDIR /build

COPY proto /build/proto
COPY libs /libs
COPY services/admin-auth-service/go.mod services/admin-auth-service/go.sum* ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.10 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
    go mod download

RUN mkdir -p api/proto/adminauthpb && \
    protoc --go_out=. --go_opt=paths=source_relative \
           --go-grpc_out=. --go-grpc_opt=paths=source_relative \
           -I /build/proto \
           /build/proto/admin_auth.proto && \
    mv -f admin_auth.pb.go admin_auth_grpc.pb.go api/proto/adminauthpb/ 2>/dev/null || true

COPY services/admin-auth-service/ .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -a -o server cmd/server/main.go

# ---- Runtime ----
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /build/server /app/server
COPY --from=builder /build/migrations /app/migrations

USER app

EXPOSE 50060 8090

ENTRYPOINT ["/app/server"]
