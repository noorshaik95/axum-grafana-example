package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/noorshaik95/slate/services/onboarding-service/internal/config"
	"github.com/noorshaik95/slate/services/onboarding-service/internal/repository"
	"github.com/noorshaik95/slate/services/onboarding-service/migrations"
	"github.com/noorshaik95/slate/services/onboarding-service/pkg/database"
	"github.com/noorshaik95/slate/services/onboarding-service/pkg/kafka"
	"github.com/noorshaik95/slate/services/onboarding-service/pkg/logger"
	"github.com/noorshaik95/slate/services/onboarding-service/pkg/metrics"
	"github.com/noorshaik95/slate/services/onboarding-service/pkg/tracing"
	ws "github.com/noorshaik95/slate/services/onboarding-service/pkg/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true // TODO: Implement proper CORS checking in production
	},
}

func main() {
	if err := run(); err != nil {
		log.Error().Err(err).Msg("Server failed")
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	_ = logger.NewLogger(cfg.LogLevel)
	log.Info().Msg("Starting Onboarding Service")

	// Initialize tracing
	tp, err := tracing.InitTracer(tracing.Config{
		ServiceName:    "onboarding-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   cfg.Telemetry.OTLPEndpoint,
		OTLPInsecure:   true,
		SamplingRate:   1.0,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize tracer: %w", err)
	}
	defer func() {
		if shutdownErr := tracing.Shutdown(context.Background(), tp); shutdownErr != nil {
			log.Error().Err(shutdownErr).Msg("Failed to shutdown tracer")
		}
	}()

	// Initialize metrics
	_ = metrics.NewMetrics(nil)

	// Connect to database
	db, err := database.NewPostgresDB(
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
		cfg.Database.ConnMaxLifetime,
		cfg.Database.ConnMaxIdleTime,
	)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Run migrations
	if err := migrations.RunMigrations(db.DB); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	// Initialize repository
	repo := repository.NewRepository(db.DB)

	// Initialize Kafka producer
	kafkaProducer := kafka.NewProducer(cfg.Kafka.Brokers)
	defer kafkaProducer.Close()

	// Initialize WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// Start gRPC server
	go startGRPCServer(cfg, repo, kafkaProducer)

	// Start HTTP server for health checks and WebSocket
	go startHTTPServer(cfg, hub)

	// Start metrics server
	go startMetricsServer()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("Shutting down service...")
	return nil
}

func startGRPCServer(
	cfg *config.Config,
	_ *repository.Repository,
	_ *kafka.Producer,
) {
	listener, err := net.Listen(
		"tcp",
		fmt.Sprintf("%s:%s", cfg.Server.GRPCHost, cfg.Server.GRPCPort),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen on gRPC port")
	}

	grpcServer := grpc.NewServer()

	// TODO: Register gRPC service handlers
	// pb.RegisterOnboardingServiceServer(grpcServer, &OnboardingServer{
	// 	repo:     repo,
	// 	producer: producer,
	// })

	// Enable gRPC reflection for dynamic discovery
	reflection.Register(grpcServer)

	log.Info().
		Str("host", cfg.Server.GRPCHost).
		Str("port", cfg.Server.GRPCPort).
		Msg("Starting gRPC server")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal().Err(err).Msg("Failed to serve gRPC")
	}
}

func startHTTPServer(cfg *config.Config, hub *ws.Hub) {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"healthy"}`)); err != nil {
			log.Error().Err(err).Msg("Failed to write health check response")
		}
	})

	// WebSocket endpoint for real-time progress updates
	mux.HandleFunc("/ws/jobs/", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, hub)
	})

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Info().
		Str("host", cfg.Server.Host).
		Str("port", cfg.Server.Port).
		Msg("Starting HTTP server")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Failed to start HTTP server")
	}
}

func startMetricsServer() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:              ":9090",
		Handler:           mux,
		ReadHeaderTimeout: 15 * time.Second,
	}

	log.Info().Str("port", "9090").Msg("Starting metrics server")

	if serveErr := server.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
		log.Fatal().Err(serveErr).Msg("Failed to start metrics server")
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, hub *ws.Hub) {
	// Extract job ID from URL path
	// Example: /ws/jobs/123e4567-e89b-12d3-a456-426614174000
	jobID := r.URL.Path[len("/ws/jobs/"):]
	if jobID == "" {
		http.Error(w, "Missing job ID", http.StatusBadRequest)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := "anonymous"

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade WebSocket connection")
		return
	}

	// Create and register client
	client := ws.NewClient(hub, conn, jobID, userID)
	client.Register()

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()
}
