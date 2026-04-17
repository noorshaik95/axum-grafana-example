use crate::config::RecordingConfig;
use crate::observability::METRICS;
use google_cloud_storage::client::{Client, ClientConfig};
use google_cloud_storage::http::objects::upload::{Media, UploadObjectRequest, UploadType};
use std::path::Path;
use tokio::fs::File;
use tokio::io::AsyncReadExt;

pub struct GcsUploader {
    client: Client,
    config: RecordingConfig,
}

impl GcsUploader {
    pub async fn new(config: RecordingConfig) -> anyhow::Result<Self> {
        let client_config = ClientConfig::default().with_auth().await?;
        let client = Client::new(client_config);

        tracing::info!(
            "GCS uploader initialized for bucket: {}",
            config.gcs_bucket
        );

        Ok(Self { client, config })
    }

    /// Upload recording to GCS (AC8: Session auto-records to GCS)
    /// AC9: Recording available within 30 minutes
    pub async fn upload_recording(
        &self,
        session_id: &str,
        recording_id: &str,
        file_path: &Path,
    ) -> anyhow::Result<(String, i64)> {
        let start = std::time::Instant::now();

        tracing::info!(
            "Starting GCS upload: recording={}, file={:?}",
            recording_id,
            file_path
        );

        // Read file
        let mut file = File::open(file_path).await?;
        let mut buffer = Vec::new();
        file.read_to_end(&mut buffer).await?;

        let file_size = buffer.len() as i64;

        // Generate object key
        let object_key = format!(
            "recordings/{}/{}/{}.webm",
            chrono::Utc::now().format("%Y/%m/%d"),
            session_id,
            recording_id
        );

        // Upload to GCS
        let upload_type = UploadType::Simple(Media::new(object_key.clone()));
        let upload_request = UploadObjectRequest {
            bucket: self.config.gcs_bucket.clone(),
            ..Default::default()
        };

        self.client
            .upload_object(&upload_request, buffer, &upload_type)
            .await?;

        let gcs_url = format!(
            "gs://{}/{}",
            self.config.gcs_bucket, object_key
        );

        let duration = start.elapsed();
        METRICS
            .recording_processing_duration_seconds
            .observe(duration.as_secs_f64());
        METRICS
            .recording_file_size_bytes
            .observe(file_size as f64);
        METRICS.recordings_completed_total.inc();

        tracing::info!(
            "Recording uploaded to GCS: url={}, size={} bytes, duration={:.2}s",
            gcs_url,
            file_size,
            duration.as_secs_f64()
        );

        Ok((gcs_url, file_size))
    }

    pub async fn delete_recording(&self, object_key: &str) -> anyhow::Result<()> {
        self.client
            .delete_object(&self.config.gcs_bucket, object_key, None)
            .await?;

        tracing::info!("Recording deleted from GCS: {}", object_key);

        Ok(())
    }

    pub fn generate_signed_url(&self, object_key: &str, duration_hours: u32) -> String {
        // In production, generate a signed URL with expiration
        // For now, return a placeholder
        format!(
            "https://storage.googleapis.com/{}/{}",
            self.config.gcs_bucket, object_key
        )
    }
}
