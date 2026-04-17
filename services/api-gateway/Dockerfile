# Build Stage
FROM rust:1.90 as builder

# Install protobuf compiler for proto compilation
RUN apt-get update && apt-get install -y protobuf-compiler && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy common-rust library to match the path in Cargo.toml (../../libs/common-rust from /app)
COPY libs/common-rust /libs/common-rust

# Copy proto files for proto compilation (needed for build.rs)
COPY proto ./proto
COPY services/api-gateway/build.rs ./build.rs

# Copy Cargo files first for better caching
COPY services/api-gateway/Cargo.toml ./Cargo.toml
COPY services/api-gateway/benches ./benches

# Create dummy main.rs to build dependencies
RUN mkdir -p src && \
    echo "fn main() {}" > src/main.rs && \
    cargo build --release && \
    rm -rf src

# Now copy actual source code
COPY services/api-gateway/src ./src

# Build the application (this will compile protos via build.rs)
# Use cache mounts to speed up subsequent builds
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/app/target \
    cargo build --release && \
    cp target/release/api-gateway /app/api-gateway

# Runtime Stage
FROM debian:bookworm-slim

# Install curl for health checks
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the compiled binary
COPY --from=builder /app/api-gateway /usr/local/bin/app

# Create config and openapi directories
RUN mkdir -p /app/config /app/services/api-gateway/openapi

# Copy default gateway configuration file
COPY config/gateway-config.docker.yaml /app/config/gateway-config.yaml

# Copy OpenAPI documentation
COPY services/api-gateway/openapi /app/services/api-gateway/openapi

EXPOSE 8080
CMD ["app"]
