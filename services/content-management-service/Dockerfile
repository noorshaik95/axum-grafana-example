# Multi-stage build for content-management-service
# Stage 1: Builder
FROM rust:latest as builder

# Install protobuf compiler and librdkafka for Kafka support
RUN apt-get update && apt-get install -y \
    protobuf-compiler \
    pkg-config \
    librdkafka-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy common-rust library to match the path in Cargo.toml (../../libs/common-rust from /build/services/content-management-service)
COPY libs/common-rust /build/libs/common-rust

# Copy proto files to match build.rs expectations
COPY proto /build/proto

# Set up the service directory structure
WORKDIR /build/services/content-management-service

# Copy Cargo files first for better caching
COPY services/content-management-service/Cargo.toml ./Cargo.toml
COPY services/content-management-service/build.rs ./build.rs

# Create dummy main.rs to build dependencies
RUN mkdir -p src && \
    echo "fn main() {}" > src/main.rs && \
    cargo build --release && \
    rm -rf src

# Now copy actual source code
COPY services/content-management-service/src ./src
COPY services/content-management-service/migrations ./migrations

# Build the application with cache mounts
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/build/target \
    cargo build --release && \
    cp target/release/content-management-service /usr/local/bin/content-management-service

# Stage 2: Runtime
FROM debian:sid-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    ffmpeg \
    libssl3 \
    libpq5 \
    librdkafka1 \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user for security
RUN useradd -m -u 1000 cms && \
    mkdir -p /app && \
    chown -R cms:cms /app

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /usr/local/bin/content-management-service /usr/local/bin/content-management-service

# Switch to non-root user
USER cms

# Expose ports
# 50052: gRPC service
# 8082: HTTP health check
# 9092: Prometheus metrics
EXPOSE 50052 8082 9092

# Run the service
CMD ["content-management-service"]
