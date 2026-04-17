# Build stage
FROM rust:latest AS builder

WORKDIR /build

# Install dependencies
RUN apt-get update && apt-get install -y \
    protobuf-compiler \
    libprotobuf-dev \
    pkg-config \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy proto files first
COPY proto/ /build/proto/

# Copy service files
COPY services/video-conferencing-service/ /build/services/video-conferencing-service/

# Build the service
WORKDIR /build/services/video-conferencing-service
RUN cargo build --release

# Runtime stage
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    libssl3 \
    && rm -rf /var/lib/apt/lists/*

# Copy binary and migrations
COPY --from=builder /build/services/video-conferencing-service/target/release/video-conferencing-service /app/
COPY --from=builder /build/services/video-conferencing-service/migrations/ /app/migrations/

# Create non-root user
RUN useradd -m -u 1000 appuser && \
    chown -R appuser:appuser /app

USER appuser

# Expose ports
# gRPC: 50052
# WebSocket/HTTP: 8082
# Metrics: 9092
EXPOSE 50055 8086 9092

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8086/health || exit 1

CMD ["/app/video-conferencing-service"]
