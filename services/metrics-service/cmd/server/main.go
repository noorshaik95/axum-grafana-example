package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	commongrpc "slate/libs/common-go/grpc"
	"slate/libs/common-go/logging"
	"slate/libs/common-go/tracing"
	"slate/services/metrics-service/internal/config"
	"slate/services/metrics-service/internal/handlers"
	kafkapkg "slate/services/metrics-service/internal/kafka"
	"slate/services/metrics-service/internal/repository"
	"slate/services/metrics-service/migrations"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	log := logging.NewLogger("metrics-service", logLevel)
	log.Info().Msg("Starting Metrics Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load configuration")
		os.Exit(1)
	}

	// Initialize tracing via common-go
	shutdownTracer, err := tracing.InitTracer("metrics-service", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize tracing, continuing without it")
	} else {
		defer shutdownTracer()
		log.Info().Msg("Tracing initialized via common-go")
	}

	// Connect to database with retries
	db, err := connectWithRetry(log, cfg.Database, 5, 3*time.Second)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to database")
		os.Exit(1)
	}
	defer db.Close()
	log.Info().Msg("Database connection established")

	// Run migrations
	if err := migrations.RunMigrations(db, "./migrations"); err != nil {
		log.Error().Err(err).Msg("Failed to run migrations")
		os.Exit(1)
	}
	log.Info().Msg("Migrations completed successfully")

	// Initialize repository
	repo := repository.New(db)

	// Start Kafka consumer
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()

	kafkaConsumer := kafkapkg.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, repo)
	go func() {
		if err := kafkaConsumer.Run(consumerCtx); err != nil {
			log.Error().Err(err).Msg("Kafka consumer exited with error")
		}
	}()
	log.Info().Msg("Kafka consumer started")

	// Set up Chi router with REST API
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"metrics-service"}`))
	})

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// Register metrics handlers
	handler := handlers.New(repo)
	handler.RegisterRoutes(r)

	// Register roster health routes
	handler.RegisterRosterRoutes(r)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    cfg.Server.Address(),
		Handler: r,
	}

	go func() {
		log.Info().Str("address", cfg.Server.Address()).Msg("Starting HTTP server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTP server failed")
			os.Exit(1)
		}
	}()

	// Start gRPC server with OTel tracing interceptors
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			commongrpc.TracingUnaryInterceptor("metrics-service"),
			commongrpc.LoggingUnaryInterceptor(log),
		),
	)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Error().Err(err).Msg("Failed to listen for gRPC")
		os.Exit(1)
	}

	go func() {
		log.Info().Str("address", cfg.GRPC.Address()).Msg("Starting gRPC server")
		if err := grpcServer.Serve(lis); err != nil {
			log.Error().Err(err).Msg("gRPC server failed")
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down gracefully...")

	consumerCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	httpServer.Shutdown(shutdownCtx)
	grpcServer.GracefulStop()

	log.Info().Msg("Metrics Service stopped")
}

func connectWithRetry(log *logging.Logger, dbCfg config.DatabaseConfig, maxRetries int, delay time.Duration) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.DBName, dbCfg.SSLMode)

	var db *sql.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Warn().Err(err).Int("attempt", i+1).Msg("Failed to open database")
			time.Sleep(delay)
			continue
		}

		db.SetMaxOpenConns(getEnvAsInt("DB_MAX_OPEN_CONNS", 25))
		db.SetMaxIdleConns(getEnvAsInt("DB_MAX_IDLE_CONNS", 5))
		db.SetConnMaxLifetime(5 * time.Minute)

		if err = db.Ping(); err != nil {
			log.Warn().Err(err).Int("attempt", i+1).Msg("Failed to ping database")
			db.Close()
			time.Sleep(delay)
			continue
		}

		return db, nil
	}

	return nil, fmt.Errorf("failed to connect after %d retries: %w", maxRetries, err)
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
