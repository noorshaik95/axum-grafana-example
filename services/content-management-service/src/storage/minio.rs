use aws_config::BehaviorVersion;
use aws_sdk_s3::{
    config::{Credentials, Region},
    presigning::PresigningConfig,
    primitives::ByteStream,
    Client,
};
use bytes::Bytes;
use std::time::Duration;
use tracing::{debug, error, info, instrument};

use crate::storage::StorageError;

/// MinioClient provides tenant-aware S3-compatible storage operations
#[derive(Clone)]
pub struct MinioClient {
    client: Client,
    bucket_prefix: String,
}

impl MinioClient {
    /// Creates a new MinioClient configured for a MinIO endpoint
    #[instrument(skip(access_key, secret_key))]
    pub async fn new(
        endpoint: &str,
        access_key: &str,
        secret_key: &str,
        bucket_prefix: &str,
    ) -> Result<Self, StorageError> {
        info!(
            "Initializing MinIO client with endpoint: {}, bucket_prefix: {}",
            endpoint, bucket_prefix
        );

        let credentials = Credentials::new(
            access_key,
            secret_key,
            None,
            None,
            "content-management-minio",
        );

        let config = aws_config::defaults(BehaviorVersion::latest())
            .region(Region::new("us-east-1"))
            .credentials_provider(credentials)
            .endpoint_url(endpoint)
            .load()
            .await;

        let s3_config = aws_sdk_s3::config::Builder::from(&config)
            .force_path_style(true)
            .build();

        let client = Client::from_conf(s3_config);

        Ok(Self {
            client,
            bucket_prefix: bucket_prefix.to_string(),
        })
    }

    /// Returns the bucket name for a given tenant
    pub fn tenant_bucket(&self, tenant_id: &str) -> String {
        format!("{}-{}", self.bucket_prefix, tenant_id)
    }

    /// Creates a bucket if it does not already exist
    #[instrument(skip(self))]
    pub async fn create_bucket(&self, bucket_name: &str) -> Result<(), StorageError> {
        match self.client.head_bucket().bucket(bucket_name).send().await {
            Ok(_) => {
                info!("Bucket '{}' already exists", bucket_name);
                Ok(())
            }
            Err(_) => {
                info!("Creating bucket '{}'", bucket_name);
                self.client
                    .create_bucket()
                    .bucket(bucket_name)
                    .send()
                    .await
                    .map_err(|e| {
                        StorageError::AwsSdkError(format!("Failed to create bucket: {}", e))
                    })?;
                info!("Bucket '{}' created successfully", bucket_name);
                Ok(())
            }
        }
    }

    /// Generates a presigned PUT URL for direct browser upload
    #[instrument(skip(self))]
    pub async fn generate_presigned_upload_url(
        &self,
        bucket: &str,
        key: &str,
        expiry_secs: u64,
    ) -> Result<String, StorageError> {
        debug!(
            "Generating presigned upload URL: bucket={}, key={}, expiry={}s",
            bucket, key, expiry_secs
        );

        let presigning_config =
            PresigningConfig::expires_in(Duration::from_secs(expiry_secs)).map_err(|e| {
                error!("Failed to create presigning config: {}", e);
                StorageError::PresignedUrlFailed(e.to_string())
            })?;

        let presigned_request = self
            .client
            .put_object()
            .bucket(bucket)
            .key(key)
            .presigned(presigning_config)
            .await
            .map_err(|e| {
                error!(
                    "Failed to generate presigned upload URL for {}/{}: {}",
                    bucket, key, e
                );
                StorageError::PresignedUrlFailed(e.to_string())
            })?;

        Ok(presigned_request.uri().to_string())
    }

    /// Generates a presigned GET URL for downloading content
    #[instrument(skip(self))]
    pub async fn generate_presigned_download_url(
        &self,
        bucket: &str,
        key: &str,
        expiry_secs: u64,
    ) -> Result<String, StorageError> {
        debug!(
            "Generating presigned download URL: bucket={}, key={}, expiry={}s",
            bucket, key, expiry_secs
        );

        let presigning_config =
            PresigningConfig::expires_in(Duration::from_secs(expiry_secs)).map_err(|e| {
                error!("Failed to create presigning config: {}", e);
                StorageError::PresignedUrlFailed(e.to_string())
            })?;

        let presigned_request = self
            .client
            .get_object()
            .bucket(bucket)
            .key(key)
            .presigned(presigning_config)
            .await
            .map_err(|e| {
                error!(
                    "Failed to generate presigned download URL for {}/{}: {}",
                    bucket, key, e
                );
                StorageError::PresignedUrlFailed(e.to_string())
            })?;

        Ok(presigned_request.uri().to_string())
    }

    /// Uploads bytes to MinIO
    #[instrument(skip(self, data))]
    pub async fn put_object(
        &self,
        bucket: &str,
        key: &str,
        data: Bytes,
        content_type: &str,
    ) -> Result<(), StorageError> {
        debug!(
            "Uploading object: bucket={}, key={}, size={}, content_type={}",
            bucket,
            key,
            data.len(),
            content_type
        );

        self.client
            .put_object()
            .bucket(bucket)
            .key(key)
            .body(ByteStream::from(data))
            .content_type(content_type)
            .send()
            .await
            .map_err(|e| StorageError::UploadFailed(e.to_string()))?;

        info!("Successfully uploaded object: {}/{}", bucket, key);
        Ok(())
    }

    /// Hard-deletes an object from MinIO
    #[instrument(skip(self))]
    pub async fn delete_object(&self, bucket: &str, key: &str) -> Result<(), StorageError> {
        debug!("Deleting object: bucket={}, key={}", bucket, key);

        self.client
            .delete_object()
            .bucket(bucket)
            .key(key)
            .send()
            .await
            .map_err(|e| StorageError::DeleteFailed(e.to_string()))?;

        info!("Successfully deleted object: {}/{}", bucket, key);
        Ok(())
    }

    /// Lists object keys matching a prefix
    #[instrument(skip(self))]
    pub async fn list_objects(
        &self,
        bucket: &str,
        prefix: &str,
    ) -> Result<Vec<String>, StorageError> {
        debug!(
            "Listing objects: bucket={}, prefix={}",
            bucket, prefix
        );

        let response = self
            .client
            .list_objects_v2()
            .bucket(bucket)
            .prefix(prefix)
            .send()
            .await
            .map_err(|e| StorageError::AwsSdkError(e.to_string()))?;

        let keys: Vec<String> = response
            .contents()
            .iter()
            .filter_map(|obj| obj.key().map(|k| k.to_string()))
            .collect();

        info!(
            "Found {} objects with prefix '{}' in bucket '{}'",
            keys.len(),
            prefix,
            bucket
        );
        Ok(keys)
    }
}
