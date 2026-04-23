FROM golang:1.25-alpine AS builder

WORKDIR /app

# Dependencies referenced by go.mod's `replace` directives.
COPY libs/common-go /app/libs/common-go
COPY services/user-auth-service /app/services/user-auth-service

WORKDIR /app/services/discussion-service

COPY services/discussion-service/go.mod .
COPY services/discussion-service/go.sum* .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY services/discussion-service/ .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /discussion-service ./cmd/server/

FROM alpine:3.20

RUN apk --no-cache add ca-certificates curl
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /discussion-service .
COPY --from=builder /app/services/discussion-service/migrations ./migrations
USER appuser

# Tenant provisioner names containers `discussion-{slug}` and wires them on
# port 50056 of the tenant network.
EXPOSE 50056

CMD ["./discussion-service"]
