use rdkafka::config::ClientConfig;
use rdkafka::producer::{FutureProducer, FutureRecord};
use std::time::Duration;
use tracing::{error, info};
use uuid::Uuid;

/// Emit a JSON event to a Kafka topic.
pub async fn emit_kafka(topic: &str, payload: serde_json::Value) -> anyhow::Result<()> {
    let brokers = std::env::var("KAFKA_BROKERS").unwrap_or_else(|_| "kafka:9092".to_string());

    let producer: FutureProducer = ClientConfig::new()
        .set("bootstrap.servers", &brokers)
        .set("message.timeout.ms", "5000")
        .create()
        .map_err(|e| anyhow::anyhow!("Failed to create Kafka producer: {e}"))?;

    let event_id = Uuid::new_v4().to_string();
    let timestamp = chrono::Utc::now().to_rfc3339();
    let payload_bytes = serde_json::to_vec(&payload)?;

    let record = FutureRecord::to(topic)
        .payload(&payload_bytes)
        .key(&event_id)
        .headers(
            rdkafka::message::OwnedHeaders::new()
                .insert(rdkafka::message::Header {
                    key: "event_id",
                    value: Some(event_id.as_bytes()),
                })
                .insert(rdkafka::message::Header {
                    key: "timestamp",
                    value: Some(timestamp.as_bytes()),
                })
                .insert(rdkafka::message::Header {
                    key: "topic",
                    value: Some(topic.as_bytes()),
                }),
        );

    match producer.send(record, Duration::from_secs(5)).await {
        Ok(_) => {
            info!(topic = topic, event_id = %event_id, "Kafka event emitted");
            Ok(())
        }
        Err((err, _)) => {
            error!(topic = topic, error = %err, "Failed to emit Kafka event");
            Err(anyhow::anyhow!("Kafka send failed: {err}"))
        }
    }
}
