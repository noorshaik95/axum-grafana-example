# Multi-stage Docker build for Onboarding Service (Rust)

# Stage 1: Builder
FROM rust:1.86 AS builder

RUN apt-get update && apt-get install -y cmake protobuf-compiler librdkafka-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy manifest first for dependency caching
COPY services/onboarding-service/Cargo.toml ./

# Create a dummy main.rs to build dependencies
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release 2>/dev/null || true
RUN rm -rf src

# Copy actual source
COPY services/onboarding-service/src ./src

# Touch main.rs to invalidate the dummy build
RUN touch src/main.rs

RUN cargo build --release

# Stage 2: Runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates curl librdkafka1 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/target/release/onboarding-service /usr/local/bin/app

# Copy migrations for reference
COPY services/onboarding-service/migrations/ /app/migrations/

# Create non-root user
RUN groupadd -g 1000 appuser && \
    useradd -u 1000 -g appuser -m appuser
USER appuser

EXPOSE 9080

HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -sf http://localhost:9080/health || exit 1

CMD ["app"]
