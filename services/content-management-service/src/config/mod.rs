use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::time::Duration;

/// Main configuration structure for the Content Management Service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub server: ServerConfig,
    pub database: DatabaseConfig,
    pub s3: S3Config,
    pub minio: MinioConfig,
    pub elasticsearch: ElasticsearchConfig,
    pub redis: RedisConfig,
    pub kafka: KafkaConfig,
    pub observability: ObservabilityConfig,
    pub analytics: AnalyticsConfig,
    pub upload: UploadConfig,
    pub transcoding: TranscodingConfig,
}

/// MinIO configuration for tenant-isolated content storage
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MinioConfig {
    pub endpoint: String,
    pub access_key: String,
    pub secret_key: String,
    pub bucket_prefix: String,
}

/// Kafka configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KafkaConfig {
    pub brokers: String,
    pub consumer_group: String,
}

/// Server configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerConfig {
    pub host: String,
    pub port: u16,
    pub grpc_port: u16,
    pub metrics_port: u16,
    pub shutdown_timeout_seconds: u64,
}

/// Database configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseConfig {
    pub url: String,
    pub max_connections: u32,
    pub min_connections: u32,
    pub connection_timeout_seconds: u64,
    pub idle_timeout_seconds: u64,
    pub max_lifetime_seconds: u64,
}

/// S3/MinIO configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct S3Config {
    pub endpoint: String,
    pub access_key: String,
    pub secret_key: String,
    pub bucket: String,
    pub region: String,
    pub use_path_style: bool,
    pub presigned_url_expiry_seconds: u64,
    pub video_presigned_url_expiry_seconds: u64,
}

/// ElasticSearch configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ElasticsearchConfig {
    pub url: String,
    pub index: String,
    pub max_retries: u32,
    pub timeout_seconds: u64,
}

/// Redis configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RedisConfig {
    pub url: String,
    pub queue_name: String,
    pub connection_timeout_seconds: u64,
    pub max_pool_size: u32,
}

/// Observability configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ObservabilityConfig {
    pub service_name: String,
    pub otlp_endpoint: String,
    pub log_level: String,
    pub enable_tracing: bool,
    pub enable_metrics: bool,
    pub enable_logging: bool,
}

/// Analytics service configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AnalyticsConfig {
    pub service_url: String,
    pub batch_size: usize,
    pub batch_interval_seconds: u64,
    pub retry_interval_seconds: u64,
    pub max_queue_age_hours: u64,
}

/// Upload configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UploadConfig {
    pub max_file_size_bytes: u64,
    pub chunk_size_bytes: u64,
    pub session_expiry_hours: u64,
    pub allowed_video_types: Vec<String>,
    pub allowed_document_types: Vec<String>,
    pub enable_malware_scan: bool,
}

/// Transcoding configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TranscodingConfig {
    pub worker_count: usize,
    pub max_retries: u32,
    pub segment_duration_seconds: u32,
    pub output_formats: Vec<String>,
    pub bitrate_variants: Vec<BitrateVariant>,
}

/// Video bitrate variant configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BitrateVariant {
    pub name: String,
    pub width: u32,
    pub height: u32,
    pub bitrate_kbps: u32,
}

