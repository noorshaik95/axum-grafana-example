use serde::{Deserialize, Serialize};
use std::env;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub server: ServerConfig,
    pub database: DatabaseConfig,
    pub webrtc: WebRTCConfig,
    pub recording: RecordingConfig,
    pub observability: ObservabilityConfig,
    pub session: SessionConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerConfig {
    pub host: String,
    pub grpc_port: u16,
    pub ws_port: u16,
    pub http_port: u16,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseConfig {
    pub host: String,
    pub port: u16,
    pub name: String,
    pub user: String,
    pub password: String,
    pub max_connections: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebRTCConfig {
    pub ice_servers: Vec<IceServer>,
    pub port_range_min: u16,
    pub port_range_max: u16,
    pub public_ip: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IceServer {
    pub urls: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub username: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub credential: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecordingConfig {
    pub enabled: bool,
    pub gcs_bucket: String,
    pub gcs_project_id: String,
    pub processing_timeout_seconds: u64,
    pub temp_storage_path: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ObservabilityConfig {
    pub tempo_endpoint: String,
    pub service_name: String,
    pub prometheus_port: u16,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionConfig {
    pub max_participants: usize,
    pub join_window_minutes: i64, // AC3: 10 minutes before start
    pub default_duration_minutes: i64,
    pub max_duration_minutes: i64,
    pub auto_end_after_minutes: i64,
}

impl Config {
    pub fn from_env() -> anyhow::Result<Self> {
        dotenvy::dotenv().ok();

        Ok(Config {
            server: ServerConfig {
                host: env::var("SERVER_HOST").unwrap_or_else(|_| "0.0.0.0".to_string()),
                grpc_port: env::var("GRPC_PORT")
                    .unwrap_or_else(|_| "50052".to_string())
                    .parse()?,
                ws_port: env::var("WS_PORT")
                    .unwrap_or_else(|_| "8082".to_string())
                    .parse()?,
                http_port: env::var("HTTP_PORT")
                    .unwrap_or_else(|_| "8083".to_string())
                    .parse()?,
            },
            database: DatabaseConfig {
                host: env::var("DB_HOST").unwrap_or_else(|_| "localhost".to_string()),
                port: env::var("DB_PORT")
                    .unwrap_or_else(|_| "5432".to_string())
                    .parse()?,
                name: env::var("DB_NAME")
                    .unwrap_or_else(|_| "slate_video".to_string()),
                user: env::var("DB_USER").unwrap_or_else(|_| "postgres".to_string()),
                password: env::var("DB_PASSWORD").unwrap_or_else(|_| "postgres".to_string()),
                max_connections: env::var("DB_MAX_CONNECTIONS")
                    .unwrap_or_else(|_| "25".to_string())
                    .parse()?,
            },
            webrtc: WebRTCConfig {
                ice_servers: vec![
                    IceServer {
                        urls: vec!["stun:stun.l.google.com:19302".to_string()],
                        username: None,
                        credential: None,
                    },
                ],
                port_range_min: env::var("WEBRTC_PORT_MIN")
                    .unwrap_or_else(|_| "50000".to_string())
                    .parse()?,
                port_range_max: env::var("WEBRTC_PORT_MAX")
                    .unwrap_or_else(|_| "50100".to_string())
                    .parse()?,
                public_ip: env::var("WEBRTC_PUBLIC_IP").ok(),
            },
            recording: RecordingConfig {
                enabled: env::var("RECORDING_ENABLED")
                    .unwrap_or_else(|_| "true".to_string())
                    .parse()?,
                gcs_bucket: env::var("GCS_BUCKET")
                    .unwrap_or_else(|_| "slate-recordings".to_string()),
                gcs_project_id: env::var("GCS_PROJECT_ID")
                    .unwrap_or_else(|_| "slate-project".to_string()),
                processing_timeout_seconds: env::var("RECORDING_TIMEOUT")
                    .unwrap_or_else(|_| "1800".to_string()) // 30 minutes (AC9)
                    .parse()?,
                temp_storage_path: env::var("TEMP_STORAGE_PATH")
                    .unwrap_or_else(|_| "/tmp/recordings".to_string()),
            },
            observability: ObservabilityConfig {
                tempo_endpoint: env::var("TEMPO_ENDPOINT")
                    .unwrap_or_else(|_| "http://tempo:4317".to_string()),
                service_name: env::var("SERVICE_NAME")
                    .unwrap_or_else(|_| "video-conferencing-service".to_string()),
                prometheus_port: env::var("PROMETHEUS_PORT")
                    .unwrap_or_else(|_| "9092".to_string())
                    .parse()?,
            },
            session: SessionConfig {
                max_participants: env::var("MAX_PARTICIPANTS")
                    .unwrap_or_else(|_| "50".to_string()) // AC5
                    .parse()?,
                join_window_minutes: env::var("JOIN_WINDOW_MINUTES")
                    .unwrap_or_else(|_| "10".to_string()) // AC3
                    .parse()?,
                default_duration_minutes: env::var("DEFAULT_DURATION_MINUTES")
                    .unwrap_or_else(|_| "60".to_string())
                    .parse()?,
                max_duration_minutes: env::var("MAX_DURATION_MINUTES")
                    .unwrap_or_else(|_| "240".to_string())
                    .parse()?,
                auto_end_after_minutes: env::var("AUTO_END_AFTER_MINUTES")
                    .unwrap_or_else(|_| "300".to_string())
                    .parse()?,
            },
        })
    }

    pub fn database_url(&self) -> String {
        format!(
            "postgresql://{}:{}@{}:{}/{}",
            self.database.user,
            self.database.password,
            self.database.host,
            self.database.port,
            self.database.name
        )
    }
}

#[cfg(test)]
mod tests;
