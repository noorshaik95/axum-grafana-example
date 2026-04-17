use crate::db::repositories::ContentObjectRepository;
use crate::storage::minio::MinioClient;
use rdkafka::config::ClientConfig;
use rdkafka::consumer::{CommitMode, Consumer, StreamConsumer};
use rdkafka::message::Message;
use serde::Deserialize;
use std::sync::Arc;
use tokio_stream::StreamExt;
use tracing::{error, info, instrument, warn};
use uuid::Uuid;

#[derive(Debug, Deserialize)]
struct CourseDeletedEvent {
    pub tenant_id: Uuid,
    pub course_id: Uuid,
}

#[derive(Debug, Deserialize)]
struct TenantDisabledEvent {
    pub tenant_id: Uuid,
}

#[derive(Debug, Deserialize)]
struct TenantProvisionedEvent {
    pub tenant_id: Uuid,
}

/// Kafka consumer for handling external events that affect content
pub struct KafkaConsumer {
    consumer: StreamConsumer,
    minio_client: Arc<MinioClient>,
    content_repo: Arc<ContentObjectRepository>,
}

impl KafkaConsumer {
    pub fn new(
        brokers: &str,
        group_id: &str,
        minio_client: Arc<MinioClient>,
        content_repo: Arc<ContentObjectRepository>,
    ) -> Result<Self, rdkafka::error::KafkaError> {
        let consumer: StreamConsumer = ClientConfig::new()
            .set("bootstrap.servers", brokers)
            .set("group.id", group_id)
            .set("enable.auto.commit", "false")
            .set("auto.offset.reset", "earliest")
            .create()?;

        consumer.subscribe(&["course.deleted", "tenant.disabled", "tenant.provisioned"])?;

        Ok(Self {
            consumer,
            minio_client,
            content_repo,
        })
    }

    /// Run the consumer loop
    #[instrument(skip(self))]
    pub async fn run(&self) {
        info!("Starting Kafka consumer for content events");

        let mut stream = self.consumer.stream();

        while let Some(result) = stream.next().await {
            match result {
                Ok(msg) => {
                    let topic = msg.topic();
                    let payload = match msg.payload_view::<str>() {
                        Some(Ok(s)) => s,
                        Some(Err(e)) => {
                            error!("Failed to deserialize message payload: {}", e);
                            let _ = self.consumer.commit_message(&msg, CommitMode::Async);
                            continue;
                        }
                        None => {
                            warn!("Received message with empty payload on topic {}", topic);
                            let _ = self.consumer.commit_message(&msg, CommitMode::Async);
                            continue;
                        }
                    };

                    match topic {
                        "course.deleted" => self.handle_course_deleted(payload).await,
                        "tenant.disabled" => self.handle_tenant_disabled(payload).await,
                        "tenant.provisioned" => self.handle_tenant_provisioned(payload).await,
                        _ => warn!("Received message on unexpected topic: {}", topic),
                    }

                    if let Err(e) = self.consumer.commit_message(&msg, CommitMode::Async) {
                        error!("Failed to commit message: {}", e);
                    }
                }
                Err(e) => {
                    error!("Kafka consumer error: {}", e);
                }
            }
        }
    }

    async fn handle_course_deleted(&self, payload: &str) {
        let event: CourseDeletedEvent = match serde_json::from_str(payload) {
            Ok(e) => e,
            Err(e) => {
                error!("Failed to parse course.deleted event: {}", e);
                return;
            }
        };

        info!(
            tenant_id = %event.tenant_id,
            course_id = %event.course_id,
            "Handling course.deleted event"
        );

        // Soft-delete all content records and get them for MinIO cleanup
        match self
            .content_repo
            .soft_delete_by_course(event.tenant_id, event.course_id)
            .await
        {
            Ok(deleted_objects) => {
                for obj in &deleted_objects {
                    if let Err(e) = self
                        .minio_client
                        .delete_object(&obj.minio_bucket, &obj.minio_object_key)
                        .await
                    {
                        error!(
                            content_id = %obj.id,
                            "Failed to delete MinIO object during course deletion: {}",
                            e
                        );
                    }
                }
                info!(
                    "Deleted {} content objects for course {}",
                    deleted_objects.len(),
                    event.course_id
                );
            }
            Err(e) => {
                error!(
                    "Failed to soft-delete content for course {}: {}",
                    event.course_id, e
                );
            }
        }
    }

    async fn handle_tenant_disabled(&self, payload: &str) {
        let event: TenantDisabledEvent = match serde_json::from_str(payload) {
            Ok(e) => e,
            Err(e) => {
                error!("Failed to parse tenant.disabled event: {}", e);
                return;
            }
        };

        info!(
            tenant_id = %event.tenant_id,
            "Handling tenant.disabled event — restricting all content"
        );

        match self
            .content_repo
            .restrict_tenant_content(event.tenant_id)
            .await
        {
            Ok(count) => {
                info!(
                    "Restricted {} content objects for tenant {}",
                    count, event.tenant_id
                );
            }
            Err(e) => {
                error!(
                    "Failed to restrict content for tenant {}: {}",
                    event.tenant_id, e
                );
            }
        }
    }

    async fn handle_tenant_provisioned(&self, payload: &str) {
        let event: TenantProvisionedEvent = match serde_json::from_str(payload) {
            Ok(e) => e,
            Err(e) => {
                error!("Failed to parse tenant.provisioned event: {}", e);
                return;
            }
        };

        let bucket_name = self.minio_client.tenant_bucket(&event.tenant_id.to_string());

        info!(
            tenant_id = %event.tenant_id,
            bucket = %bucket_name,
            "Handling tenant.provisioned event — creating MinIO bucket"
        );

        if let Err(e) = self.minio_client.create_bucket(&bucket_name).await {
            error!(
                "Failed to create bucket for tenant {}: {}",
                event.tenant_id, e
            );
        }
    }
}
