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

    let endpoint = Endpoint::builder()
        .bind(OnboardingWorkflowImpl.serve())
        .build();

    HttpServer::new(endpoint)
        .listen_and_serve("0.0.0.0:9080".parse()?)
        .await;

    Ok(())
}
