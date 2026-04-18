FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy local module deps that go.mod references so go mod download can resolve them
COPY proto/ ./proto/
COPY libs/common-go /app/libs/common-go

WORKDIR /app/services/email-service

# Copy go.mod/go.sum first — download layer only busts when these change,
# not every time source code changes.
COPY services/email-service/go.mod .
COPY services/email-service/go.sum* .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy full source now that deps are cached
COPY services/email-service/ .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /email-service ./cmd/server/

# Runtime stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates curl

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /email-service .
COPY --from=builder /app/services/email-service/migrations ./migrations

USER appuser

EXPOSE 8086 50056

CMD ["./email-service"]
