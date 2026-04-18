FROM golang:1.25-alpine AS builder

RUN apk add --no-cache protobuf protobuf-dev git make

WORKDIR /app

COPY proto /app/proto
COPY libs/common-go /app/libs/common-go

WORKDIR /app/services/assignment-grading-service

# go.mod/go.sum before source so module download isn't busted by source changes
COPY services/assignment-grading-service/go.mod .
COPY services/assignment-grading-service/go.sum* .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && \
    go mod download

COPY services/assignment-grading-service/ .

RUN mkdir -p api/proto && \
    protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    -I /app/proto \
    /app/proto/assignment.proto && \
    mv -f assignment.pb.go assignment_grpc.pb.go api/proto/ 2>/dev/null || true

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /server cmd/server/main.go

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /server .
COPY --from=builder /app/services/assignment-grading-service/migrations ./migrations

RUN mkdir -p /app/storage && chown -R appuser:appgroup /app/storage

USER appuser

EXPOSE 50053

CMD ["./server"]
