package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	pb "slate/services/email-service/api/proto"
)

func main() {
	log.Println("Starting Email Service (In-Platform Messaging)")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize OpenTelemetry tracing via common-go
	shutdown, err := tracing.InitTracer("email-service", cfg.Observability.OTLPEndpoint)
	if err != nil {
		log.Printf("Failed to initialize tracing: %v (continuing without tracing)", err)
	} else {
		log.Println("OpenTelemetry tracing initialized via common-go")
		defer shutdown()
	}

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		log.Println("Waiting for database...")
		time.Sleep(time.Second)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Database not reachable: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// Run migrations
	migrationsPath := "./migrations"
	if envPath := os.Getenv("MIGRATIONS_PATH"); envPath != "" {
		migrationsPath = envPath
	}
	if err := migrations.RunMigrations(db, migrationsPath); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations applied")

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
		log.Println("Kafka consumers started")
	}

	// Setup HTTP server with Chi router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
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

	// Start gRPC server
	grpcLis, err := net.Listen("tcp", cfg.GRPC.Address())
	if err != nil {
		log.Fatalf("Failed to listen gRPC: %v", err)
	}

	grpcSrv := grpc.NewServer()
	emailGrpc := grpcserver.NewEmailServer(repo)
	pb.RegisterMessagingServiceServer(grpcSrv, emailGrpc)
	grpc_health_v1.RegisterHealthServer(grpcSrv, health.NewServer())
	reflection.Register(grpcSrv)

	go func() {
		log.Printf("gRPC server listening on %s", cfg.GRPC.Address())
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
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
		log.Printf("HTTP server listening on %s", cfg.Server.Address())
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	grpcSrv.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}

	log.Println("Email service stopped")
}
