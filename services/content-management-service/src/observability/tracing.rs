use common_rust::observability::TracingConfig;

/// Initialize distributed tracing with OpenTelemetry and structured logging
/// Delegates to common-rust shared observability library for standardized setup.
pub fn init_tracing(service_name: &str, _otlp_endpoint: &str) -> anyhow::Result<()> {
    let config = TracingConfig {
        service_name: service_name.to_string(),
        otlp_endpoint: Some(_otlp_endpoint.to_string()),
        log_level: std::env::var("RUST_LOG").unwrap_or_else(|_| "info".to_string()),
        json_format: true,
    };

    common_rust::observability::init_tracing(config)?;

    tracing::info!(service_name = %service_name, "Tracing initialized via common-rust");

    Ok(())
}

/// Shutdown tracing and flush pending spans
pub fn shutdown_tracing() {
    common_rust::observability::shutdown_tracing();
}
