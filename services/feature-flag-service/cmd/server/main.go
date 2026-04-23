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
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	commongrpc "slate/libs/common-go/grpc"
	"slate/libs/common-go/logging"
	"slate/libs/common-go/tracing"

	pb "slate/services/feature-flag-service/api/proto"
	"slate/services/feature-flag-service/internal/cache"
	"slate/services/feature-flag-service/internal/config"
	flaggrpc "slate/services/feature-flag-service/internal/grpc"
	"slate/services/feature-flag-service/internal/repository"
	"slate/services/feature-flag-service/internal/service"
	"slate/services/feature-flag-service/migrations"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	log := logging.NewLogger("feature-flag-service", logLevel)
	log.Info().Msg("Starting Feature Flag Service (W3)")

	cfg, err := config.Load()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load config")
		os.Exit(1)
	}

	tracing.EnsureDefaultPropagator()
	if shutdown, err := tracing.InitTracer("feature-flag-service", cfg.Observability.OTLPEndpoint); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize tracing (continuing without tracing)")
	} else {
		log.Info().Msg("OpenTelemetry tracing initialized")
		defer shutdown()
	}

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
	if envPath := os.Getenv("MIGRATIONS_PATH"); envPath != "" {
		migrationsPath = envPath
	}
	if err := migrations.RunMigrations(db, migrationsPath); err != nil {
		log.Error().Err(err).Msg("Failed to run migrations")
		os.Exit(1)
	}
	log.Info().Msg("Migrations applied")

	store := repository.NewPostgresStore(db)
	evaluator := service.NewEvaluator()

	var flagCache cache.Cache = cache.NoopCache{}
	if cfg.Redis.Enabled {
		rdb := redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := rdb.Ping(pingCtx).Err(); err != nil {
			log.Warn().Err(err).Msg("Redis unreachable, falling back to noop cache")
			_ = rdb.Close()
		} else {
			flagCache = cache.NewRedisCache(rdb, cache.DefaultTTL)
			log.Info().Str("addr", cfg.Redis.Addr).Msg("Redis cache enabled")
			defer rdb.Close()
		}
		cancel()
	}

	grpcLis, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Error().Err(err).Msg("Failed to listen gRPC")
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			commongrpc.TracingUnaryInterceptor("feature-flag-service"),
			commongrpc.LoggingUnaryInterceptor(log),
		),
	)
	flagSrv := flaggrpc.NewServer(store, evaluator, flagCache)
	pb.RegisterFlagServiceServer(grpcSrv, flagSrv)
	grpc_health_v1.RegisterHealthServer(grpcSrv, grpchealth.NewServer())
	reflection.Register(grpcSrv)

	go func() {
		log.Info().Str("address", cfg.GRPC.Address()).Msg("gRPC server listening")
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Error().Err(err).Msg("gRPC server failed")
			os.Exit(1)
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","error":"%s"}`, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy"}`)
	})
	r.Handle("/metrics", promhttp.Handler())

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
	log.Info().Msg("Feature flag service stopped")
}
