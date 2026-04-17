-- Metrics Service Database Schema
-- Epic 9: US-9.2 Usage Metrics Dashboard

-- API request metrics table (AC3: API requests per minute)
CREATE TABLE IF NOT EXISTS api_request_metrics (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(36),
    user_id VARCHAR(36),
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,
    status_code INTEGER NOT NULL,
    response_time_ms BIGINT NOT NULL,
    service_name VARCHAR(100),
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Partitioning by timestamp for performance
    CONSTRAINT valid_status_code CHECK (status_code >= 100 AND status_code < 600),
    CONSTRAINT valid_response_time CHECK (response_time_ms >= 0)
);

-- Partition by time (we'll create partitions for last 30 days)
CREATE INDEX idx_api_metrics_timestamp ON api_request_metrics(timestamp DESC);
CREATE INDEX idx_api_metrics_tenant ON api_request_metrics(tenant_id, timestamp DESC);
CREATE INDEX idx_api_metrics_status ON api_request_metrics(status_code, timestamp);
CREATE INDEX idx_api_metrics_path ON api_request_metrics(path, timestamp);

-- Error tracking table (AC4: Error rate last 24 hours)
CREATE TABLE IF NOT EXISTS error_metrics (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(36),
    error_type VARCHAR(100) NOT NULL,
    error_message TEXT,
    stack_trace TEXT,
    service_name VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL, -- info, warning, error, critical
    count INTEGER NOT NULL DEFAULT 1,
    first_seen TIMESTAMP NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMP NOT NULL DEFAULT NOW(),
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_error_metrics_timestamp ON error_metrics(timestamp DESC);
CREATE INDEX idx_error_metrics_tenant ON error_metrics(tenant_id, timestamp DESC);
CREATE INDEX idx_error_metrics_type ON error_metrics(error_type);
CREATE INDEX idx_error_metrics_service ON error_metrics(service_name, timestamp);
CREATE INDEX idx_error_metrics_severity ON error_metrics(severity, timestamp);

-- Tenant metrics summary table (AC1, AC2, AC5: Tenant-level breakdown)
CREATE TABLE IF NOT EXISTS tenant_metrics_summary (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id VARCHAR(36) UNIQUE NOT NULL,
    tenant_name VARCHAR(255),
    tier VARCHAR(50),

    -- AC1: Active tenants count
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- AC2: Total users, courses, storage
    user_count INTEGER NOT NULL DEFAULT 0,
    course_count INTEGER NOT NULL DEFAULT 0,
    storage_used_bytes BIGINT NOT NULL DEFAULT 0,
    storage_quota_bytes BIGINT NOT NULL DEFAULT 0,
    storage_usage_percentage DECIMAL(5,2),

    -- AC3: API requests
    api_requests_last_minute BIGINT NOT NULL DEFAULT 0,
    api_requests_last_hour BIGINT NOT NULL DEFAULT 0,
    api_requests_last_24h BIGINT NOT NULL DEFAULT 0,
    api_requests_total BIGINT NOT NULL DEFAULT 0,

    -- AC4: Error rate
    errors_last_hour BIGINT NOT NULL DEFAULT 0,
    errors_last_24h BIGINT NOT NULL DEFAULT 0,
    error_rate_percentage DECIMAL(5,2) NOT NULL DEFAULT 0,

    -- Performance metrics
    avg_response_time_ms DECIMAL(10,2),
    p95_response_time_ms DECIMAL(10,2),
    p99_response_time_ms DECIMAL(10,2),

    -- Uptime metrics
    uptime_percentage DECIMAL(5,2) NOT NULL DEFAULT 100,
    last_downtime TIMESTAMP,

    -- Activity tracking
    active_users_count INTEGER NOT NULL DEFAULT 0,
    last_active_at TIMESTAMP,

    -- Update tracking (AC6: Metrics updated every 30 seconds)
    last_updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_summary_tenant ON tenant_metrics_summary(tenant_id);
CREATE INDEX idx_tenant_summary_active ON tenant_metrics_summary(is_active);
CREATE INDEX idx_tenant_summary_tier ON tenant_metrics_summary(tier);
CREATE INDEX idx_tenant_summary_updated ON tenant_metrics_summary(last_updated_at DESC);

-- System-wide metrics summary table
CREATE TABLE IF NOT EXISTS system_metrics_summary (
    id VARCHAR(36) PRIMARY KEY,
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,

    -- AC1: Active tenants count
    active_tenants_count INTEGER NOT NULL DEFAULT 0,
    total_tenants_count INTEGER NOT NULL DEFAULT 0,

    -- AC2: Total users, courses, storage
    total_users BIGINT NOT NULL DEFAULT 0,
    total_courses BIGINT NOT NULL DEFAULT 0,
    total_storage_bytes BIGINT NOT NULL DEFAULT 0,
    total_storage_quota_bytes BIGINT NOT NULL DEFAULT 0,
    storage_usage_percentage DECIMAL(5,2),

    -- AC3: API requests per minute
    api_requests_per_minute DECIMAL(10,2),
    total_api_requests BIGINT NOT NULL DEFAULT 0,

    -- AC4: Error rate
    error_rate_percentage DECIMAL(5,2) NOT NULL DEFAULT 0,
    total_errors BIGINT NOT NULL DEFAULT 0,

    -- Uptime metrics
    uptime_percentage DECIMAL(5,2) NOT NULL DEFAULT 100,
    uptime_seconds BIGINT NOT NULL DEFAULT 0,
    downtime_seconds BIGINT NOT NULL DEFAULT 0,

    -- Performance metrics
    avg_response_time_ms DECIMAL(10,2),
    p95_response_time_ms DECIMAL(10,2),
    p99_response_time_ms DECIMAL(10,2),

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_system_summary_period ON system_metrics_summary(period_start DESC, period_end DESC);
CREATE INDEX idx_system_summary_created ON system_metrics_summary(created_at DESC);

-- Real-time metrics buffer (AC6: Updated every 30 seconds)
CREATE TABLE IF NOT EXISTS realtime_metrics_buffer (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(36),
    metric_type VARCHAR(100) NOT NULL, -- api_request, error, connection, user_activity
    metric_value DECIMAL(20,4) NOT NULL,
    labels JSONB,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),

    -- TTL: Auto-delete after 5 minutes (only keep recent data)
    expires_at TIMESTAMP NOT NULL DEFAULT NOW() + INTERVAL '5 minutes'
);

CREATE INDEX idx_realtime_buffer_timestamp ON realtime_metrics_buffer(timestamp DESC);
CREATE INDEX idx_realtime_buffer_tenant ON realtime_metrics_buffer(tenant_id, timestamp DESC);
CREATE INDEX idx_realtime_buffer_type ON realtime_metrics_buffer(metric_type, timestamp);
CREATE INDEX idx_realtime_buffer_expires ON realtime_metrics_buffer(expires_at);

-- Alerts table (AC7: Alerts if error rate >1% or uptime <99.5%)
CREATE TABLE IF NOT EXISTS alerts (
    id VARCHAR(36) PRIMARY KEY,
    alert_type VARCHAR(100) NOT NULL, -- error_rate_high, uptime_low, storage_quota_exceeded, etc.
    severity VARCHAR(20) NOT NULL, -- info, warning, critical
    title VARCHAR(500) NOT NULL,
    message TEXT NOT NULL,

    -- Alert thresholds (AC7)
    threshold_value DECIMAL(10,4),
    current_value DECIMAL(10,4),

    -- Scope
    tenant_id VARCHAR(36),
    tenant_name VARCHAR(255),
    service_name VARCHAR(100),

    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    triggered_at TIMESTAMP NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP,
    acknowledged_at TIMESTAMP,
    acknowledged_by VARCHAR(255),

    -- Metadata
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alerts_type ON alerts(alert_type);
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_tenant ON alerts(tenant_id);
CREATE INDEX idx_alerts_active ON alerts(is_active, triggered_at DESC);
CREATE INDEX idx_alerts_triggered ON alerts(triggered_at DESC);

-- Alert history (for tracking alert patterns)
CREATE TABLE IF NOT EXISTS alert_history (
    id BIGSERIAL PRIMARY KEY,
    alert_id VARCHAR(36) NOT NULL,
    event_type VARCHAR(50) NOT NULL, -- triggered, acknowledged, resolved, escalated
    event_data JSONB,
    user_id VARCHAR(36),
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_history_alert ON alert_history(alert_id, timestamp DESC);
CREATE INDEX idx_alert_history_timestamp ON alert_history(timestamp DESC);

-- Metric aggregations (pre-computed for performance)
CREATE TABLE IF NOT EXISTS metric_aggregations (
    id VARCHAR(36) PRIMARY KEY,
    metric_name VARCHAR(100) NOT NULL,
    aggregation_type VARCHAR(50) NOT NULL, -- hourly, daily, weekly
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    tenant_id VARCHAR(36),

    -- Aggregated values
    min_value DECIMAL(20,4),
    max_value DECIMAL(20,4),
    avg_value DECIMAL(20,4),
    sum_value DECIMAL(20,4),
    count_value BIGINT,
    p50_value DECIMAL(20,4),
    p95_value DECIMAL(20,4),
    p99_value DECIMAL(20,4),

    labels JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(metric_name, aggregation_type, period_start, tenant_id)
);

CREATE INDEX idx_aggregations_metric ON metric_aggregations(metric_name, aggregation_type, period_start DESC);
CREATE INDEX idx_aggregations_tenant ON metric_aggregations(tenant_id, period_start DESC);
CREATE INDEX idx_aggregations_period ON metric_aggregations(period_start DESC, period_end DESC);

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create function to clean up old realtime buffer data
CREATE OR REPLACE FUNCTION cleanup_realtime_buffer()
RETURNS void AS $$
BEGIN
    DELETE FROM realtime_metrics_buffer WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Record migration
INSERT INTO schema_migrations (version) VALUES ('001_init') ON CONFLICT DO NOTHING;
