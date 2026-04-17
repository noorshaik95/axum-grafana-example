package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/noorshaik95/slate/services/onboarding-service/internal/config"
	"github.com/noorshaik95/slate/services/onboarding-service/internal/models"
	"github.com/noorshaik95/slate/services/onboarding-service/internal/repository"
	"github.com/noorshaik95/slate/services/onboarding-service/migrations"
	"github.com/noorshaik95/slate/services/onboarding-service/pkg/database"
	"github.com/noorshaik95/slate/services/onboarding-service/pkg/kafka"
	"github.com/noorshaik95/slate/services/onboarding-service/pkg/logger"
	"github.com/noorshaik95/slate/services/onboarding-service/pkg/metrics"
	"github.com/noorshaik95/slate/services/onboarding-service/pkg/tracing"
)

func main() {
	if err := run(); err != nil {
		log.Error().Err(err).Msg("Worker failed")
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	_ = logger.NewLogger(cfg.LogLevel)
	log.Info().Msg("Starting Onboarding Worker")

	// Initialize tracing
	tp, err := tracing.InitTracer(tracing.Config{
		ServiceName:    "onboarding-worker",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   cfg.Telemetry.OTLPEndpoint,
		OTLPInsecure:   true,
		SamplingRate:   1.0,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize tracer: %w", err)
	}
	defer func() {
		if shutdownErr := tracing.Shutdown(context.Background(), tp); shutdownErr != nil {
			log.Error().Err(shutdownErr).Msg("Failed to shutdown tracer")
		}
	}()

	// Initialize metrics
	_ = metrics.NewMetrics(nil)

	// Connect to database
	db, err := database.NewPostgresDB(
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
		cfg.Database.ConnMaxLifetime,
		cfg.Database.ConnMaxIdleTime,
	)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Run migrations
	if err := migrations.RunMigrations(db.DB); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	// Initialize repository
	repo := repository.NewRepository(db.DB)

	// Initialize Kafka consumer
	consumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.ConsumerGroup,
		kafka.TopicOnboardingJobs,
	)
	defer consumer.Close()

	// Initialize Kafka producer for progress updates
	producer := kafka.NewProducer(cfg.Kafka.Brokers)
	defer producer.Close()

	// Create worker
	worker := &Worker{
		repo:     repo,
		producer: producer,
		cfg:      cfg,
	}

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info().Msg("Received shutdown signal")
		cancel()
	}()

	// Start consuming messages
	log.Info().Msg("Worker started, waiting for tasks...")
	if err := consumer.Consume(ctx, worker.HandleTask); err != nil {
		if err != context.Canceled {
			return fmt.Errorf("consumer error: %w", err)
		}
	}

	log.Info().Msg("Worker shut down gracefully")
	return nil
}

// Worker processes onboarding tasks
type Worker struct {
	repo     *repository.Repository
	producer *kafka.Producer
	cfg      *config.Config
}

// HandleTask processes a single onboarding task from Kafka
func (w *Worker) HandleTask(ctx context.Context, _, value []byte) error {
	var taskMsg models.OnboardingTaskMessage
	if err := json.Unmarshal(value, &taskMsg); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal task message")
		return err
	}

	log.Info().
		Str("task_id", taskMsg.TaskID).
		Str("job_id", taskMsg.JobID).
		Str("email", taskMsg.UserData.Email).
		Int("attempt", taskMsg.Attempt).
		Msg("Processing onboarding task")

	// Get task from database
	task, err := w.repo.GetTask(ctx, taskMsg.TaskID, taskMsg.TenantID)
	if err != nil {
		log.Error().Err(err).Str("task_id", taskMsg.TaskID).Msg("Failed to get task")
		return err
	}

	// Update task status to processing
	if err := w.repo.UpdateTaskStatus(ctx, task.ID, models.TaskStatusProcessing, "", ""); err != nil {
		log.Error().Err(err).Msg("Failed to update task status")
	}

	// Process the task based on role
	userID, err := w.processUser(ctx, task)
	if err != nil {
		log.Error().
			Err(err).
			Str("task_id", task.ID).
			Str("email", task.Email).
			Msg("Failed to process user")

		// Update task as failed
		if updateErr := w.repo.UpdateTaskStatus(ctx, task.ID, models.TaskStatusFailed, "", err.Error()); updateErr != nil {
			log.Error().Err(updateErr).Msg("Failed to update task status to failed")
		}

		// Retry if attempts remaining
		if taskMsg.Attempt < w.cfg.Worker.MaxRetries {
			time.Sleep(time.Duration(w.cfg.Worker.RetryBackoffMS) * time.Millisecond)
			taskMsg.Attempt++
			if publishErr := w.producer.PublishTask(ctx, task.ID, &taskMsg); publishErr != nil {
				log.Error().Err(publishErr).Msg("Failed to publish retry task")
			}
		}

		return err
	}

	// Update task as completed
	if err := w.repo.UpdateTaskStatus(ctx, task.ID, models.TaskStatusCompleted, userID, ""); err != nil {
		log.Error().Err(err).Msg("Failed to update task status")
	}

	// Create audit log
	w.createAuditLog(ctx, task, userID)

	// Publish progress update
	w.publishProgress(ctx, task.JobID, task.TenantID)

	log.Info().
		Str("task_id", task.ID).
		Str("user_id", userID).
		Str("email", task.Email).
		Msg("Task processed successfully")

	return nil
}

