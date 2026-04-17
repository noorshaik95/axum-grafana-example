use opentelemetry::{global, KeyValue};
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{runtime, trace as sdktrace, Resource};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

pub fn init_tracing(service_name: &str, tempo_endpoint: &str) -> anyhow::Result<()> {
    // Create OTLP tracer
    let tracer = opentelemetry_otlp::new_pipeline()
        .tracing()
        .with_exporter(
            opentelemetry_otlp::new_exporter()
                .tonic()
                .with_endpoint(tempo_endpoint),
        )
        .with_trace_config(
            sdktrace::Config::default()
                .with_resource(Resource::new(vec![KeyValue::new(
                    "service.name",
                    service_name.to_string(),
                )]))
                .with_sampler(sdktrace::Sampler::AlwaysOn),
        )
        .install_batch(runtime::Tokio)?;

    global::set_tracer_provider(tracer.provider().unwrap());

    // Set up tracing subscriber with OpenTelemetry layer and JSON formatting
    tracing_subscriber::registry()
        .with(EnvFilter::try_from_default_env().unwrap_or_else(|_| {
            "video_conferencing_service=debug,tower_http=debug,axum=debug,sqlx=info".into()
        }))
        .with(tracing_opentelemetry::layer().with_tracer(tracer))
        .with(
            tracing_subscriber::fmt::layer()
                .json()
                .with_current_span(true)
                .with_span_list(true)
        )
        .init();

    tracing::info!("OpenTelemetry tracing initialized for {}", service_name);
    Ok(())
}

pub fn shutdown_tracing() {
    global::shutdown_tracer_provider();
}
