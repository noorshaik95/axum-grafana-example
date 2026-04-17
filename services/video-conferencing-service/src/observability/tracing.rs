use opentelemetry::trace::TracerProvider;
use opentelemetry::KeyValue;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{trace as sdktrace, Resource};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

pub fn init_tracing(service_name: &str, tempo_endpoint: &str) -> anyhow::Result<()> {
    // Create OTLP exporter
    let exporter = opentelemetry_otlp::SpanExporter::builder()
        .with_tonic()
        .with_endpoint(tempo_endpoint)
        .build()?;

    // Create tracer provider
    let tracer_provider = sdktrace::SdkTracerProvider::builder()
        .with_batch_exporter(exporter)
        .with_resource(Resource::builder().with_attributes(vec![KeyValue::new(
            "service.name",
            service_name.to_string(),
        )]).build())
        .with_sampler(sdktrace::Sampler::AlwaysOn)
        .build();

    let tracer = tracer_provider.tracer(service_name.to_string());

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
                .with_span_list(true),
        )
        .init();

    tracing::info!("OpenTelemetry tracing initialized for {}", service_name);
    Ok(())
}

pub fn shutdown_tracing() {
    // OpenTelemetry 0.29 handles shutdown via drop of the SdkTracerProvider
    tracing::info!("Shutting down tracing");
}
