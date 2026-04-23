FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy local module deps so go mod download can resolve the replace directive.
COPY proto/ ./proto/
COPY libs/common-go /app/libs/common-go

WORKDIR /app/services/incident-service

COPY services/incident-service/go.mod .
COPY services/incident-service/go.sum* .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY services/incident-service/ .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /incident-service ./cmd/server/

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates curl

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /incident-service .
COPY --from=builder /app/services/incident-service/migrations ./migrations

USER appuser

EXPOSE 8091 50061

CMD ["./incident-service"]