impl Config {
    /// Load configuration from environment variables and optional YAML file
    pub fn load() -> Result<Self> {
        // Load .env file if it exists
        dotenv::dotenv().ok();

        // Start with default configuration
        let mut config = Config::default();

        // Try to load from config file if specified
        if let Ok(config_path) = std::env::var("CONFIG_FILE") {
            let mut builder = config::Config::builder()
                .add_source(config::File::with_name(&config_path).required(true));

            let file_config = builder
                .build()
                .context("Failed to build configuration from file")?;
            config = file_config
                .try_deserialize()
                .context("Failed to deserialize configuration from file")?;
        }

        // Override with environment variables
        // Server
        if let Ok(val) = std::env::var("SERVER__HOST") {
            config.server.host = val;
        }
        if let Ok(val) = std::env::var("SERVER__PORT") {
            config.server.port = val.parse().context("Invalid SERVER__PORT")?;
        }
        if let Ok(val) = std::env::var("SERVER__GRPC_PORT") {
            config.server.grpc_port = val.parse().context("Invalid SERVER__GRPC_PORT")?;
        }
        if let Ok(val) = std::env::var("SERVER__METRICS_PORT") {
            config.server.metrics_port = val.parse().context("Invalid SERVER__METRICS_PORT")?;
        }

        // Database
        if let Ok(val) = std::env::var("DATABASE__URL") {
            config.database.url = val;
        }
        if let Ok(val) = std::env::var("DATABASE__MAX_CONNECTIONS") {
            config.database.max_connections =
                val.parse().context("Invalid DATABASE__MAX_CONNECTIONS")?;
        }
        if let Ok(val) = std::env::var("DATABASE__MIN_CONNECTIONS") {
            config.database.min_connections =
                val.parse().context("Invalid DATABASE__MIN_CONNECTIONS")?;
        }

        // S3
        if let Ok(val) = std::env::var("S3__ENDPOINT") {
            config.s3.endpoint = val;
        }
        if let Ok(val) = std::env::var("S3__ACCESS_KEY") {
            config.s3.access_key = val;
        }
        if let Ok(val) = std::env::var("S3__SECRET_KEY") {
            config.s3.secret_key = val;
        }
        if let Ok(val) = std::env::var("S3__BUCKET") {
            config.s3.bucket = val;
        }
        if let Ok(val) = std::env::var("S3__REGION") {
            config.s3.region = val;
        }

        // MinIO
        if let Ok(val) = std::env::var("MINIO_ENDPOINT") {
            config.minio.endpoint = val;
        }
        if let Ok(val) = std::env::var("MINIO_ACCESS_KEY") {
            config.minio.access_key = val;
        }
        if let Ok(val) = std::env::var("MINIO_SECRET_KEY") {
            config.minio.secret_key = val;
        }
        if let Ok(val) = std::env::var("MINIO_BUCKET_PREFIX") {
            config.minio.bucket_prefix = val;
        }

        // Kafka
        if let Ok(val) = std::env::var("KAFKA__BROKERS") {
            config.kafka.brokers = val;
        }
        if let Ok(val) = std::env::var("KAFKA__CONSUMER_GROUP") {
            config.kafka.consumer_group = val;
        }

        // ElasticSearch
        if let Ok(val) = std::env::var("ELASTICSEARCH__URL") {
            config.elasticsearch.url = val;
        }
        if let Ok(val) = std::env::var("ELASTICSEARCH__INDEX") {
            config.elasticsearch.index = val;
        }

        // Redis
        if let Ok(val) = std::env::var("REDIS__URL") {
            config.redis.url = val;
        }
        if let Ok(val) = std::env::var("REDIS__QUEUE_NAME") {
            config.redis.queue_name = val;
        }

        // Observability
        if let Ok(val) = std::env::var("OBSERVABILITY__SERVICE_NAME") {
            config.observability.service_name = val;
        }
        if let Ok(val) = std::env::var("OBSERVABILITY__OTLP_ENDPOINT") {
            config.observability.otlp_endpoint = val;
        }
        if let Ok(val) = std::env::var("OBSERVABILITY__LOG_LEVEL") {
            config.observability.log_level = val;
        }

        // Analytics
        if let Ok(val) = std::env::var("ANALYTICS__SERVICE_URL") {
            config.analytics.service_url = val;
        }

        // Upload
        if let Ok(val) = std::env::var("UPLOAD__MAX_FILE_SIZE_BYTES") {
            config.upload.max_file_size_bytes =
                val.parse().context("Invalid UPLOAD__MAX_FILE_SIZE_BYTES")?;
        }

        // Validate the configuration
        config.validate()?;

        Ok(config)
    }

    /// Validate configuration values
    pub fn validate(&self) -> Result<()> {
        // Validate server configuration
        if self.server.port == 0 {
            anyhow::bail!("Server port must be greater than 0");
        }
        if self.server.grpc_port == 0 {
            anyhow::bail!("gRPC port must be greater than 0");
        }
        if self.server.metrics_port == 0 {
            anyhow::bail!("Metrics port must be greater than 0");
        }

        // Validate database configuration
        if self.database.url.is_empty() {
            anyhow::bail!("Database URL is required");
        }
        if self.database.max_connections == 0 {
            anyhow::bail!("Database max_connections must be greater than 0");
        }
        if self.database.min_connections > self.database.max_connections {
            anyhow::bail!("Database min_connections cannot exceed max_connections");
        }

        // Validate S3 configuration
        if self.s3.endpoint.is_empty() {
            anyhow::bail!("S3 endpoint is required");
        }
        if self.s3.access_key.is_empty() {
            anyhow::bail!("S3 access_key is required");
        }
        if self.s3.secret_key.is_empty() {
            anyhow::bail!("S3 secret_key is required");
        }
        if self.s3.bucket.is_empty() {
            anyhow::bail!("S3 bucket is required");
        }
        if self.s3.region.is_empty() {
            anyhow::bail!("S3 region is required");
        }

        // Validate ElasticSearch configuration
        if self.elasticsearch.url.is_empty() {
            anyhow::bail!("ElasticSearch URL is required");
        }
        if self.elasticsearch.index.is_empty() {
            anyhow::bail!("ElasticSearch index is required");
        }

        // Validate Redis configuration
        if self.redis.url.is_empty() {
            anyhow::bail!("Redis URL is required");
        }
        if self.redis.queue_name.is_empty() {
            anyhow::bail!("Redis queue_name is required");
        }

        // Validate observability configuration
        if self.observability.service_name.is_empty() {
            anyhow::bail!("Service name is required");
        }
        if self.observability.otlp_endpoint.is_empty() {
            anyhow::bail!("OTLP endpoint is required");
        }

        // Validate analytics configuration
        if self.analytics.service_url.is_empty() {
            anyhow::bail!("Analytics service URL is required");
        }
        if self.analytics.batch_size == 0 {
            anyhow::bail!("Analytics batch_size must be greater than 0");
        }

        // Validate upload configuration
        if self.upload.max_file_size_bytes == 0 {
            anyhow::bail!("Upload max_file_size_bytes must be greater than 0");
        }
        if self.upload.chunk_size_bytes == 0 {
            anyhow::bail!("Upload chunk_size_bytes must be greater than 0");
        }
        if self.upload.allowed_video_types.is_empty() {
            anyhow::bail!("At least one allowed video type is required");
        }
        if self.upload.allowed_document_types.is_empty() {
            anyhow::bail!("At least one allowed document type is required");
        }

        // Validate transcoding configuration
        if self.transcoding.worker_count == 0 {
            anyhow::bail!("Transcoding worker_count must be greater than 0");
        }
        if self.transcoding.bitrate_variants.is_empty() {
            anyhow::bail!("At least one bitrate variant is required");
        }

        Ok(())
    }

