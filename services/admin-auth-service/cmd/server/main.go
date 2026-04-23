// admin-auth-service entrypoint.
//
// Starts a gRPC server on :50060 for AdminAuthService and a plain HTTP server
// on :8090 for health / metrics. Trace propagation is handled by a single
// unary interceptor that extracts traceparent + x-request-id from incoming
// metadata and tags the span with both + admin id.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"

	commonlog "slate/libs/common-go/logging"
	commontracing "slate/libs/common-go/tracing"
	pb "slate/services/admin-auth-service/api/proto/adminauthpb"
	"slate/services/admin-auth-service/internal/audit"
	"slate/services/admin-auth-service/internal/config"
	grpchdl "slate/services/admin-auth-service/internal/grpc"
	jwtpkg "slate/services/admin-auth-service/internal/jwt"
	"slate/services/admin-auth-service/internal/kafka"
	"slate/services/admin-auth-service/internal/repository"
	"slate/services/admin-auth-service/internal/service"
	"slate/services/admin-auth-service/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load: %v\n", err)
		os.Exit(1)
	}

	log := commonlog.NewLogger(cfg.Service, cfg.LogLevel)
	log.Info().
		Str("env", cfg.Environment).
		Int("grpc_port", cfg.GRPCPort).
		Int("http_port", cfg.HTTPPort).
		Msg("starting admin-auth-service")

	shutdownTrace, err := commontracing.InitTracer(cfg.Service, cfg.OTLPEndpoint)
	if err != nil {
		log.Warn().Err(err).Msg("tracing init failed; continuing without OTLP")
	}
	defer shutdownTrace()
	commontracing.EnsureDefaultPropagator()

	db, err := sql.Open("postgres", cfg.DatabaseDSN)
	if err != nil {
		log.Error().Err(err).Msg("open database")
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Warn().Err(err).Msg("database unreachable at startup; migrations will retry")
	}

	migrationsPath := firstExisting(
		filepath.Join(".", "migrations"),
		"/app/migrations",
		filepath.Join(".", "services", "admin-auth-service", "migrations"),
	)
	if migrationsPath != "" {
		if err := migrations.RunMigrations(db, migrationsPath); err != nil {
			log.Error().Err(err).Str("path", migrationsPath).Msg("migrations failed")
			os.Exit(1)
		}
	}

	adminRepo := repository.NewAdminRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	roleRepo := repository.NewRoleRepository(db)

	kafkaProducer := kafka.NewProducer(cfg.KafkaBrokers)
	defer kafkaProducer.Close()

	auditRec := audit.NewRecorder(auditRepo, kafkaProducer)

	tokens := jwtpkg.NewTokenService(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL, cfg.ImpersonationTTL)
	if err := loadPlatformKeys(cfg, tokens); err != nil {
		log.Warn().Err(err).Msg("platform RSA keys not loaded; impersonation will fail until configured")
	}

	revoker := service.NewMemoryRevoker()

	svc := service.NewAdminService(adminRepo, auditRec, tokens, revoker, cfg.ImpersonationRedirectFn).
		WithReads(roleRepo, auditRepo)

	httpSrv := startHTTPServer(cfg.HTTPAddress(), adminRepo, log)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(traceExtractUnary(), actorExtractUnary(tokens)),
	)
	adminAuthSrv := grpchdl.NewAdminAuthServer(svc)
	if cfg.TenantDBDSN != "" {
		tenantDB, err := sql.Open("postgres", cfg.TenantDBDSN)
		if err != nil {
			log.Warn().Err(err).Msg("tenant DB open failed; slug lookups will be skipped")
		} else if pingErr := tenantDB.Ping(); pingErr != nil {
			log.Warn().Err(pingErr).Msg("tenant DB unreachable; slug lookups will be skipped")
			_ = tenantDB.Close()
		} else {
			adminAuthSrv.WithTenantDB(tenantDB)
			defer tenantDB.Close()
			log.Info().Msg("tenant DB connected for slug lookups")
		}
	}
	pb.RegisterAdminAuthServiceServer(grpcSrv, adminAuthSrv)
	healthSrv := health.NewServer()
	healthSrv.SetServingStatus(cfg.Service, healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)
	reflection.Register(grpcSrv)

	lis, err := net.Listen("tcp", cfg.GRPCAddress())
	if err != nil {
		log.Error().Err(err).Str("addr", cfg.GRPCAddress()).Msg("listen")
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Info().Msg("shutdown signal received")
		grpcSrv.GracefulStop()
	}()

	log.Info().Str("addr", cfg.GRPCAddress()).Msg("admin-auth-service gRPC listening")
	if err := grpcSrv.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		log.Error().Err(err).Msg("gRPC server exited with error")
		os.Exit(1)
	}
	log.Info().Msg("admin-auth-service stopped")
}

func loadPlatformKeys(cfg *config.Config, tokens *jwtpkg.TokenService) error {
	privKey, err := jwtpkg.LoadRSAPrivateKeyFromPath(cfg.PlatformPrivateKeyPath)
	if err != nil {
		return fmt.Errorf("load private key: %w", err)
	}
	if cfg.PlatformPublicKeyPath == "" {
		tokens.WithRSAKeys(privKey, &privKey.PublicKey)
		return nil
	}
	pubKey, err := jwtpkg.LoadRSAPublicKeyFromPath(cfg.PlatformPublicKeyPath)
	if err != nil {
		tokens.WithRSAKeys(privKey, &privKey.PublicKey)
		return fmt.Errorf("load public key: %w", err)
	}
	tokens.WithRSAKeys(privKey, pubKey)
	return nil
}

func traceExtractUnary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx = commontracing.ExtractTraceparent(ctx)
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get(commontracing.RequestIDHeader); len(vals) > 0 {
				ctx = commontracing.WithRequestID(ctx, vals[0])
			}
		}
		return handler(ctx, req)
	}
}

// actorExtractUnary pulls the admin id from the bearer token on the metadata
// when present. It's a best-effort read — Validate is the gate; this just
// tags span + audit rows with a caller identity when available.
func actorExtractUnary(tokens *jwtpkg.TokenService) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}
		for _, hdr := range []string{"authorization", "x-admin-token"} {
			vals := md.Get(hdr)
			if len(vals) == 0 {
				continue
			}
			token := vals[0]
			if len(token) > 7 && (token[:7] == "Bearer " || token[:7] == "bearer ") {
				token = token[7:]
			}
			if claims, err := tokens.ParseAdminToken(token); err == nil {
				ctx = grpchdl.WithActor(ctx, claims.UserID)
				break
			}
		}
		return handler(ctx, req)
	}
}

func startHTTPServer(addr string, repo *repository.AdminRepository, log *commonlog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := "ok"
		if err := repo.PingContext(r.Context()); err != nil {
			status = "degraded"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "service": "admin-auth-service"})
	})
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Info().Str("addr", addr).Msg("health HTTP listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("health HTTP server exited")
		}
	}()
	return srv
}

func firstExisting(paths ...string) string {
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}
