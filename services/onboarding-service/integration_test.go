//go:build integration
// +build integration

package onboarding_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/noorshaik95/slate/services/onboarding-service/internal/models"
	"github.com/noorshaik95/slate/services/onboarding-service/internal/repository"
	"github.com/noorshaik95/slate/services/onboarding-service/migrations"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	// Setup test database
	var err error
	testDB, err = setupTestDatabase()
	if err != nil {
		fmt.Printf("Failed to setup test database: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup - must happen before os.Exit
	cleanupTestDatabase(testDB)

	os.Exit(code)
}

func setupTestDatabase() (*sql.DB, error) {
	// Use environment variables or defaults
	host := getEnvOrDefault("TEST_DB_HOST", "localhost")
	port := getEnvOrDefault("TEST_DB_PORT", "5432")
	user := getEnvOrDefault("TEST_DB_USER", "postgres")
	password := getEnvOrDefault("TEST_DB_PASSWORD", "postgres")
	dbname := getEnvOrDefault("TEST_DB_NAME", "onboarding_test")

	// Connect to postgres database to create test database
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		host, port, user, password)
	adminDB, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer adminDB.Close()

	// Drop test database if exists
	_, err = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbname))
	if err != nil {
		return nil, fmt.Errorf("failed to drop test database: %w", err)
	}

	// Create test database
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbname))
	if err != nil {
		return nil, fmt.Errorf("failed to create test database: %w", err)
	}

	// Connect to test database
	connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	// Run migrations
	if err := migrations.RunMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func cleanupTestDatabase(db *sql.DB) {
	// Test database will be dropped in next test run
	// Just close the connection for now
	db.Close()
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// testTenantID is the default tenant created by migrations
const testTenantID = "00000000-0000-0000-0000-000000000001"

func TestIntegration_JobLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewRepository(testDB)

	// Create a job
	job := &models.Job{
		TenantID:    testTenantID,
		Name:        "Integration Test Job",
		Description: "Testing job lifecycle",
		SourceType:  models.SourceTypeCSV,
		Status:      models.JobStatusPending,
		TotalUsers:  100,
		CreatedBy:   "test-user",
	}

	err := repo.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	if job.ID == "" {
		t.Error("Job ID should be generated")
	}

	// Retrieve the job
	retrieved, err := repo.GetJob(ctx, job.ID, job.TenantID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	if retrieved.Name != job.Name {
		t.Errorf("Job name mismatch: got %v, want %v", retrieved.Name, job.Name)
	}
	if retrieved.Status != models.JobStatusPending {
		t.Errorf("Job status = %v, want %v", retrieved.Status, models.JobStatusPending)
	}

	// Update job status
	err = repo.UpdateJobStatus(ctx, job.ID, models.JobStatusProcessing, map[string]int{
		"processed":  50,
		"successful": 45,
		"failed":     5,
	})
	if err != nil {
		t.Fatalf("Failed to update job status: %v", err)
	}

	// Verify update
	updated, err := repo.GetJob(ctx, job.ID, job.TenantID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	if updated.Status != models.JobStatusProcessing {
		t.Errorf("Updated status = %v, want %v", updated.Status, models.JobStatusProcessing)
	}
	if updated.ProcessedUsers != 50 {
		t.Errorf("ProcessedUsers = %v, want 50", updated.ProcessedUsers)
	}
	if updated.SuccessfulUsers != 45 {
		t.Errorf("SuccessfulUsers = %v, want 45", updated.SuccessfulUsers)
	}
	if updated.FailedUsers != 5 {
		t.Errorf("FailedUsers = %v, want 5", updated.FailedUsers)
	}
}

func TestIntegration_TaskLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewRepository(testDB)

	// Create a job first
	job := &models.Job{
		TenantID:   testTenantID,
		Name:       "Task Test Job",
		SourceType: models.SourceTypeCSV,
		Status:     models.JobStatusPending,
		TotalUsers: 10,
		CreatedBy:  "test-user",
	}
	err := repo.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Create a task
	task := &models.Task{
		JobID:         job.ID,
		TenantID:      job.TenantID,
		Email:         "test@example.com",
		FirstName:     "Test",
		LastName:      "User",
		Role:          models.RoleStudent,
		Department:    "Computer Science",
		CourseCodes:   []string{"CS101", "CS102"},
		PreferredLang: "en",
		Status:        models.TaskStatusPending,
	}

	err = repo.CreateTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	if task.ID == "" {
		t.Error("Task ID should be generated")
	}

	// Retrieve the task
	retrieved, err := repo.GetTask(ctx, task.ID, task.TenantID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrieved.Email != task.Email {
		t.Errorf("Task email mismatch: got %v, want %v", retrieved.Email, task.Email)
	}
	if retrieved.Role != models.RoleStudent {
		t.Errorf("Task role = %v, want %v", retrieved.Role, models.RoleStudent)
	}
	if len(retrieved.CourseCodes) != 2 {
		t.Errorf("CourseCodes length = %v, want 2", len(retrieved.CourseCodes))
	}

	// Update task status
	err = repo.UpdateTaskStatus(ctx, task.ID, models.TaskStatusCompleted, "user-123", "")
	if err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	// Verify update
	updated, err := repo.GetTask(ctx, task.ID, task.TenantID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	if updated.Status != models.TaskStatusCompleted {
		t.Errorf("Updated status = %v, want %v", updated.Status, models.TaskStatusCompleted)
	}
	if !updated.UserID.Valid || updated.UserID.String != "user-123" {
		t.Errorf("UserID = %v, want user-123", updated.UserID)
	}
	if !updated.ProcessedAt.Valid {
		t.Error("ProcessedAt should be set")
	}
}

func TestIntegration_BatchTaskCreation(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewRepository(testDB)

	// Create a job
	job := &models.Job{
		TenantID:   testTenantID,
		Name:       "Batch Test Job",
		SourceType: models.SourceTypeCSV,
		Status:     models.JobStatusPending,
		TotalUsers: 100,
		CreatedBy:  "test-user",
	}
	err := repo.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Create 100 tasks in batch
	tasks := make([]*models.Task, 100)
	for i := 0; i < 100; i++ {
		tasks[i] = &models.Task{
			JobID:         job.ID,
			TenantID:      job.TenantID,
			Email:         fmt.Sprintf("user%d@example.com", i),
			FirstName:     fmt.Sprintf("User%d", i),
			LastName:      "Test",
			Role:          models.RoleStudent,
			PreferredLang: "en",
			Status:        models.TaskStatusPending,
		}
	}

	// Measure batch insertion time
	start := time.Now()
	err = repo.CreateTasksBatch(ctx, tasks)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to create tasks batch: %v", err)
	}

	t.Logf("Created 100 tasks in %v", duration)

	// Verify tasks were created
	pendingTasks, err := repo.ListPendingTasks(ctx, job.ID, 150)
	if err != nil {
		t.Fatalf("Failed to list pending tasks: %v", err)
	}

	if len(pendingTasks) != 100 {
		t.Errorf("Expected 100 pending tasks, got %d", len(pendingTasks))
	}

	// Verify task IDs were generated
	for i, task := range tasks {
		if task.ID == "" {
			t.Errorf("Task %d ID not generated", i)
		}
	}
}

func TestIntegration_AuditLog(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewRepository(testDB)

	// Create a job
	job := &models.Job{
		TenantID:   testTenantID,
		Name:       "Audit Test Job",
		SourceType: models.SourceTypeAPI,
		Status:     models.JobStatusPending,
		TotalUsers: 1,
		CreatedBy:  "test-user",
	}
	err := repo.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Create audit log
	auditLog := &models.AuditLog{
		TenantID:    job.TenantID,
		JobID:       sql.NullString{String: job.ID, Valid: true},
		EventType:   models.EventJobCreated,
		EventData:   `{"job_name":"Audit Test Job"}`,
		PerformedBy: "test-user",
		IPAddress:   "192.168.1.1",
		UserAgent:   "Test/1.0",
	}

	err = repo.CreateAuditLog(ctx, auditLog)
	if err != nil {
		t.Fatalf("Failed to create audit log: %v", err)
	}

	if auditLog.ID == "" {
		t.Error("Audit log ID should be generated")
	}

	// Verify audit log is immutable (cannot update)
	_, err = testDB.Exec("UPDATE onboarding_audit_logs SET event_type = 'modified' WHERE id = $1", auditLog.ID)
	if err == nil {
		t.Error("Audit log should be immutable, but UPDATE succeeded")
	}

	// Verify audit log cannot be deleted
	_, err = testDB.Exec("DELETE FROM onboarding_audit_logs WHERE id = $1", auditLog.ID)
	if err == nil {
		t.Error("Audit log should be immutable, but DELETE succeeded")
	}
}

func TestIntegration_JobProgress(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewRepository(testDB)

	// Create a job
	job := &models.Job{
		TenantID:   testTenantID,
		Name:       "Progress Test Job",
		SourceType: models.SourceTypeCSV,
		Status:     models.JobStatusProcessing,
		TotalUsers: 1000,
		CreatedBy:  "test-user",
	}
	err := repo.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Update progress
	progress := &models.JobProgress{
		JobID:              job.ID,
		CurrentStage:       "validating_users",
		ProgressPercentage: 25.5,
		CurrentTaskID:      "task-123",
		Metrics:            `{"validated":255,"errors":5}`,
	}

	err = repo.UpdateJobProgress(ctx, progress)
	if err != nil {
		t.Fatalf("Failed to update job progress: %v", err)
	}

	// Retrieve progress
	retrieved, err := repo.GetJobProgress(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to get job progress: %v", err)
	}

	if retrieved.CurrentStage != "validating_users" {
		t.Errorf("CurrentStage = %v, want validating_users", retrieved.CurrentStage)
	}
	if retrieved.ProgressPercentage != 25.5 {
		t.Errorf("ProgressPercentage = %v, want 25.5", retrieved.ProgressPercentage)
	}

	// Update progress again (should upsert)
	progress.CurrentStage = "creating_users"
	progress.ProgressPercentage = 75.0
	err = repo.UpdateJobProgress(ctx, progress)
	if err != nil {
		t.Fatalf("Failed to update job progress (2nd time): %v", err)
	}

	// Verify upsert worked
	updated, err := repo.GetJobProgress(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to get updated job progress: %v", err)
	}

	if updated.CurrentStage != "creating_users" {
		t.Errorf("Updated CurrentStage = %v, want creating_users", updated.CurrentStage)
	}
	if updated.ProgressPercentage != 75.0 {
		t.Errorf("Updated ProgressPercentage = %v, want 75.0", updated.ProgressPercentage)
	}
}
