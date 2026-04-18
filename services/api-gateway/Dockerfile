# Build Stage
FROM rust:1.91 AS builder

RUN apt-get update && apt-get install -y protobuf-compiler && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY libs/common-rust /libs/common-rust
COPY proto ./proto
COPY services/api-gateway/build.rs ./build.rs
COPY services/api-gateway/Cargo.toml ./Cargo.toml
COPY services/api-gateway/benches ./benches

# Pre-build deps with full cache mounts so downloads and compilations persist
# on the host between builds. Layer cache handles the no-change case;
# BuildKit cache mounts handle the Cargo.toml-changed case.
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/app/target \
    mkdir -p src && \
    echo "fn main() {}" > src/main.rs && \
    cargo build --release && \
    rm -rf src

COPY services/api-gateway/src ./src

RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/root/.cargo/git \
    --mount=type=cache,target=/app/target \
    cargo build --release && \
    cp target/release/api-gateway /app/api-gateway

# Runtime Stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/api-gateway /usr/local/bin/app

RUN mkdir -p /app/config /app/services/api-gateway/openapi

COPY config/gateway-config.docker.yaml /app/config/gateway-config.yaml
COPY services/api-gateway/openapi /app/services/api-gateway/openapi

EXPOSE 8080
CMD ["app"]
