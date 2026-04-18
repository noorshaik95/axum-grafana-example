package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commongrpc "slate/libs/common-go/grpc"
	"slate/libs/common-go/logging"
	"slate/libs/common-go/tracing"
	"slate/services/email-service/internal/config"
	grpcserver "slate/services/email-service/internal/grpc"
	"slate/services/email-service/internal/handlers"
	"slate/services/email-service/internal/kafka"
	"slate/services/email-service/internal/repository"
	"slate/services/email-service/migrations"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	pb "slate/services/email-service/api/proto"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	log := logging.NewLogger("email-service", logLevel)
	log.Info().Msg("Starting Email Service (In-Platform Messaging)")

	cfg, err := config.Load()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load config")
		os.Exit(1)
	}

	// Initialize OpenTelemetry tracing via common-go
	shutdown, err := tracing.InitTracer("email-service", cfg.Observability.OTLPEndpoint)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize tracing (continuing without tracing)")
	} else {
		log.Info().Msg("OpenTelemetry tracing initialized")
		defer shutdown()
	}

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		log.Error().Err(err).Msg("Failed to open database")
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		log.Info().Msg("Waiting for database...")
		time.Sleep(time.Second)
	}
	if err := db.Ping(); err != nil {
		log.Error().Err(err).Msg("Database not reachable")
		os.Exit(1)
	}
	log.Info().Msg("Connected to PostgreSQL")

	// Run migrations
	migrationsPath := "./migrations"
	if envPath := os.Getenv("MIGRATIONS_PATH"); envPath != "" {
		migrationsPath = envPath
	}
	if err := migrations.RunMigrations(db, migrationsPath); err != nil {
		log.Error().Err(err).Msg("Failed to run migrations")
		os.Exit(1)
	}
	log.Info().Msg("Migrations applied")

	// Initialize repository
	repo := repository.NewMessageRepository(db)

	// Initialize Kafka producer
	producer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Enabled)
	defer producer.Close()

	// Initialize Kafka consumer
	if cfg.Kafka.Enabled {
		consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, repo)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		consumer.Start(ctx)
		defer consumer.Close()
		log.Info().Msg("Kafka consumers started")
	}

	// Setup HTTP server with Chi router
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	handler := handlers.NewMessageHandler(repo, producer)

	r.Route("/messages", func(r chi.Router) {
		r.Post("/", handler.SendMessage)
		r.Post("/bulk", handler.BulkSend)
		r.Get("/inbox", handler.GetInbox)
		r.Get("/sent", handler.GetSent)
		r.Get("/unread-count", handler.GetUnreadCount)
		r.Get("/threads/{threadId}", handler.GetThread)
		r.Get("/{id}", handler.GetMessage)
		r.Post("/{id}/reply", handler.ReplyMessage)
		r.Patch("/{id}/read", handler.MarkRead)
		r.Patch("/{id}/unread", handler.MarkUnread)
		r.Delete("/{id}", handler.DeleteMessage)
	})

	// Discussion threads (course Q&A)
	discussionHandler := handlers.NewDiscussionHandler()
	discussionHandler.RegisterDiscussionRoutes(r)

	// Admin broadcast
	broadcastHandler := handlers.NewBroadcastHandler()
	r.Post("/admin/broadcast", broadcastHandler.CreateBroadcast)
	r.Get("/admin/broadcast/history", broadcastHandler.GetHistory)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","error":"%s"}`, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy"}`)
	})

	// Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// gRPC server with OTel stats handler + tracing interceptor
	grpcLis, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Error().Err(err).Msg("Failed to listen gRPC")
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			commongrpc.TracingUnaryInterceptor("email-service"),
			commongrpc.LoggingUnaryInterceptor(log),
		),
	)
	emailGrpc := grpcserver.NewEmailServer(repo)
	pb.RegisterMessagingServiceServer(grpcSrv, emailGrpc)
	grpc_health_v1.RegisterHealthServer(grpcSrv, grpchealth.NewServer())
	reflection.Register(grpcSrv)

	go func() {
		log.Info().Str("address", cfg.GRPC.Address()).Msg("gRPC server listening")
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Error().Err(err).Msg("gRPC server failed")
			os.Exit(1)
		}
	}()

	// Start HTTP server
	httpSrv := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("address", cfg.Server.Address()).Msg("HTTP server listening")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTP server failed")
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down...")

	grpcSrv.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Warn().Err(err).Msg("HTTP shutdown error")
	}

	log.Info().Msg("Email service stopped")
}
