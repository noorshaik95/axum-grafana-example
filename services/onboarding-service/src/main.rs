mod kafka;
mod state;
mod workflow;

use restate_sdk::prelude::*;
use tracing::info;
use tracing_subscriber::EnvFilter;
use workflow::{OnboardingWorkflow, OnboardingWorkflowImpl};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Initialize tracing
    tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| EnvFilter::new("info,onboarding_service=debug")),
        )
        .json()
        .init();

    info!("Starting onboarding service");

    let endpoint = Endpoint::builder()
        .bind(OnboardingWorkflowImpl.serve())
        .build();

    HttpServer::new(endpoint)
        .listen_and_serve("0.0.0.0:9080".parse()?)
        .await;

    Ok(())
}
