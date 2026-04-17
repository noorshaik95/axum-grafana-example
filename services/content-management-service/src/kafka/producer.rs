use rdkafka::config::ClientConfig;
use rdkafka::producer::{FutureProducer, FutureRecord};
use serde::Serialize;
use std::time::Duration;
use tracing::{error, info, instrument};
use uuid::Uuid;

/// Kafka event producer for content lifecycle events
#[derive(Clone)]
pub struct KafkaProducer {
    producer: FutureProducer,
}

#[derive(Debug, Serialize)]
pub struct ContentUploadedEvent {
    pub content_id: Uuid,
    pub tenant_id: Uuid,
    pub course_id: Option<Uuid>,
    pub uploader_id: Uuid,
    pub filename: String,
    pub content_type: String,
}

#[derive(Debug, Serialize)]
pub struct ContentDeletedEvent {
    pub content_id: Uuid,
    pub tenant_id: Uuid,
    pub course_id: Option<Uuid>,
}

impl KafkaProducer {
    pub fn new(brokers: &str) -> Result<Self, rdkafka::error::KafkaError> {
        let producer: FutureProducer = ClientConfig::new()
            .set("bootstrap.servers", brokers)
            .set("message.timeout.ms", "5000")
            .create()?;

        Ok(Self { producer })
    }

    #[instrument(skip(self, event))]
    pub async fn emit_content_uploaded(&self, event: ContentUploadedEvent) {
        let payload = match serde_json::to_string(&event) {
            Ok(p) => p,
            Err(e) => {
                error!("Failed to serialize content.uploaded event: {}", e);
                return;
            }
        };

        let key = event.content_id.to_string();
        let record = FutureRecord::to("content.uploaded")
            .payload(&payload)
            .key(&key);

        match self.producer.send(record, Duration::from_secs(5)).await {
            Ok(_) => info!(content_id = %event.content_id, "Emitted content.uploaded event"),
            Err((e, _)) => error!("Failed to emit content.uploaded event: {}", e),
        }
    }

    #[instrument(skip(self, event))]
    pub async fn emit_content_deleted(&self, event: ContentDeletedEvent) {
        let payload = match serde_json::to_string(&event) {
            Ok(p) => p,
            Err(e) => {
                error!("Failed to serialize content.deleted event: {}", e);
                return;
            }
        };

        let key = event.content_id.to_string();
        let record = FutureRecord::to("content.deleted")
            .payload(&payload)
            .key(&key);

        match self.producer.send(record, Duration::from_secs(5)).await {
            Ok(_) => info!(content_id = %event.content_id, "Emitted content.deleted event"),
            Err((e, _)) => error!("Failed to emit content.deleted event: {}", e),
        }
    }
}
