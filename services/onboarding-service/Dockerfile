# Multi-stage Docker build for Onboarding Service (Rust + Restate)
# Build context: repo root

# Stage 1: Builder
FROM rust:1.91 AS builder

RUN apt-get update && apt-get install -y cmake protobuf-compiler librdkafka-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY libs/common-rust /libs/common-rust
COPY services/onboarding-service/Cargo.toml ./Cargo.toml

RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/app/target \
    mkdir -p src && echo "fn main() {}" > src/main.rs && \
    cargo build --release && \
    rm -rf src

COPY services/onboarding-service/src ./src

RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/app/target \
    cargo build --release && \
    cp target/release/onboarding-service /app/onboarding-service

# Stage 2: Runtime (ubuntu:24.04 has GLIBC 2.39, needed by time crate + rdkafka)
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y ca-certificates curl librdkafka1 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/onboarding-service /usr/local/bin/app
COPY services/onboarding-service/migrations/ /app/migrations/

EXPOSE 9080 9081

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -sf http://localhost:9081/health || exit 1

CMD ["app"]
