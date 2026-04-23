package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	commongrpc "slate/libs/common-go/grpc"
	"slate/libs/common-go/logging"
	"slate/libs/common-go/tracing"
	aipb "slate/services/ai-service/api/proto"
	"slate/services/ai-service/internal/budget"
	"slate/services/ai-service/internal/cache"
	"slate/services/ai-service/internal/claude"
	"slate/services/ai-service/internal/config"
	grpcserver "slate/services/ai-service/internal/grpc"
	"slate/services/ai-service/internal/kafka"
	"slate/services/ai-service/internal/service"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	log := logging.NewLogger("ai-service", logLevel)
	log.Info().Msg("Starting AI Service")

	cfg, err := config.Load()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load config")
		os.Exit(1)
	}

	shutdown, err := tracing.InitTracer("ai-service", cfg.Observability.OTLPEndpoint)
	if err != nil {
		log.Warn().Err(err).Msg("tracing init failed (continuing)")
	} else {
		defer shutdown()
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		pingCancel()
		log.Warn().Err(err).Msg("redis ping failed — service will run degraded")
	} else {
		pingCancel()
		log.Info().Msg("connected to Redis")
	}
	defer rdb.Close()

	bt := budget.NewTracker(rdb, cfg.Budget.MonthlyTokenBudget)
	cc := cache.New(rdb)
	claudeClient := claude.NewClient(cfg.Anthropic.APIKey, cfg.Anthropic.DefaultModel, bt)

	welcomer := service.NewWelcomer(claudeClient, cfg.Anthropic.DefaultModel, cfg.Anthropic.WelcomeMaxTokens)
	grades := service.NewGradeProjector(cc)
	planner := service.NewStudyPlanner(claudeClient, cc, cfg.Anthropic.StudyPlanModel, cfg.Anthropic.StudyPlanMaxTokens)
	palette := service.NewCmdPalette(claudeClient, cc, cfg.Anthropic.DefaultModel, cfg.Anthropic.CmdPaletteMaxTokens)
	feedback := service.NewDraftFeedback()

	if cfg.Kafka.Enabled {
		consumer := kafka.NewInvalidationConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, grades, planner, log)
		cancel := consumer.Start(context.Background())
		defer cancel()
		log.Info().Strs("brokers", cfg.Kafka.Brokers).Msg("kafka invalidation consumer started")
	}

	lis, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Error().Err(err).Msg("listen failed")
		os.Exit(1)
	}
	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			commongrpc.TracingUnaryInterceptor("ai-service"),
			grpcserver.CorrelationInterceptor(),
			commongrpc.LoggingUnaryInterceptor(log),
		),
	)
	aipb.RegisterAiServiceServer(grpcSrv, &grpcserver.Server{
		Welcomer:  welcomer,
		Grades:    grades,
		StudyPlan: planner,
		Palette:   palette,
		Feedback:  feedback,
	})
	healthSrv := grpchealth.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("slate.ai.v1.AiService", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)
	reflection.Register(grpcSrv)

	go func() {
		log.Info().Str("address", cfg.GRPC.Address()).Msg("gRPC listening")
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error().Err(err).Msg("gRPC serve failed")
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutdown signal received")
	grpcSrv.GracefulStop()
	log.Info().Msg("ai-service stopped")
}
