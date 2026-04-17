use crate::observability::METRICS;
use axum::response::IntoResponse;
use prometheus::TextEncoder;

pub async fn metrics_handler() -> impl IntoResponse {
    let encoder = TextEncoder::new();
    let metric_families = METRICS.registry.gather();
    let metrics_text = encoder.encode_to_string(&metric_families).unwrap();

    ([("Content-Type", "text/plain; version=0.0.4")], metrics_text)
}
