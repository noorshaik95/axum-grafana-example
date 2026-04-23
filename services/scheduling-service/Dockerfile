FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY libs/common-go /app/libs/common-go

WORKDIR /app/services/scheduling-service

COPY services/scheduling-service/go.mod .
COPY services/scheduling-service/go.sum* .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY services/scheduling-service/ .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /scheduling-service ./cmd/server/

FROM alpine:3.20

RUN apk --no-cache add ca-certificates curl
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /scheduling-service .
COPY --from=builder /app/services/scheduling-service/migrations ./migrations

USER appuser

# Tenant service — one container per tenant provisioned as "scheduling-{slug}".
EXPOSE 8093 50063

CMD ["./scheduling-service"]
