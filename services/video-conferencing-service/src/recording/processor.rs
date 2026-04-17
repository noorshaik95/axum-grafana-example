use crate::config::RecordingConfig;
use crate::database::repository::VideoRepository;
use crate::recording::GcsUploader;
use std::path::PathBuf;
use std::sync::Arc;
use tokio::sync::mpsc;
use uuid::Uuid;

pub struct RecordingProcessor {
    repo: VideoRepository,
    gcs_uploader: Arc<GcsUploader>,
    config: RecordingConfig,
    rx: mpsc::Receiver<RecordingTask>,
}

pub struct RecordingTask {
    pub recording_id: Uuid,
    pub session_id: Uuid,
    pub file_path: PathBuf,
}

impl RecordingProcessor {
    pub async fn new(
        repo: VideoRepository,
        gcs_uploader: Arc<GcsUploader>,
        config: RecordingConfig,
    ) -> (Self, mpsc::Sender<RecordingTask>) {
        let (tx, rx) = mpsc::channel(100);

        let processor = Self {
            repo,
            gcs_uploader,
            config,
            rx,
        };

        (processor, tx)
    }

    /// Process recordings in background (AC9: Available within 30 minutes)
    pub async fn run(mut self) {
        tracing::info!("Recording processor started");

        while let Some(task) = self.rx.recv().await {
            self.process_recording(task).await;
        }

        tracing::info!("Recording processor stopped");
    }

    async fn process_recording(&self, task: RecordingTask) {
        let start = std::time::Instant::now();

        tracing::info!(
            "Processing recording: id={}, session={}, file={:?}",
            task.recording_id,
            task.session_id,
            task.file_path
        );

        // Update status to PROCESSING
        if let Err(e) = self
            .repo
            .update_recording_status(task.recording_id, "PROCESSING", None)
            .await
        {
            tracing::error!("Failed to update recording status: {}", e);
            return;
        }

        // Upload to GCS (AC8)
        match self
            .gcs_uploader
            .upload_recording(
                &task.session_id.to_string(),
                &task.recording_id.to_string(),
                &task.file_path,
            )
            .await
        {
            Ok((gcs_url, file_size)) => {
                // Extract object key from URL
                let object_key = gcs_url.trim_start_matches(&format!(
                    "gs://{}/",
                    self.config.gcs_bucket
                ));

                let duration_seconds = start.elapsed().as_secs() as i32;

                // Update recording with GCS info (AC9: Available within 30 min)
                if let Err(e) = self
                    .repo
                    .complete_recording(
                        task.recording_id,
                        duration_seconds,
                        file_size,
                        gcs_url,
                        self.config.gcs_bucket.clone(),
                        object_key.to_string(),
                    )
                    .await
                {
                    tracing::error!("Failed to complete recording: {}", e);
                    let _ = self
                        .repo
                        .update_recording_status(
                            task.recording_id,
                            "FAILED",
                            Some(format!("Failed to update database: {}", e)),
                        )
                        .await;
                    return;
                }

                // Clean up local file
                if let Err(e) = tokio::fs::remove_file(&task.file_path).await {
                    tracing::warn!("Failed to delete local recording file: {}", e);
                }

                tracing::info!(
                    "Recording processing completed: id={}, duration={}s",
                    task.recording_id,
                    duration_seconds
                );
            }
            Err(e) => {
                tracing::error!("Failed to upload recording to GCS: {}", e);
                let _ = self
                    .repo
                    .update_recording_status(
                        task.recording_id,
                        "FAILED",
                        Some(format!("GCS upload failed: {}", e)),
                    )
                    .await;

                crate::observability::METRICS.recordings_failed_total.inc();
            }
        }
    }
}
