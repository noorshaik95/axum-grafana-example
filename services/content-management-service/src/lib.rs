pub mod analytics;
pub mod config;
pub mod content;
pub mod db;
pub mod download;
pub mod error;
pub mod grpc;
pub mod handlers;
pub mod health;
pub mod health_server;
pub mod kafka;
pub mod models;
pub mod observability;
pub mod progress;
pub mod search;
pub mod storage;
pub mod streaming;
pub mod transcoding;
pub mod upload;
pub mod visibility;

// Include generated protobuf code
pub mod proto {
    pub mod content {
        tonic::include_proto!("content");
    }
}
