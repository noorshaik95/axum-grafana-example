FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy local module deps that go.mod references
COPY proto/ ./proto/
COPY libs/common-go /app/libs/common-go

WORKDIR /app/services/metrics-service

# go.mod/go.sum before source — download only reruns when dependencies change
COPY services/metrics-service/go.mod .
COPY services/metrics-service/go.sum* .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY services/metrics-service/ .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /metrics-service ./cmd/server/

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates curl

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /metrics-service .
COPY --from=builder /app/services/metrics-service/migrations ./migrations

USER appuser

EXPOSE 8087 50057

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8087/health || exit 1

CMD ["./metrics-service"]
