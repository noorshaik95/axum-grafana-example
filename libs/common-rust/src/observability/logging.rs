#[cfg(feature = "observability")]
use serde::{Deserialize, Serialize};

/// Tracing configuration
#[cfg(feature = "observability")]
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TracingConfig {
    pub service_name: String,
    /// OTLP gRPC endpoint, e.g. "http://tempo:4317". When None, traces are not
    /// exported (local JSON logging only — useful for unit tests).
    pub otlp_endpoint: Option<String>,
    pub log_level: String,
    /// Emit logs as JSON (recommended in production for Loki ingestion).
    pub json_format: bool,
}

/// Initialise tracing with:
///   - OpenTelemetry OTLP export to Tempo (when `otlp_endpoint` is Some)
///   - `tracing-opentelemetry` layer so every log line carries `trace_id` / `span_id`
///   - JSON-formatted stdout logger (Promtail-friendly)
///
/// Call once at service startup, before the first `tracing::info!` macro.
#[cfg(feature = "observability")]
pub fn init_tracing(config: TracingConfig) -> Result<(), anyhow::Error> {
    use opentelemetry::KeyValue;
    use opentelemetry_otlp::WithExportConfig;
    use opentelemetry_sdk::{trace as sdktrace, Resource};
    use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new(&config.log_level));

    // --- JSON formatter layer (always on) -----------------------------------------
    let fmt_layer = tracing_subscriber::fmt::layer()
        .json()
        .with_current_span(true) // emits trace_id / span_id fields
        .with_span_list(false);

    // --- OTel / Tempo layer (only when endpoint is provided) ----------------------
    if let Some(endpoint) = config.otlp_endpoint.as_deref().filter(|e| !e.is_empty()) {
        let exporter = opentelemetry_otlp::SpanExporter::builder()
            .with_tonic()
            .with_endpoint(endpoint)
            .build()?;

        let tracer_provider = sdktrace::SdkTracerProvider::builder()
            .with_batch_exporter(exporter)
            .with_resource(
                Resource::builder()
                    .with_attributes(vec![KeyValue::new(
                        "service.name",
                        config.service_name.clone(),
                    )])
                    .build(),
            )
            .with_sampler(sdktrace::Sampler::AlwaysOn)
            .build();

        let tracer = tracer_provider.tracer(config.service_name.clone());

        // Set the global OTel provider so other code can use `opentelemetry::global`.
        opentelemetry::global::set_tracer_provider(tracer_provider);

        tracing_subscriber::registry()
            .with(env_filter)
            .with(tracing_opentelemetry::layer().with_tracer(tracer))
            .with(fmt_layer)
            .init();
    } else {
        // No OTLP endpoint — local logging only (trace_id will be absent).
        tracing_subscriber::registry()
            .with(env_filter)
            .with(fmt_layer)
            .init();
    }

    tracing::info!(
        service = %config.service_name,
        otlp = config.otlp_endpoint.as_deref().unwrap_or("disabled"),
        "Observability initialised"
    );

    Ok(())
}

/// Flush and shut down the global OTel tracer provider.
/// Call during graceful shutdown to ensure all spans are exported.
#[cfg(feature = "observability")]
pub fn shutdown_tracing() {
    opentelemetry::global::shutdown_tracer_provider();
    tracing::info!("OpenTelemetry tracer provider shut down");
}
