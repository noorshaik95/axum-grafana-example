FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY libs/common-go /app/libs/common-go

WORKDIR /app/services/feature-flag-service

COPY services/feature-flag-service/go.mod .
COPY services/feature-flag-service/go.sum* .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY services/feature-flag-service/ .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /feature-flag-service ./cmd/server/

FROM alpine:3.20

RUN apk --no-cache add ca-certificates curl
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /feature-flag-service .
COPY --from=builder /app/services/feature-flag-service/migrations ./migrations

USER appuser

EXPOSE 8092 50062

CMD ["./feature-flag-service"]