// processUser creates the user based on their role
func (w *Worker) processUser(ctx context.Context, task *models.Task) (string, error) {
	// This is a simplified implementation
	// In production, this would:
	// 1. Call user-auth-service via gRPC to create the user
	// 2. Assign appropriate roles based on task.Role
	// 3. Enroll in courses if student/instructor
	// 4. Set up resources (storage quota, workspace, calendar)
	// 5. Send welcome emails

	switch task.Role {
	case models.RoleStudent:
		return w.onboardStudent(ctx, task)
	case models.RoleInstructor:
		return w.onboardInstructor(ctx, task)
	default:
		return w.onboardGenericUser(ctx, task)
	}
}

func (w *Worker) onboardStudent(_ context.Context, task *models.Task) (string, error) {
	// Simplified student onboarding
	// TODO: Integrate with user-auth-service gRPC
	// TODO: Enroll in courses
	// TODO: Allocate 5GB storage
	// TODO: Send welcome email

	log.Info().
		Str("email", task.Email).
		Str("role", "student").
		Msg("Onboarding student")

	// Simulate user creation
	return "user-" + task.ID, nil
}

func (w *Worker) onboardInstructor(_ context.Context, task *models.Task) (string, error) {
	// Simplified instructor onboarding
	// TODO: Integrate with user-auth-service gRPC
	// TODO: Assign to courses
	// TODO: Grant content creation rights
	// TODO: Allocate 50GB storage
	// TODO: Send training materials

	log.Info().
		Str("email", task.Email).
		Str("role", "instructor").
		Msg("Onboarding instructor")

	// Simulate user creation
	return "user-" + task.ID, nil
}

func (w *Worker) onboardGenericUser(_ context.Context, task *models.Task) (string, error) {
	log.Info().
		Str("email", task.Email).
		Str("role", task.Role).
		Msg("Onboarding generic user")

	// Simulate user creation
	return "user-" + task.ID, nil
}

func (w *Worker) createAuditLog(ctx context.Context, task *models.Task, userID string) {
	eventData, err := json.Marshal(map[string]interface{}{
		"task_id": task.ID,
		"email":   task.Email,
		"role":    task.Role,
		"user_id": userID,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal audit log event data")
		return
	}

	auditLog := &models.AuditLog{
		TenantID:    task.TenantID,
		JobID:       sql.NullString{String: task.JobID, Valid: task.JobID != ""},
		TaskID:      sql.NullString{String: task.ID, Valid: task.ID != ""},
		EventType:   models.EventUserCreated,
		EventData:   string(eventData),
		PerformedBy: "system",
	}

	if err := w.repo.CreateAuditLog(ctx, auditLog); err != nil {
		log.Error().Err(err).Msg("Failed to create audit log")
	}
}

func (w *Worker) publishProgress(ctx context.Context, jobID, tenantID string) {
	// Get job statistics
	job, err := w.repo.GetJob(ctx, jobID, tenantID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get job for progress update")
		return
	}

	progressMsg := &models.ProgressUpdateMessage{
		JobID:              jobID,
		TenantID:           tenantID,
		CurrentStage:       "processing_users",
		ProgressPercentage: float64(job.ProcessedUsers) / float64(job.TotalUsers) * 100,
		ProcessedCount:     job.ProcessedUsers,
		TotalCount:         job.TotalUsers,
		SuccessCount:       job.SuccessfulUsers,
		FailedCount:        job.FailedUsers,
	}

	if err := w.producer.PublishProgress(ctx, jobID, progressMsg); err != nil {
		log.Error().Err(err).Msg("Failed to publish progress update")
	}
}
