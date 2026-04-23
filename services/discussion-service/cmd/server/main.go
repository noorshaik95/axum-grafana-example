package main

import (
	"database/sql"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	commongrpc "slate/libs/common-go/grpc"
	"slate/libs/common-go/logging"
	"slate/libs/common-go/tracing"

	pb "slate/services/discussion-service/api/proto"
	"slate/services/discussion-service/internal/config"
	grpcserver "slate/services/discussion-service/internal/grpc"
	"slate/services/discussion-service/internal/kafka"
	"slate/services/discussion-service/internal/repository"
	"slate/services/discussion-service/internal/userauth"
	"slate/services/discussion-service/migrations"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	log := logging.NewLogger("discussion-service", logLevel)
	log.Info().Msg("Starting discussion-service")

	cfg, err := config.Load()
	if err != nil {
		log.Error().Err(err).Msg("load config")
		os.Exit(1)
	}

	shutdown, err := tracing.InitTracer("discussion-service", cfg.Observability.OTLPEndpoint)
	if err != nil {
		log.Warn().Err(err).Msg("tracing init failed")
	} else {
		defer shutdown()
	}
	// W3C TraceContext + Baggage as the default propagator so inbound extract
	// and outbound inject (gRPC metadata + Kafka headers) use the same encoding
	// the rest of the platform expects.
	tracing.EnsureDefaultPropagator()

	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		log.Error().Err(err).Msg("open db")
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
		log.Info().Msg("waiting for db")
		time.Sleep(time.Second)
	}
	if err := db.Ping(); err != nil {
		log.Error().Err(err).Msg("db unreachable")
		os.Exit(1)
	}

	migrationsPath := "./migrations"
	if p := os.Getenv("MIGRATIONS_PATH"); p != "" {
		migrationsPath = p
	}
	if err := migrations.RunMigrations(db, migrationsPath); err != nil {
		log.Error().Err(err).Msg("migrations")
		os.Exit(1)
	}

	repo := repository.New(db)
	producer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Enabled)
	defer producer.Close()

	// Dial user-auth-service for @mention username resolution. On dial failure
	// we fall back to the NoopResolver so the service still boots in isolation
	// (e.g. dev compose without user-auth, or tests). When user-auth is reachable
	// but returns errors at runtime, those are surfaced per-RPC.
	var resolver userauth.Resolver = userauth.NoopResolver{}
	if cfg.UserAuth.Addr != "" {
		authConn, dialErr := grpc.NewClient(
			cfg.UserAuth.Addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		)
		if dialErr != nil {
			log.Warn().Err(dialErr).Str("addr", cfg.UserAuth.Addr).
				Msg("user-auth dial failed; @mention resolution will no-op")
		} else {
			defer authConn.Close()
			resolver = userauth.NewGRPCResolver(authConn)
			log.Info().Str("addr", cfg.UserAuth.Addr).Msg("user-auth gRPC resolver wired")
		}
	}

	srv := grpcserver.New(grpcserver.Options{
		Repo:         repo,
		Producer:     producer,
		UserResolver: resolver,
		TenantSlug:   cfg.TenantSlug,
	})

	lis, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Error().Err(err).Msg("listen")
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			commongrpc.TracingUnaryInterceptor("discussion-service"),
			commongrpc.LoggingUnaryInterceptor(log),
		),
	)
	pb.RegisterDiscussionServiceServer(grpcSrv, srv)
	grpc_health_v1.RegisterHealthServer(grpcSrv, grpchealth.NewServer())
	reflection.Register(grpcSrv)

	go func() {
		log.Info().Str("address", cfg.GRPC.Address()).Msg("gRPC listening")
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error().Err(err).Msg("gRPC server failed")
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down")
	grpcSrv.GracefulStop()
}
