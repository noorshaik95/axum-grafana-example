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

	"slate/services/tenant-service/internal/config"
	"slate/services/tenant-service/internal/repository"
	"slate/services/tenant-service/internal/service"
	"slate/services/tenant-service/migrations"
	"slate/services/tenant-service/pkg/circuitbreaker"
	"slate/services/tenant-service/pkg/database"
	"slate/services/tenant-service/pkg/logger"
	"slate/services/tenant-service/pkg/metrics"
	"slate/services/tenant-service/pkg/ratelimit"
	"slate/services/tenant-service/pkg/tracing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize logger
	logger.InitLogger()
	log.Info().Msg("Starting Tenant Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize tracing
	tp, err := tracing.InitTracer(tracing.Config{
		ServiceName:    "tenant-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OTLPInsecure:   true,
		SamplingRate:   1.0,
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize tracing, continuing without it")
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tracing.Shutdown(ctx, tp); err != nil {
				log.Error().Err(err).Msg("Failed to shutdown tracer")
			}
		}()
		log.Info().Msg("Tracing initialized successfully")
	}

	// Connect to database
	db, err := database.Connect(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()
	log.Info().Msg("Database connection established")

	// Run migrations
	if err := migrations.RunMigrations(db, "./migrations"); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}
	log.Info().Msg("Migrations completed successfully")

	// Initialize metrics collector
	metricsCollector := metrics.NewMetricsCollector()
	log.Info().Msg("Metrics collector initialized")

	// Initialize rate limiter
	rateLimiter, err := ratelimit.NewRateLimiter(&ratelimit.Config{
		CreateTenantLimit:  5,    // 5 tenant creations per hour
		CreateTenantWindow: 3600, // 1 hour
		OperationLimit:     100,  // 100 operations per minute
		OperationWindow:    60,   // 1 minute
		RedisAddr:          os.Getenv("REDIS_HOST") + ":6379",
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		RedisDB:            0,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize rate limiter")
	}
	log.Info().Msg("Rate limiter initialized")

	// Initialize circuit breakers for external services
	userServiceCB := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:         "user-service",
		MaxFailures:  5,
		Timeout:      5 * time.Second,
		ResetTimeout: 30 * time.Second,
	})
	emailServiceCB := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:         "email-service",
		MaxFailures:  3,
		Timeout:      10 * time.Second,
		ResetTimeout: 30 * time.Second,
	})
	log.Info().Msg("Circuit breakers initialized")

	// Initialize repository
	tenantRepo := repository.NewTenantRepository(db)

	// Initialize service clients with circuit breakers
	userClient := newUserServiceClient(cfg.Services.UserServiceURL, userServiceCB)
	emailClient := newEmailServiceClient(cfg.Services.EmailServiceURL, emailServiceCB, cfg.Email.Enabled)

	// Initialize tenant service
	tenantService := service.NewTenantService(
		tenantRepo,
		userClient,
		emailClient,
		metricsCollector,
		os.Getenv("BASE_SETUP_URL"),
	)

	// TODO: Register gRPC handlers once protobuf code is generated
	_ = tenantService // Temporary: prevent unused variable error
	_ = rateLimiter   // Temporary: prevent unused variable error

	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		metricsAddr := ":9090"
		log.Info().Str("address", metricsAddr).Msg("Starting metrics server")
		if err := http.ListenAndServe(metricsAddr, nil); err != nil {
			log.Error().Err(err).Msg("Metrics server failed")
		}
	}()

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// Add your interceptors here (logging, tracing, etc.)
		),
	)

	// Register health check
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Register reflection for debugging
	reflection.Register(grpcServer)

	// Start gRPC server
	lis, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen")
	}

	go func() {
		log.Info().Str("address", cfg.GRPC.Address()).Msg("Starting gRPC server")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Failed to serve")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down gracefully...")
	grpcServer.GracefulStop()
	log.Info().Msg("Tenant Service stopped")
}

// Mock clients for demonstration - in production, these would be real gRPC clients

type userServiceClient struct {
	url string
	cb  *circuitbreaker.CircuitBreaker
}

func newUserServiceClient(url string, cb *circuitbreaker.CircuitBreaker) service.UserServiceClient {
	return &userServiceClient{url: url, cb: cb}
}

func (c *userServiceClient) CreateUser(ctx context.Context, email, password, firstName, lastName string, roles []string) (string, error) {
	var userID string
	err := c.cb.Execute(func() error {
		// In production, this would make a real gRPC call to user-auth-service
		// For now, simulate user creation
		userID = fmt.Sprintf("user_%d", time.Now().Unix())
		return nil
	})
	return userID, err
}

type emailServiceClient struct {
	url     string
	cb      *circuitbreaker.CircuitBreaker
	enabled bool
}

func newEmailServiceClient(url string, cb *circuitbreaker.CircuitBreaker, enabled bool) service.EmailServiceClient {
	return &emailServiceClient{url: url, cb: cb, enabled: enabled}
}

func (c *emailServiceClient) SendWelcomeEmail(ctx context.Context, tenantName, adminEmail, adminName, setupURL, subdomain, tier string) error {
	if !c.enabled {
		log.Info().Msg("Email sending disabled, skipping welcome email")
		return nil
	}

	return c.cb.Execute(func() error {
		// In production, this would make a real gRPC call to email-service
		// For now, just log
		log.Info().
			Str("to", adminEmail).
			Str("tenant", tenantName).
			Str("setup_url", setupURL).
			Msg("Sending welcome email")
		return nil
	})
}
