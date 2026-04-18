# Build stage
FROM rust:1.91 AS builder

RUN apt-get update && apt-get install -y \
    protobuf-compiler \
    libprotobuf-dev \
    pkg-config \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

COPY proto/ /build/proto/

# Copy manifest only — pre-build all deps before touching source so the
# expensive compile step is only repeated when Cargo.toml changes.
COPY services/video-conferencing-service/Cargo.toml /build/services/video-conferencing-service/Cargo.toml
COPY services/video-conferencing-service/Cargo.lock* /build/services/video-conferencing-service/

WORKDIR /build/services/video-conferencing-service

RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/build/services/video-conferencing-service/target \
    mkdir -p src && \
    echo "fn main() {}" > src/main.rs && \
    cargo build --release && \
    rm -rf src

COPY services/video-conferencing-service/ /build/services/video-conferencing-service/

RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/build/services/video-conferencing-service/target \
    cargo build --release && \
    cp target/release/video-conferencing-service /video-conferencing-service

# Runtime stage
FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    libssl3 \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 appuser && chown -R appuser:appuser /app

COPY --from=builder /video-conferencing-service /app/video-conferencing-service
COPY --from=builder /build/services/video-conferencing-service/migrations/ /app/migrations/

USER appuser

EXPOSE 50055 8086 9092

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8086/health || exit 1

CMD ["/app/video-conferencing-service"]