    /// Get database connection timeout as Duration
    pub fn database_connection_timeout(&self) -> Duration {
        Duration::from_secs(self.database.connection_timeout_seconds)
    }

    /// Get database idle timeout as Duration
    pub fn database_idle_timeout(&self) -> Duration {
        Duration::from_secs(self.database.idle_timeout_seconds)
    }

    /// Get database max lifetime as Duration
    pub fn database_max_lifetime(&self) -> Duration {
        Duration::from_secs(self.database.max_lifetime_seconds)
    }

    /// Get server shutdown timeout as Duration
    pub fn shutdown_timeout(&self) -> Duration {
        Duration::from_secs(self.server.shutdown_timeout_seconds)
    }

    /// Get S3 presigned URL expiry as Duration
    pub fn s3_presigned_url_expiry(&self) -> Duration {
        Duration::from_secs(self.s3.presigned_url_expiry_seconds)
    }

    /// Get S3 video presigned URL expiry as Duration
    pub fn s3_video_presigned_url_expiry(&self) -> Duration {
        Duration::from_secs(self.s3.video_presigned_url_expiry_seconds)
    }

    /// Get upload session expiry as Duration
    pub fn upload_session_expiry(&self) -> Duration {
        Duration::from_secs(self.upload.session_expiry_hours * 3600)
    }

    /// Get analytics batch interval as Duration
    pub fn analytics_batch_interval(&self) -> Duration {
        Duration::from_secs(self.analytics.batch_interval_seconds)
    }

    /// Get analytics retry interval as Duration
    pub fn analytics_retry_interval(&self) -> Duration {
        Duration::from_secs(self.analytics.retry_interval_seconds)
    }

    /// Get analytics max queue age as Duration
    pub fn analytics_max_queue_age(&self) -> Duration {
        Duration::from_secs(self.analytics.max_queue_age_hours * 3600)
    }
}

