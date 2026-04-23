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

	pb "slate/services/scheduling-service/api/proto"
	"slate/services/scheduling-service/internal/cache"
	"slate/services/scheduling-service/internal/config"
	grpcserver "slate/services/scheduling-service/internal/grpc"
	"slate/services/scheduling-service/internal/repository"
	"slate/services/scheduling-service/internal/service"
	"slate/services/scheduling-service/migrations"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	log := logging.NewLogger("scheduling-service", logLevel)
	log.Info().Msg("Starting Scheduling Service")

	cfg, err := config.Load()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load config")
		os.Exit(1)
	}

	// CONTRACTS.md trace.propagation — ensure W3C TraceContext + Baggage propagator
	// is installed globally so every outbound gRPC/Kafka call carries traceparent.
	tracing.EnsureDefaultPropagator()

	shutdown, err := tracing.InitTracer("scheduling-service", cfg.Observability.OTLPEndpoint)
	if err != nil {
		log.Warn().Err(err).Msg("Tracing disabled")
	} else {
		log.Info().Msg("Tracing initialized")
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
		log.Error().Err(err).Msg("Database unreachable")
		os.Exit(1)
	}

	migrationsPath := "./migrations"
	if p := os.Getenv("MIGRATIONS_PATH"); p != "" {
		migrationsPath = p
	}
	if err := migrations.RunMigrations(db, migrationsPath); err != nil {
		log.Error().Err(err).Msg("Migration failed")
		os.Exit(1)
	}
	log.Info().Msg("Migrations applied")

	var slotCache cache.Cache = cache.NoopCache{}
	if cfg.Redis.Enabled {
		rc := redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		if err := rc.Ping(context.Background()).Err(); err != nil {
			log.Warn().Err(err).Msg("Redis unreachable — falling back to NoopCache")
		} else {
			slotCache = cache.NewRedisCache(rc, cache.DefaultTTL)
			log.Info().Str("addr", cfg.Redis.Addr).Msg("Redis connected")
			defer rc.Close()
		}
	}

	repo := repository.NewScheduleRepository(db)
	sched := service.NewScheduler(repo, slotCache, cfg.Tenant.Slug)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","error":"%s"}`, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy"}`)
	})
	r.Handle("/metrics", promhttp.Handler())

	grpcLis, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Error().Err(err).Msg("gRPC listen failed")
		os.Exit(1)
	}
	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			// Correlation must run FIRST so span created by TracingUnaryInterceptor
			// sees x-request-id / x-tenant-slug already pinned on ctx.
			grpcserver.CorrelationUnaryInterceptor(),
			commongrpc.TracingUnaryInterceptor("scheduling-service"),
			commongrpc.LoggingUnaryInterceptor(log),
		),
	)
	pb.RegisterSchedulingServiceServer(grpcSrv, grpcserver.NewSchedulingServer(sched, cfg.Tenant.Slug))
	grpc_health_v1.RegisterHealthServer(grpcSrv, grpchealth.NewServer())
	reflection.Register(grpcSrv)

	go func() {
		log.Info().Str("address", cfg.GRPC.Address()).Msg("gRPC server listening")
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Error().Err(err).Msg("gRPC serve failed")
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
			log.Error().Err(err).Msg("HTTP serve failed")
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
	_ = httpSrv.Shutdown(ctx)
	log.Info().Msg("scheduling-service stopped")
}
