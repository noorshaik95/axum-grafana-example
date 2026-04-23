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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	commongrpc "slate/libs/common-go/grpc"
	"slate/libs/common-go/logging"
	"slate/libs/common-go/tracing"
	pb "slate/services/incident-service/api/proto"
	"slate/services/incident-service/internal/cache"
	"slate/services/incident-service/internal/config"
	grpcserver "slate/services/incident-service/internal/grpc"
	"slate/services/incident-service/internal/handlers"
	"slate/services/incident-service/internal/kafka"
	"slate/services/incident-service/internal/repository"
	"slate/services/incident-service/migrations"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	log := logging.NewLogger("incident-service", logLevel)
	log.Info().Msg("Starting Incident Service")

	cfg, err := config.Load()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load config")
		os.Exit(1)
	}

	// --- tracing ---------------------------------------------------------
	tracing.EnsureDefaultPropagator()
	shutdown, err := tracing.InitTracer("incident-service", cfg.Observability.OTLPEndpoint)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize tracing (continuing without)")
	} else {
		defer shutdown()
	}

	// --- postgres --------------------------------------------------------
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

	migrationsPath := "./migrations"
	if env := os.Getenv("MIGRATIONS_PATH"); env != "" {
		migrationsPath = env
	}
	if err := migrations.RunMigrations(db, migrationsPath); err != nil {
		log.Error().Err(err).Msg("Failed to run migrations")
		os.Exit(1)
	}

	// --- repo + redis + producer ----------------------------------------
	repo := repository.New(db)

	var redisCache *cache.Cache
	if cfg.Redis.Enabled {
		rdb := redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			log.Warn().Err(err).Msg("Redis unreachable — cache disabled")
		} else {
			redisCache = cache.New(rdb)
			log.Info().Str("addr", cfg.Redis.Addr).Msg("Redis cache enabled")
		}
	}

	producer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Enabled)
	defer producer.Close()

	grpcSvc := grpcserver.NewServer(repo, producer, redisCache)

	// --- kafka consumer --------------------------------------------------
	if cfg.Kafka.Enabled {
		consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, repo, producer, redisCache)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		consumer.Start(ctx)
		defer consumer.Close()
		log.Info().Msg("Kafka consumer started: metrics.threshold_breached")
	}

	// --- HTTP router -----------------------------------------------------
	httpHandler := handlers.NewHTTPHandler(grpcSvc)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(requestIDMiddleware())

	r.Get("/api/status", httpHandler.PublicStatus)
	r.Get("/status", httpHandler.PublicStatus) // convenience

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","error":%q}`, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy"}`)
	})
	r.Handle("/metrics", promhttp.Handler())

	// --- gRPC server -----------------------------------------------------
	grpcLis, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Error().Err(err).Msg("Failed to listen gRPC")
		os.Exit(1)
	}
	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			commongrpc.TracingUnaryInterceptor("incident-service"),
			commongrpc.LoggingUnaryInterceptor(log),
		),
	)
	pb.RegisterIncidentServiceServer(grpcSrv, grpcSvc)
	grpc_health_v1.RegisterHealthServer(grpcSrv, grpchealth.NewServer())
	reflection.Register(grpcSrv)

	go func() {
		log.Info().Str("address", cfg.GRPC.Address()).Msg("gRPC server listening")
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Error().Err(err).Msg("gRPC server failed")
			os.Exit(1)
		}
	}()

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
	log.Info().Msg("Incident service stopped")
}

// requestIDMiddleware attaches X-Request-ID from inbound HTTP headers onto the
// request context (so outbound calls and Kafka publishes forward it), echoing
// it back on the response. Generates one if missing.
func requestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-ID")
			if reqID == "" {
				reqID = fmt.Sprintf("incident-%d", time.Now().UnixNano())
			}
			ctx := tracing.WithRequestID(r.Context(), reqID)
			if slug := r.Header.Get("X-Tenant-Slug"); slug != "" {
				ctx = tracing.WithTenantSlug(ctx, slug)
			}
			w.Header().Set("X-Request-ID", reqID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
