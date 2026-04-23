FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY proto/ ./proto/
COPY libs/common-go /app/libs/common-go

WORKDIR /app/services/ai-service

COPY services/ai-service/go.mod .
COPY services/ai-service/go.sum* .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY services/ai-service/ .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /ai-service ./cmd/server/

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates curl

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /ai-service .

USER appuser

EXPOSE 50064

CMD ["./ai-service"]
