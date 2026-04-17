use anyhow::Result;
use axum::Router;
use content_management_service::{
    config::Config,
    db::DatabasePool,
    health::HealthChecker,
    health_server::health_routes,
    observability::{init_metrics, init_tracing, shutdown_tracing},
};
use std::sync::Arc;
use tracing::{error, info};

const VERSION: &str = env!("CARGO_PKG_VERSION");

#[tokio::main]
async fn main() -> Result<()> {
    // Load configuration first (before tracing initialization)
    let config = Config::load()?;

    // Initialize distributed tracing and structured logging
    init_tracing(
        &config.observability.service_name,
        &config.observability.otlp_endpoint,
    )?;

    // Log service startup
    info!(
        service_name = %config.observability.service_name,
        version = %VERSION,
        grpc_port = %config.server.grpc_port,
        metrics_port = %config.server.metrics_port,
        "Service starting"
    );

    // Log configuration
    info!(
        database_url = %config.database.url.split('@').last().unwrap_or("***"),
        s3_endpoint = %config.s3.endpoint,
        elasticsearch_url = %config.elasticsearch.url,
        redis_url = %config.redis.url.split('@').last().unwrap_or("***"),
        "Configuration loaded"
    );

    // Initialize metrics
    let metrics = Arc::new(init_metrics()?);
    info!("Metrics initialized");

    // Initialize database with retry logic
    info!("Connecting to database...");
    let db_pool = match DatabasePool::new_with_retry(&config.database.url).await {
        Ok(pool) => Arc::new(pool),
        Err(e) => {
            error!("Failed to connect to database: {}", e);
            return Err(e);
        }
    };

    // Run database migrations
    if let Err(e) = db_pool.run_migrations().await {
        error!("Failed to run database migrations: {}", e);
        return Err(e);
    }

    // Verify database connection
    if let Err(e) = db_pool.health_check().await {
        error!("Database health check failed: {}", e);
        return Err(e);
    }

    info!("Database initialized successfully");

    // Initialize health checker
    let health_checker = Arc::new(HealthChecker::new(db_pool.clone()));
    info!("Health checker initialized");

    // Start metrics and health server
    let metrics_clone = metrics.clone();
    let health_checker_clone = health_checker.clone();
    let metrics_port = config.server.metrics_port;

    tokio::spawn(async move {
        // Combine metrics and health endpoints
        let app = Router::new()
            .route(
                "/metrics",
                axum::routing::get({
                    let metrics = metrics_clone.clone();
                    move || async move {
                        match metrics.encode() {
                            Ok(body) => (axum::http::StatusCode::OK, body),
                            Err(e) => {
                                error!("Failed to encode metrics: {}", e);
                                (
                                    axum::http::StatusCode::INTERNAL_SERVER_ERROR,
                                    format!("Failed to encode metrics: {}", e),
                                )
                            }
                        }
                    }
                }),
            )
            .merge(health_routes(health_checker_clone));

        let addr = format!("0.0.0.0:{}", metrics_port);
        info!("Starting metrics and health server on {}", addr);

        let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
        axum::serve(listener, app).await.unwrap();
    });

    // Initialize S3 client (for existing upload/transcoding flows)
    info!("Initializing S3 client...");
    let s3_client = Arc::new(
        content_management_service::storage::S3Client::new(
            config.s3.endpoint.clone(),
            config.s3.region.clone(),
            config.s3.access_key.clone(),
            config.s3.secret_key.clone(),
            config.s3.bucket.clone(),
        )
        .await?,
    );
    info!("S3 client initialized");

    // Initialize MinIO client (for tenant-isolated content storage)
    info!("Initializing MinIO client...");
    let minio_client = Arc::new(
        content_management_service::storage::MinioClient::new(
            &config.minio.endpoint,
            &config.minio.access_key,
            &config.minio.secret_key,
            &config.minio.bucket_prefix,
        )
        .await?,
    );
    info!("MinIO client initialized");

    // Initialize Kafka producer
    info!("Initializing Kafka producer...");
    let kafka_producer = Arc::new(
        content_management_service::kafka::KafkaProducer::new(&config.kafka.brokers)
            .map_err(|e| anyhow::anyhow!("Failed to create Kafka producer: {}", e))?,
    );
    info!("Kafka producer initialized");

    // Initialize content object repository
    let content_object_repo = Arc::new(
        content_management_service::db::repositories::ContentObjectRepository::new(
            db_pool.pool().clone(),
        ),
    );

    // Initialize ElasticSearch client
    info!("Initializing ElasticSearch client...");
    let es_client = Arc::new(
        content_management_service::search::ElasticsearchClient::new(&config.elasticsearch)?,
    );
    info!("ElasticSearch client initialized");

    // Initialize repositories
    let upload_session_repo = Arc::new(
        content_management_service::db::repositories::UploadSessionRepository::new(
            db_pool.pool().clone(),
        ),
    );
    let resource_repo = Arc::new(
        content_management_service::db::repositories::ResourceRepository::new(
            db_pool.pool().clone(),
        ),
    );
    let _transcoding_job_repo = Arc::new(
        content_management_service::db::repositories::TranscodingJobRepository::new(
            db_pool.pool().clone(),
        ),
    );
    info!("Repositories initialized");

    // Initialize service components
    info!("Initializing service components...");

    // Initialize analytics publisher first (needed by other services)
    let analytics_publisher = Arc::new(
        content_management_service::analytics::AnalyticsPublisher::new(
            &config.redis.url,
            &config.analytics.service_url,
        )?,
    );

    let content_manager = Arc::new(content_management_service::content::ContentManager::new(
        db_pool.pool().clone(),
    ));

    let upload_handler = Arc::new(content_management_service::upload::UploadHandler::new(
        upload_session_repo.clone(),
        resource_repo.clone(),
        s3_client.clone(),
    ));

    let streaming_service = Arc::new(
        content_management_service::streaming::StreamingService::with_analytics(
            content_management_service::db::repositories::ResourceRepository::new(
                db_pool.pool().clone(),
            ),
            content_management_service::db::repositories::ProgressRepository::new(
                db_pool.pool().clone(),
            ),
            analytics_publisher.clone(),
        ),
    );

    let progress_tracker = Arc::new(content_management_service::progress::ProgressTracker::new(
        content_management_service::db::repositories::ProgressRepository::new(
            db_pool.pool().clone(),
        ),
        content_management_service::db::repositories::ResourceRepository::new(
            db_pool.pool().clone(),
        ),
        content_management_service::db::repositories::LessonRepository::new(db_pool.pool().clone()),
        content_management_service::db::repositories::ModuleRepository::new(db_pool.pool().clone()),
    ));

    let search_service = Arc::new(content_management_service::search::SearchService::new(
        es_client.clone(),
    ));

    let download_manager = Arc::new(content_management_service::download::DownloadManager::new(
        content_management_service::db::repositories::ResourceRepository::new(
            db_pool.pool().clone(),
        ),
        content_management_service::db::repositories::DownloadTrackingRepository::new(
            db_pool.pool().clone(),
        ),
        (*s3_client).clone(),
        analytics_publisher.clone(),
    ));

    info!("Service components initialized");

    // Create gRPC service implementations
    let content_service =
        content_management_service::grpc::ContentServiceImpl::new(content_manager.clone());
    let upload_service =
        content_management_service::grpc::UploadServiceImpl::new(upload_handler.clone());
    let streaming_service_impl =
        content_management_service::grpc::StreamingServiceImpl::new(streaming_service.clone());
    let progress_service =
        content_management_service::grpc::ProgressServiceImpl::new(progress_tracker.clone());
    let search_service_handler =
        content_management_service::grpc::SearchServiceHandler::new(search_service.clone());
    let download_service_handler =
        content_management_service::grpc::DownloadServiceHandler::new(download_manager.clone());

    // Import gRPC server traits
    use content_management_service::proto::content::{
        content_service_server::ContentServiceServer,
        download_service_server::DownloadServiceServer,
        progress_service_server::ProgressServiceServer, search_service_server::SearchServiceServer,
        streaming_service_server::StreamingServiceServer,
        upload_service_server::UploadServiceServer,
    };

    // Start background workers
    info!("Starting background workers...");

    // Start video transcoding workers
    let worker_count = config.transcoding.worker_count;
    let work_dir = std::path::PathBuf::from("/tmp/transcoding");

    for i in 0..worker_count {
        let redis_url = config.redis.url.clone();
        let queue_name = config.redis.queue_name.clone();
        let s3_endpoint = config.s3.endpoint.clone();
        let s3_region = config.s3.region.clone();
        let s3_access_key = config.s3.access_key.clone();
        let s3_secret_key = config.s3.secret_key.clone();
        let bucket = config.s3.bucket.clone();
        let db_url = config.database.url.clone();
        let work_dir = work_dir.clone();

        tokio::spawn(async move {
            // Create a new queue connection for this worker
            let queue = match content_management_service::transcoding::TranscodingQueue::new(
                &redis_url, queue_name,
            )
            .await
            {
                Ok(q) => q,
                Err(e) => {
                    error!("Failed to create queue for worker {}: {}", i, e);
                    return;
                }
            };

            // Create S3 client for this worker
            let worker_s3 = match content_management_service::storage::S3Client::new(
                s3_endpoint,
                s3_region,
                s3_access_key,
                s3_secret_key,
                bucket.clone(),
            )
            .await
            {
                Ok(client) => client,
                Err(e) => {
                    error!("Failed to create S3 client for worker {}: {}", i, e);
                    return;
                }
            };

            // Create database pool for this worker
            let worker_pool = match content_management_service::db::DatabasePool::new(&db_url).await
            {
                Ok(pool) => pool,
                Err(e) => {
                    error!("Failed to create database pool for worker {}: {}", i, e);
                    return;
                }
            };

            let mut worker = content_management_service::transcoding::VideoTranscoder::new(
                queue,
                worker_s3,
                content_management_service::db::repositories::TranscodingJobRepository::new(
                    worker_pool.pool().clone(),
                ),
                content_management_service::db::repositories::ResourceRepository::new(
                    worker_pool.pool().clone(),
                ),
                work_dir,
                bucket,
            );

            info!("Starting transcoding worker {}", i);
            if let Err(e) = worker.run().await {
                error!("Transcoding worker {} failed: {}", i, e);
            }
        });
    }

    // Start analytics publisher worker
    let analytics_publisher_worker = analytics_publisher.clone();
    tokio::spawn(async move {
        analytics_publisher_worker.start_worker().await;
    });

    // Start analytics retry worker
    let analytics_retry_worker = analytics_publisher.clone();
    tokio::spawn(async move {
        analytics_retry_worker.start_retry_worker().await;
    });

    // Start upload session cleanup worker
    let cleanup_repo = upload_session_repo.clone();
    tokio::spawn(async move {
        let mut interval = tokio::time::interval(tokio::time::Duration::from_secs(3600)); // Run every hour
        loop {
            interval.tick().await;
            info!("Running upload session cleanup");
            if let Err(e) = cleanup_repo.cleanup_expired_sessions().await {
                error!("Failed to cleanup expired sessions: {}", e);
            }
        }
    });

    // Start Kafka consumer
    let kafka_minio = minio_client.clone();
    let kafka_content_repo = content_object_repo.clone();
    let kafka_brokers = config.kafka.brokers.clone();
    let kafka_group = config.kafka.consumer_group.clone();
    tokio::spawn(async move {
        match content_management_service::kafka::KafkaConsumer::new(
            &kafka_brokers,
            &kafka_group,
            kafka_minio,
            kafka_content_repo,
        ) {
            Ok(consumer) => {
                info!("Kafka consumer started");
                consumer.run().await;
            }
            Err(e) => {
                error!("Failed to create Kafka consumer: {}", e);
            }
        }
    });

    // Start REST API server for content management
    let rest_state = content_management_service::handlers::content_handlers::ContentAppState {
        minio_client: minio_client.clone(),
        content_repo: content_object_repo.clone(),
        kafka_producer: kafka_producer.clone(),
        bucket_prefix: config.minio.bucket_prefix.clone(),
    };

    let rest_port = config.server.port;
    let rest_host = config.server.host.clone();
    tokio::spawn(async move {
        let app = content_management_service::handlers::content_routes(rest_state);
        let addr = format!("{}:{}", rest_host, rest_port);
        info!("Starting REST API server on {}", addr);
        let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
        axum::serve(listener, app).await.unwrap();
    });

    info!("Background workers started");

    // Start gRPC server
    let grpc_addr = format!("{}:{}", config.server.host, config.server.grpc_port)
        .parse()
        .map_err(|e| anyhow::anyhow!("Invalid gRPC address: {}", e))?;

    info!("Starting gRPC server on {}", grpc_addr);

    // Reflection service
    let descriptor_set = include_bytes!(concat!(env!("OUT_DIR"), "/descriptor.bin"));
    let reflection_service = tonic_reflection::server::Builder::configure()
        .register_encoded_file_descriptor_set(descriptor_set)
        .build_v1()
        .unwrap();

    // Add OpenTelemetry gRPC layer for automatic trace propagation
    // The tower-http TraceLayer will automatically extract trace context from gRPC metadata
    let grpc_server = tonic::transport::Server::builder()
        .layer(
            tower::ServiceBuilder::new()
                // Add OpenTelemetry tracing layer for automatic trace propagation
                .layer(tower_http::trace::TraceLayer::new_for_grpc()),
        )
        .add_service(reflection_service)
        .add_service(ContentServiceServer::new(content_service))
        .add_service(UploadServiceServer::new(upload_service))
        .add_service(StreamingServiceServer::new(streaming_service_impl))
        .add_service(ProgressServiceServer::new(progress_service))
        .add_service(SearchServiceServer::new(search_service_handler))
        .add_service(DownloadServiceServer::new(download_service_handler))
        .serve_with_shutdown(grpc_addr, async {
            tokio::signal::ctrl_c()
                .await
                .expect("Failed to listen for shutdown signal");
        });

    info!("Content Management Service is ready");

    // Run gRPC server (it will shutdown when ctrl_c is received)
    grpc_server.await?;

    // Graceful shutdown initiated
    info!("Initiating graceful shutdown");
    info!("Initiating graceful shutdown...");

    // Give background workers time to finish current tasks
    info!("Waiting for background workers to finish...");
    tokio::time::sleep(tokio::time::Duration::from_secs(5)).await;

    // Flush pending analytics events
    info!("Flushing pending analytics events...");
    if let Err(e) = analytics_publisher.flush_batch().await {
        error!("Failed to flush analytics events during shutdown: {}", e);
    }

    // Close database connections
    info!("Closing database connections...");
    db_pool.close().await;
    info!("Database connections closed");

    // Shutdown tracing
    shutdown_tracing();
    info!("Shutdown complete");

    info!("Graceful shutdown complete");

    Ok(())
}
