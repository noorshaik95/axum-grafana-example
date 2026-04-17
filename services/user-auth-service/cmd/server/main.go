package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	pb "slate/services/user-auth-service/api/proto"
	"slate/services/user-auth-service/api/proto/authpb"
	"slate/services/user-auth-service/internal/auth"
	"slate/services/user-auth-service/internal/auth/saml"
	"slate/services/user-auth-service/internal/auth/services"
	"slate/services/user-auth-service/internal/auth/strategies"
	"slate/services/user-auth-service/internal/config"
	grpcHandler "slate/services/user-auth-service/internal/grpc"
	"slate/services/user-auth-service/internal/health"
	"slate/services/user-auth-service/internal/repository"
	"slate/services/user-auth-service/internal/service"
	"slate/services/user-auth-service/migrations"
	"slate/services/user-auth-service/pkg/database"
	"slate/services/user-auth-service/pkg/jwt"
	"slate/services/user-auth-service/pkg/logger"
	"slate/services/user-auth-service/pkg/metrics"
	"slate/services/user-auth-service/pkg/ratelimit"
	"slate/services/user-auth-service/pkg/tracing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize logger with level from environment
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info" // Default to info level
	}
	log := logger.NewLogger(logLevel)
	log.Info().Str("log_level", logLevel).Msg("Starting User Auth Service")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load configuration")
		os.Exit(1)
	}

	// Initialize OpenTelemetry tracing
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint == "" {
		otlpEndpoint = "tempo:4317" // Default to Tempo
	}

	tracingCfg := tracing.Config{
		ServiceName:    "user-auth-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   otlpEndpoint,
		OTLPInsecure:   true, // Use insecure for local development
		SamplingRate:   1.0,  // Sample all traces
	}

	log.Info().Str("otlp_endpoint", otlpEndpoint).Msg("Initializing OpenTelemetry tracing")
	tp, err := tracing.InitTracer(tracingCfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize tracing")
		log.Info().Msg("Continuing without tracing - using no-op tracer")
		// Set a no-op tracer provider to prevent nil pointer issues
		tp = sdktrace.NewTracerProvider()
		otel.SetTracerProvider(tp)
	} else {
		log.Info().Msg("OpenTelemetry tracing initialized successfully")
	}
	// Always defer shutdown
	defer func() {
		if tp != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tracing.Shutdown(ctx, tp); err != nil {
				log.Error().Err(err).Msg("Failed to shutdown tracer provider")
			}
		}
	}()

	// Note: The tracer name is already set in tracingCfg.ServiceName above
	// Individual handlers will use the common-go tracing package for function-level spans

	// Connect to database
	db, err := database.NewPostgresDB(cfg.Database.DSN())
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to database")
		os.Exit(1)
	}
	defer db.Close()

	log.Info().Msg("Connected to PostgreSQL database")

	// Run migrations
	migrationsPath := filepath.Join(".", "migrations")
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		// Try alternative path for Docker
		migrationsPath = "/app/migrations"
	}

	if err := migrations.RunMigrations(db.DB, migrationsPath); err != nil {
		log.Error().Err(err).Msg("Failed to run migrations")
		os.Exit(1)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.DB)
	roleRepo := repository.NewRoleRepository(db.DB)
	groupRepo := repository.NewGroupRepository(db.DB)
	mfaRepo := repository.NewMFARepository(db.DB)

	// Ensure default roles exist
	if err := roleRepo.EnsureDefaultRoles(context.Background()); err != nil {
		log.Error().Err(err).Msg("Failed to ensure default roles")
		os.Exit(1)
	}

	// Validate JWT secret strength before initializing JWT service
	// Detect development mode from environment
	environment := os.Getenv("ENVIRONMENT")
	devMode := environment == "development" || environment == "dev" || os.Getenv("DEV_MODE") == "true"

	if devMode {
		log.Warn().Msg("Running in development mode - JWT secret validation will show warnings instead of errors")
	}

	secretValidator := jwt.NewSecretValidator(devMode)
	if err := secretValidator.ValidateSecret(cfg.JWT.SecretKey); err != nil {
		log.Error().Err(err).Msg("JWT secret validation failed")
		log.Error().Msg("CRITICAL: JWT secret does not meet security requirements")
		log.Error().Msg("Please update JWT_SECRET environment variable with a strong secret (32+ characters, mixed case, numbers, special characters)")
		os.Exit(1)
	}

	log.Info().Msg("JWT secret validation passed")

	// Initialize JWT service
	tokenService := jwt.NewTokenService(
		cfg.JWT.SecretKey,
		cfg.JWT.AccessTokenDuration,
		cfg.JWT.RefreshTokenDuration,
	)
	tokenServiceAdapter := jwt.NewTokenServiceAdapter(tokenService)

	// Initialize Prometheus metrics
	registry := prometheus.NewRegistry()
	metricsCollector := metrics.NewMetrics(registry)
	log.Info().Msg("Prometheus metrics initialized")

	// Start metrics HTTP server
	metricsAddr := ":9090" // Internal container port
	metricsServer := &http.Server{
		Addr:    metricsAddr,
		Handler: promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry}),
	}

	go func() {
		log.Info().Str("address", metricsAddr).Msg("Starting metrics HTTP server")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Metrics server failed")
		}
	}()

	// Start HTTP health check server on SERVER_PORT
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"user-auth-service"}`))
	})
	healthAddr := cfg.Server.Address()
	healthServer := &http.Server{
		Addr:              healthAddr,
		Handler:           healthMux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info().Str("address", healthAddr).Msg("Starting HTTP health server")
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Health server failed")
		}
	}()

	// Initialize Redis client for token blacklist and rate limiting
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	var redisClient *redis.Client
	var tokenBlacklist *jwt.TokenBlacklist

	if redisHost != "" && redisPort != "" {
		redisAddr := redisHost + ":" + redisPort
		log.Info().Str("redis_addr", redisAddr).Msg("Connecting to Redis")

		redisClient = redis.NewClient(&redis.Options{
			Addr: redisAddr,
		})

		// Test Redis connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Error().Err(err).Msg("Failed to connect to Redis")
			log.Info().Msg("Continuing without Redis (token blacklist and rate limiting disabled)")
			redisClient = nil
		} else {
			log.Info().Msg("Connected to Redis successfully")
			// Initialize token blacklist
			tokenBlacklist = jwt.NewTokenBlacklist(redisClient)
			log.Info().Msg("Token blacklist initialized")
		}
	} else {
		log.Info().Msg("Redis not configured (REDIS_HOST or REDIS_PORT not set)")
	}

	// Initialize OAuth and SAML repositories
	oauthRepo := repository.NewOAuthRepository(db.DB)
	samlRepo := repository.NewSAMLRepository(db.DB)

	// Initialize authentication strategy manager first (before UserService)
	log.Info().Str("auth_type", cfg.Auth.Type).Msg("Initializing authentication strategies")
	strategyManager := initializeAuthStrategiesWithoutNormal(cfg, userRepo, oauthRepo, samlRepo, roleRepo, tokenServiceAdapter, log)

	// Create adapter for strategy manager to avoid import cycles
	strategyManagerAdapter := service.NewStrategyManagerAdapter(strategyManager)

	// Initialize services with strategy manager adapter
	userService := service.NewUserService(userRepo, roleRepo, tokenServiceAdapter, tokenBlacklist, log, metricsCollector, strategyManagerAdapter)

	// Register Normal strategy now that UserService exists
	registerNormalStrategy(strategyManager, userService, log)
	log.Info().Msg("Authentication strategies initialized successfully")

	// Initialize rate limiter with fallback support
	var rateLimiter *ratelimit.FallbackRateLimiter
	rateLimitEnabled := os.Getenv("RATE_LIMIT_ENABLED")

	if rateLimitEnabled == "true" && redisHost != "" && redisPort != "" {
		log.Info().Msg("Initializing rate limiter with fallback support")

		// Configure rate limits from environment or use defaults
		loginLimit := ratelimit.RateLimit{
			MaxAttempts: 5,                // 5 attempts
			Window:      15 * time.Minute, // per 15 minutes
		}
		registerLimit := ratelimit.RateLimit{
			MaxAttempts: 3,             // 3 attempts
			Window:      1 * time.Hour, // per hour
		}

		redisAddr := redisHost + ":" + redisPort
		rateLimiter, err = ratelimit.NewFallbackRateLimiter(redisAddr, loginLimit, registerLimit, metricsCollector)
		if err != nil {
			log.Error().Err(err).Msg("Failed to initialize rate limiter")
			log.Info().Msg("Continuing without rate limiting")
		} else {
			if rateLimiter.IsUsingRedis() {
				log.Info().Msg("Rate limiter initialized with Redis")
			} else {
				log.Warn().Msg("Rate limiter initialized with in-memory fallback (Redis unavailable)")
			}
			defer rateLimiter.Close()
		}
	} else {
		log.Info().Msg("Rate limiting disabled (RATE_LIMIT_ENABLED not set or Redis not configured)")
	}

	// Initialize gRPC server with OpenTelemetry interceptors
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()), // OpenTelemetry stats handler (new API)
		grpc.ChainUnaryInterceptor(
			tracing.TracingUnaryInterceptor(),    // Explicit trace context extraction and span creation
			tracing.LoggingUnaryInterceptor(log), // Debug interceptor with logger instance
		),
	)

	// Register services
	// Convert rateLimiter to interface properly - if nil, pass nil interface
	var rateLimiterInterface ratelimit.RateLimiter
	if rateLimiter != nil {
		rateLimiterInterface = rateLimiter
	}
	userServiceServer := grpcHandler.NewUserServiceServer(userService, strategyManager, rateLimiterInterface, groupRepo, mfaRepo)
	pb.RegisterUserServiceServer(grpcServer, userServiceServer)

	// Register AuthService (wraps UserServiceServer for API Gateway compatibility)
	authServiceServer := grpcHandler.NewAuthServiceServer(userServiceServer)
	authpb.RegisterAuthServiceServer(grpcServer, authServiceServer)
	log.Info().Msg("AuthService registered successfully")

	// Register health check service
	healthChecker := health.NewHealthChecker(db.DB)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthChecker)
	log.Info().Msg("gRPC health check service registered")

	// Enable reflection for debugging with tools like grpcurl
	reflection.Register(grpcServer)

	// Start gRPC server
	lis, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Error().Err(err).Str("address", cfg.GRPC.Address()).Msg("Failed to listen")
		os.Exit(1)
	}

	log.Info().Str("address", cfg.GRPC.Address()).Msg("gRPC server listening")

	// Handle graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Info().Msg("Shutting down servers")

		// Shutdown health server
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := healthServer.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Failed to shutdown health server")
		}

		// Shutdown metrics server
		shutdownCtx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()
		if err := metricsServer.Shutdown(shutdownCtx2); err != nil {
			log.Error().Err(err).Msg("Failed to shutdown metrics server")
		}

		// Shutdown gRPC server
		grpcServer.GracefulStop()
	}()

	// Start serving
	if err := grpcServer.Serve(lis); err != nil {
		log.Error().Err(err).Msg("Failed to serve")
		os.Exit(1)
	}

	log.Info().Msg("Server stopped")
}

// initializeAuthStrategiesWithoutNormal initializes the strategy manager and registers
// OAuth and SAML strategies. Normal strategy is registered separately after UserService
// is created to avoid circular dependency.
func initializeAuthStrategiesWithoutNormal(
	cfg *config.Config,
	userRepo *repository.UserRepository,
	oauthRepo *repository.OAuthRepository,
	samlRepo *repository.SAMLRepository,
	roleRepo *repository.RoleRepository,
	tokenSvc service.TokenServiceInterface,
	log *logger.Logger,
) *auth.StrategyManager {
	// Get tracer for distributed tracing
	tracer := otel.Tracer("user-auth-service")

	// Create strategy manager
	manager := auth.NewStrategyManager(cfg, tracer, log)

	// Create SessionManager for OAuth and SAML strategies
	sessionMgr := services.NewSessionManager(oauthRepo, samlRepo, tracer, log)

	// Always initialize OAuth strategy (for multi-auth support)
	log.Info().
		Int("provider_count", len(cfg.OAuth.Providers)).
		Str("environment", cfg.Environment).
		Msg("Registering OAuth authentication strategy")

	oauthStrategy := strategies.NewOAuthAuthStrategy(
		&cfg.OAuth,
		userRepo,
		oauthRepo,
		nil, // userService will be set later if needed
		tokenSvc,
		sessionMgr,
		tracer,
		log,
		cfg.Environment,
	)
	if err := manager.RegisterStrategy(oauthStrategy); err != nil {
		log.Error().Err(err).Msg("Failed to register OAuth authentication strategy")
		os.Exit(1)
	}

	// Always initialize SAML strategy (for multi-auth support)
	log.Info().
		Int("provider_count", len(cfg.SAML.Providers)).
		Str("environment", cfg.Environment).
		Msg("Registering SAML authentication strategy")

	// Create HTTP client for metadata fetching
	httpClient := &http.Client{Timeout: 30 * time.Second}

	// Create SAML metadata cache
	metadataCache := saml.NewSAMLMetadataCache(samlRepo, httpClient, tracer, log)

	samlStrategy := strategies.NewSAMLAuthStrategy(
		&cfg.SAML,
		nil, // userService will be set later if needed
		userRepo,
		samlRepo,
		roleRepo,
		tokenSvc,
		sessionMgr,
		metadataCache,
		tracer,
		log,
		cfg.Environment,
	)
	if err := manager.RegisterStrategy(samlStrategy); err != nil {
		log.Error().Err(err).Msg("Failed to register SAML authentication strategy")
		os.Exit(1)
	}

	return manager
}

// registerNormalStrategy registers the Normal authentication strategy after UserService
// has been created. This avoids circular dependency issues.
func registerNormalStrategy(
	manager *auth.StrategyManager,
	userService *service.UserService,
	log *logger.Logger,
) {
	// Get tracer for distributed tracing
	tracer := otel.Tracer("user-auth-service")

	// Register NormalAuthStrategy
	log.Info().Msg("Registering Normal authentication strategy")
	normalStrategy := strategies.NewNormalAuthStrategy(userService, tracer, log)
	if err := manager.RegisterStrategy(normalStrategy); err != nil {
		log.Error().Err(err).Msg("Failed to register Normal authentication strategy")
		os.Exit(1)
	}

	// Log summary of enabled strategies
	activeAuthType := manager.GetActiveAuthType()
	log.Info().
		Str("primary_auth_type", string(activeAuthType)).
		Msg("All authentication strategies registered")
}
