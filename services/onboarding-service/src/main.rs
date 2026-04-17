mod health;
mod kafka;
mod state;
mod workflow;

use common_rust::observability::{init_tracing, TracingConfig};
use restate_sdk::prelude::*;
use tracing::info;
use workflow::{OnboardingWorkflow, OnboardingWorkflowImpl};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Initialize tracing via common-rust observability
    init_tracing(TracingConfig {
        service_name: "onboarding-service".to_string(),
        otlp_endpoint: std::env::var("OTEL_EXPORTER_OTLP_ENDPOINT").ok(),
        log_level: std::env::var("RUST_LOG")
            .unwrap_or_else(|_| "info,onboarding_service=debug".to_string()),
        json_format: true,
    })?;

    info!("Starting onboarding service");

    // Spawn health server on a separate port (Restate HttpServer owns 9080 and
    // does not expose /health, so Prometheus and Docker HEALTHCHECK use 9081).
    let health_addr = std::env::var("HEALTH_ADDR")
        .unwrap_or_else(|_| "0.0.0.0:9081".to_string());
    tokio::spawn(async move {
        if let Err(e) = health::start_health_server(&health_addr).await {
            tracing::error!("Health server error: {}", e);
        }
    });

    let endpoint = Endpoint::builder()
        .bind(OnboardingWorkflowImpl.serve())
        .build();

    HttpServer::new(endpoint)
        .listen_and_serve("0.0.0.0:9080".parse()?)
        .await;

    Ok(())
}
