use axum::{routing::get, Router};
use std::net::SocketAddr;
use std::sync::Arc;
use tonic::transport::Server;
use tower_http::cors::CorsLayer;
use tower_http::trace::TraceLayer;
use video_conferencing_service::{
    config::Config,
    database::{create_pool, repository::VideoRepository, run_migrations},
    grpc::VideoConferencingServiceImpl,
    handlers::{health_handler, metrics_handler},
    models::proto::video_conferencing_service_server::VideoConferencingServiceServer,
    observability::init_tracing,
    recording::{GcsUploader, RecordingProcessor},
    webrtc::SignalingServer,
};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    // Load configuration
    let config = Config::from_env()?;
    let config = Arc::new(config);

    // Initialize tracing
    init_tracing(
        &config.observability.service_name,
        &config.observability.tempo_endpoint,
    )?;

    tracing::info!("Starting Video Conferencing Service");
    tracing::info!("Configuration: {:?}", config);

    // Create database connection pool
    let db_pool = create_pool(&config.database_url(), config.database.max_connections).await?;

    // Run migrations
    run_migrations(&db_pool).await?;

    let repo = VideoRepository::new(db_pool);

    // Initialize GCS uploader
    let gcs_uploader = Arc::new(GcsUploader::new(config.recording.clone()).await?);

    // Initialize recording processor
    let (recording_processor, recording_tx) =
        RecordingProcessor::new(repo.clone(), gcs_uploader.clone(), config.recording.clone())
            .await;

    // Spawn recording processor task
    tokio::spawn(async move {
        recording_processor.run().await;
    });

    // Create gRPC service
    let grpc_service =
        VideoConferencingServiceImpl::new(repo.clone(), config.clone());
    let reflection_service = tonic_reflection::server::Builder::configure()
        .register_encoded_file_descriptor_set(tonic::include_file_descriptor_set!(
            "video_conferencing_descriptor"
        ))
        .build()
        .unwrap();

    // Create WebRTC signaling server
    let signaling_server = SignalingServer::new();

    // Create HTTP router for WebSocket, metrics, and health
    let http_app = Router::new()
        .route("/health", get(health_handler))
        .route("/metrics", get(metrics_handler))
        .merge(signaling_server.router())
        .layer(CorsLayer::permissive())
        .layer(TraceLayer::new_for_http());

    // Spawn gRPC server
    let grpc_addr: SocketAddr = format!("{}:{}", config.server.host, config.server.grpc_port)
        .parse()
        .expect("Invalid gRPC address");

    let grpc_server_handle = tokio::spawn(async move {
        tracing::info!("gRPC server listening on {}", grpc_addr);

        Server::builder()
            .add_service(VideoConferencingServiceServer::new(grpc_service))
            .add_service(reflection_service)
            .serve(grpc_addr)
            .await
            .expect("gRPC server failed");
    });

    // Spawn WebSocket/HTTP server
    let ws_addr: SocketAddr = format!("{}:{}", config.server.host, config.server.ws_port)
        .parse()
        .expect("Invalid WebSocket address");

    let ws_server_handle = tokio::spawn(async move {
        tracing::info!("WebSocket/HTTP server listening on {}", ws_addr);

        let listener = tokio::net::TcpListener::bind(ws_addr)
            .await
            .expect("Failed to bind WebSocket server");

        axum::serve(listener, http_app)
            .await
            .expect("WebSocket server failed");
    });

    // Spawn Prometheus metrics server
    let metrics_addr: SocketAddr = format!("0.0.0.0:{}", config.observability.prometheus_port)
        .parse()
        .expect("Invalid metrics address");

    let metrics_app = Router::new().route("/metrics", get(metrics_handler));

    let metrics_server_handle = tokio::spawn(async move {
        tracing::info!("Prometheus metrics server listening on {}", metrics_addr);

        let listener = tokio::net::TcpListener::bind(metrics_addr)
            .await
            .expect("Failed to bind metrics server");

        axum::serve(listener, metrics_app)
            .await
            .expect("Metrics server failed");
    });

    tracing::info!("Video Conferencing Service started successfully");
    tracing::info!("  gRPC:      {}", grpc_addr);
    tracing::info!("  WebSocket: {}", ws_addr);
    tracing::info!("  Metrics:   {}", metrics_addr);

    // Wait for all servers
    tokio::select! {
        _ = grpc_server_handle => {
            tracing::error!("gRPC server stopped unexpectedly");
        }
        _ = ws_server_handle => {
            tracing::error!("WebSocket server stopped unexpectedly");
        }
        _ = metrics_server_handle => {
            tracing::error!("Metrics server stopped unexpectedly");
        }
        _ = tokio::signal::ctrl_c() => {
            tracing::info!("Received shutdown signal");
        }
    }

    // Graceful shutdown
    tracing::info!("Shutting down Video Conferencing Service");
    video_conferencing_service::observability::tracing::shutdown_tracing();

    Ok(())
}