impl Default for Config {
    fn default() -> Self {
        Self {
            server: ServerConfig {
                host: "0.0.0.0".to_string(),
                port: 8082,
                grpc_port: 50054,
                metrics_port: 9092,
                shutdown_timeout_seconds: 30,
            },
            database: DatabaseConfig {
                url: "postgresql://cms:cms_password@localhost:5432/cms".to_string(),
                max_connections: 20,
                min_connections: 5,
                connection_timeout_seconds: 30,
                idle_timeout_seconds: 600,
                max_lifetime_seconds: 1800,
            },
            s3: S3Config {
                endpoint: "http://localhost:9000".to_string(),
                access_key: "minioadmin".to_string(),
                secret_key: "minioadmin".to_string(),
                bucket: "content-storage".to_string(),
                region: "us-east-1".to_string(),
                use_path_style: true,
                presigned_url_expiry_seconds: 3600, // 1 hour for documents
                video_presigned_url_expiry_seconds: 7200, // 2 hours for videos
            },
            minio: MinioConfig {
                endpoint: "http://localhost:9000".to_string(),
                access_key: "minioadmin".to_string(),
                secret_key: "minioadmin".to_string(),
                bucket_prefix: "slate-content".to_string(),
            },
            elasticsearch: ElasticsearchConfig {
                url: "http://localhost:9200".to_string(),
                index: "content".to_string(),
                max_retries: 2,
                timeout_seconds: 30,
            },
            redis: RedisConfig {
                url: "redis://localhost:6379".to_string(),
                queue_name: "transcoding_jobs".to_string(),
                connection_timeout_seconds: 5,
                max_pool_size: 10,
            },
            kafka: KafkaConfig {
                brokers: "localhost:9092".to_string(),
                consumer_group: "content-management-service".to_string(),
            },
            observability: ObservabilityConfig {
                service_name: "content-management-service".to_string(),
                otlp_endpoint: "http://localhost:4317".to_string(),
                log_level: "info".to_string(),
                enable_tracing: true,
                enable_metrics: true,
                enable_logging: true,
            },
            analytics: AnalyticsConfig {
                service_url: "http://localhost:50053".to_string(),
                batch_size: 100,
                batch_interval_seconds: 30,
                retry_interval_seconds: 300, // 5 minutes
                max_queue_age_hours: 24,
            },
            upload: UploadConfig {
                max_file_size_bytes: 524_288_000, // 500MB
                chunk_size_bytes: 5_242_880,      // 5MB
                session_expiry_hours: 24,
                allowed_video_types: vec![
                    "video/mp4".to_string(),
                    "video/quicktime".to_string(),
                    "video/x-msvideo".to_string(),
                ],
                allowed_document_types: vec![
                    "application/pdf".to_string(),
                    "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
                        .to_string(),
                ],
                enable_malware_scan: false, // Disabled by default, requires ClamAV
            },
            transcoding: TranscodingConfig {
                worker_count: 2,
                max_retries: 3,
                segment_duration_seconds: 6,
                output_formats: vec!["hls".to_string(), "dash".to_string()],
                bitrate_variants: vec![
                    BitrateVariant {
                        name: "360p".to_string(),
                        width: 640,
                        height: 360,
                        bitrate_kbps: 800,
                    },
                    BitrateVariant {
                        name: "480p".to_string(),
                        width: 854,
                        height: 480,
                        bitrate_kbps: 1400,
                    },
                    BitrateVariant {
                        name: "720p".to_string(),
                        width: 1280,
                        height: 720,
                        bitrate_kbps: 2800,
                    },
                    BitrateVariant {
                        name: "1080p".to_string(),
                        width: 1920,
                        height: 1080,
                        bitrate_kbps: 5000,
                    },
                ],
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config_is_valid() {
        let config = Config::default();
        assert!(config.validate().is_ok());
    }

    #[test]
    fn test_invalid_server_port() {
        let mut config = Config::default();
        config.server.port = 0;
        assert!(config.validate().is_err());
    }

    #[test]
    fn test_invalid_database_url() {
        let mut config = Config::default();
        config.database.url = String::new();
        assert!(config.validate().is_err());
    }

    #[test]
    fn test_invalid_database_connections() {
        let mut config = Config::default();
        config.database.min_connections = 10;
        config.database.max_connections = 5;
        assert!(config.validate().is_err());
    }

    #[test]
    fn test_invalid_s3_endpoint() {
        let mut config = Config::default();
        config.s3.endpoint = String::new();
        assert!(config.validate().is_err());
    }

    #[test]
    fn test_invalid_elasticsearch_url() {
        let mut config = Config::default();
        config.elasticsearch.url = String::new();
        assert!(config.validate().is_err());
    }

    #[test]
    fn test_invalid_redis_url() {
        let mut config = Config::default();
        config.redis.url = String::new();
        assert!(config.validate().is_err());
    }

    #[test]
    fn test_invalid_upload_config() {
        let mut config = Config::default();
        config.upload.max_file_size_bytes = 0;
        assert!(config.validate().is_err());
    }

    #[test]
    fn test_duration_conversions() {
        let config = Config::default();
        assert_eq!(
            config.database_connection_timeout(),
            Duration::from_secs(30)
        );
        assert_eq!(config.shutdown_timeout(), Duration::from_secs(30));
        assert_eq!(config.s3_presigned_url_expiry(), Duration::from_secs(3600));
        assert_eq!(
            config.upload_session_expiry(),
            Duration::from_secs(24 * 3600)
        );
    }

    #[test]
    fn test_empty_allowed_types() {
        let mut config = Config::default();
        config.upload.allowed_video_types = vec![];
        assert!(config.validate().is_err());

        let mut config = Config::default();
        config.upload.allowed_document_types = vec![];
        assert!(config.validate().is_err());
    }

    #[test]
    fn test_transcoding_validation() {
        let mut config = Config::default();
        config.transcoding.worker_count = 0;
        assert!(config.validate().is_err());

        let mut config = Config::default();
        config.transcoding.bitrate_variants = vec![];
        assert!(config.validate().is_err());
    }
}
