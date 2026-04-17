package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"slate/services/assignment-grading-service/internal/config"
	grpchandler "slate/services/assignment-grading-service/internal/grpc"
	"slate/services/assignment-grading-service/internal/handlers"
	"slate/services/assignment-grading-service/internal/health"
	"slate/services/assignment-grading-service/internal/repository"
	"slate/services/assignment-grading-service/internal/service"
	"slate/services/assignment-grading-service/migrations"
	"slate/services/assignment-grading-service/pkg/database"
	"slate/services/assignment-grading-service/pkg/kafka"
	"slate/services/assignment-grading-service/pkg/logger"
	"slate/services/assignment-grading-service/pkg/metrics"
	"slate/services/assignment-grading-service/pkg/storage"
	"slate/services/assignment-grading-service/pkg/tracing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	pb "slate/services/assignment-grading-service/api/proto"
)

func main() {
	// Initialize logger
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	log := logger.NewLogger(logLevel)
	log.Info().Str("log_level", logLevel).Msg("Starting Assignment Grading Service")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load configuration")
		os.Exit(1)
	}

	// Initialize OpenTelemetry tracing
	tracingCfg := tracing.Config{
		ServiceName:    "assignment-grading-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   cfg.Observability.OTLPEndpoint,
		OTLPInsecure:   cfg.Observability.OTLPInsecure,
		SamplingRate:   1.0,
	}

	log.Info().Str("otlp_endpoint", cfg.Observability.OTLPEndpoint).Msg("Initializing OpenTelemetry tracing")
	tp, err := tracing.InitTracer(tracingCfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize tracing")
		log.Info().Msg("Continuing without tracing")
	} else {
		log.Info().Msg("OpenTelemetry tracing initialized successfully")
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if shutdownErr := tracing.Shutdown(ctx, tp); shutdownErr != nil {
				log.Error().Err(shutdownErr).Msg("Failed to shutdown tracer provider")
			}
		}()
	}

	// Connect to database
	db, err := database.NewPostgresDB(cfg.Database.DSN())
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to database")
		if tp != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if shutdownErr := tracing.Shutdown(shutdownCtx, tp); shutdownErr != nil {
				log.Error().Err(shutdownErr).Msg("Failed to shutdown tracer")
			}
			cancel()
		}
		os.Exit(1)
	}
	defer db.Close()

	log.Info().Msg("Connected to PostgreSQL database")

	// Run migrations
	migrationsPath := filepath.Join(".", "migrations")
	if _, statErr := os.Stat(migrationsPath); os.IsNotExist(statErr) {
		migrationsPath = "/app/migrations"
	}

	if migrationErr := migrations.RunMigrations(db.DB, migrationsPath); migrationErr != nil {
		log.Error().Err(migrationErr).Msg("Failed to run migrations")
		os.Exit(1)
	}

	// Initialize Prometheus metrics
	registry := prometheus.NewRegistry()
	metricsCollector := metrics.NewMetrics(registry)
	log.Info().Msg("Prometheus metrics initialized")
	_ = metricsCollector

	// Start metrics HTTP server
	metricsAddr := fmt.Sprintf(":%d", cfg.Observability.MetricsPort)
	metricsServer := &http.Server{
		Addr:              metricsAddr,
		Handler:           promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info().Str("address", metricsAddr).Msg("Starting metrics HTTP server")
		if serverErr := metricsServer.ListenAndServe(); serverErr != nil && serverErr != http.ErrServerClosed {
			log.Error().Err(serverErr).Msg("Metrics server failed")
		}
	}()

	// Initialize Kafka producer
	kafkaProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.Enabled)
	if cfg.Kafka.Enabled {
		log.Info().Strs("brokers", cfg.Kafka.Brokers).Msg("Kafka producer initialized")
		defer kafkaProducer.Close()
	} else {
		log.Info().Msg("Kafka is disabled")
	}

	// Initialize file storage
	fileStorage, err := storage.NewLocalFileStorage(cfg.Storage.LocalPath, cfg.Storage.MaxSize)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize file storage")
		os.Exit(1)
	}
	log.Info().Str("path", cfg.Storage.LocalPath).Msg("File storage initialized")

	// Initialize repositories
	assignmentRepo := repository.NewAssignmentRepository(db.DB)
	submissionRepo := repository.NewSubmissionRepository(db.DB)
	gradeRepo := repository.NewGradeRepository(db.DB)
	gradingRuleRepo := repository.NewGradingRuleRepository(db.DB)

	// Initialize services
	assignmentService := service.NewAssignmentService(assignmentRepo, kafkaProducer)
	submissionService := service.NewSubmissionService(assignmentRepo, submissionRepo, fileStorage, kafkaProducer)
	gradingService := service.NewGradingServiceWithDB(assignmentRepo, submissionRepo, gradeRepo, kafkaProducer, db.DB)
	gradebookService := service.NewGradebookServiceFull(assignmentRepo, submissionRepo, gradeRepo, gradingRuleRepo)
	gradingRuleService := service.NewGradingRuleService(gradingRuleRepo)

	log.Info().Msg("Services initialized")

	// Initialize Kafka consumer
	if cfg.Kafka.Enabled {
		consumer := kafka.NewConsumer(kafka.ConsumerConfig{
			Brokers: cfg.Kafka.Brokers,
			GroupID: "assignment-grading-service",
			Topics:  []string{"course-events", "user-events"},
			Enabled: true,
		})

		// Handle course.deleted: soft-delete all assignments for the course
		consumer.RegisterHandler("course.deleted", func(ctx context.Context, event kafka.Event) error {
			courseID, ok := event.Data["course_id"].(string)
			if !ok {
				return fmt.Errorf("missing course_id in event data")
			}
			log.Info().Str("course_id", courseID).Msg("Handling course.deleted event")
			return assignmentRepo.SoftDeleteByCourse(ctx, courseID)
		})

		// Handle user.enrolled: create empty grade records
		consumer.RegisterHandler("user.enrolled", func(ctx context.Context, event kafka.Event) error {
			studentID, _ := event.Data["student_id"].(string)
			courseID, _ := event.Data["course_id"].(string)
			tenantID, _ := event.Data["tenant_id"].(string)
			if studentID == "" || courseID == "" {
				return fmt.Errorf("missing student_id or course_id in event data")
			}
			log.Info().Str("student_id", studentID).Str("course_id", courseID).Msg("Handling user.enrolled event")

			assignments, _, err := assignmentRepo.ListByCourse(ctx, courseID, 1, 1000)
			if err != nil {
				return fmt.Errorf("failed to list assignments: %w", err)
			}
			for _, a := range assignments {
				if err := gradeRepo.CreateEmpty(ctx, tenantID, a.ID, studentID, courseID); err != nil {
					log.Error().Err(err).Str("assignment_id", a.ID).Msg("Failed to create empty grade")
				}
			}
			return nil
		})

		consumerCtx, consumerCancel := context.WithCancel(context.Background())
		consumer.Start(consumerCtx)
		defer func() {
			consumerCancel()
			consumer.Close()
		}()
		log.Info().Msg("Kafka consumer started")
	}

	// Initialize gRPC handlers
	assignmentHandler := grpchandler.NewAssignmentServiceServer(assignmentService)
	submissionHandler := grpchandler.NewSubmissionServiceServer(submissionService)
	gradingHandler := grpchandler.NewGradingServiceServer(gradingService)
	gradebookHandler := grpchandler.NewGradebookServiceServer(gradebookService)

	log.Info().Msg("gRPC handlers initialized")

	// Initialize gRPC server
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			tracing.TracingUnaryInterceptor(),
			tracing.LoggingUnaryInterceptor(log),
		),
	)

	pb.RegisterAssignmentServiceServer(grpcServer, assignmentHandler)
	pb.RegisterSubmissionServiceServer(grpcServer, submissionHandler)
	pb.RegisterGradingServiceServer(grpcServer, gradingHandler)
	pb.RegisterGradebookServiceServer(grpcServer, gradebookHandler)

	healthChecker := health.NewHealthChecker(db.DB)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthChecker)
	reflection.Register(grpcServer)

	// Start gRPC server
	lis, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Error().Err(err).Str("address", cfg.GRPC.Address()).Msg("Failed to listen")
		os.Exit(1)
	}

	log.Info().Str("address", cfg.GRPC.Address()).Msg("gRPC server listening")

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Error().Err(err).Msg("gRPC server failed")
		}
	}()

	// Start REST API server
	restRouter := handlers.NewRouter(assignmentService, submissionService, gradingService, gradebookService, gradingRuleService)
	restAddr := cfg.Server.Address()
	restServer := &http.Server{
		Addr:              restAddr,
		Handler:           restRouter.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		log.Info().Str("address", restAddr).Msg("Starting REST API server")
		if serverErr := restServer.ListenAndServe(); serverErr != nil && serverErr != http.ErrServerClosed {
			log.Error().Err(serverErr).Msg("REST server failed")
		}
	}()

	// Handle graceful shutdown
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	<-sigint

	log.Info().Msg("Shutting down servers")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := restServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown REST server")
	}

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown metrics server")
	}

	grpcServer.GracefulStop()

	log.Info().Msg("Server stopped")
}
