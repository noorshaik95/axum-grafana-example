# Multi-stage build for content-management-service
# Build context: repo root

# Stage 1: Builder
FROM rust:1.91 as builder

# cmake-build compiles librdkafka from source (avoids system version mismatch)
RUN apt-get update && apt-get install -y \
    protobuf-compiler \
    pkg-config \
    cmake \
    libssl-dev \
    libsasl2-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

COPY libs/common-rust /build/libs/common-rust
COPY proto /build/proto

WORKDIR /build/services/content-management-service

COPY services/content-management-service/Cargo.toml ./Cargo.toml
COPY services/content-management-service/build.rs ./build.rs

# target path must match the actual cargo output dir for cache hits to land
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/build/services/content-management-service/target \
    mkdir -p src && \
    echo "fn main() {}" > src/main.rs && \
    cargo build --release && \
    rm -rf src

COPY services/content-management-service/src ./src
COPY services/content-management-service/migrations ./migrations

RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/build/services/content-management-service/target \
    cargo build --release && \
    cp target/release/content-management-service /usr/local/bin/content-management-service

# Stage 2: Runtime
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    ffmpeg \
    libssl3t64 \
    libpq5 \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /app && chown -R 1000:1000 /app

WORKDIR /app

COPY --from=builder /usr/local/bin/content-management-service /usr/local/bin/content-management-service

USER 1000

EXPOSE 50052 8082 9092

CMD ["content-management-service"]
