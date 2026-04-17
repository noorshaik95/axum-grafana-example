package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/noorshaik95/slate/services/onboarding-service/internal/models"
)

// Repository handles all database operations
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Pagination parameter indices
const (
	limitArgIndex  = 1 // Index offset for LIMIT parameter in dynamic queries
	offsetArgIndex = 2 // Index offset for OFFSET parameter in dynamic queries
)

// ===== Job Operations =====

func (r *Repository) CreateJob(ctx context.Context, job *models.Job) error {
	job.ID = uuid.New().String()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	query := `
		INSERT INTO onboarding_jobs (
			id, tenant_id, name, description, source_type, source_reference,
			status, total_users, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.TenantID, job.Name, job.Description, job.SourceType,
		job.SourceReference, job.Status, job.TotalUsers, job.CreatedBy,
		job.CreatedAt, job.UpdatedAt,
	)

	return err
}

func (r *Repository) GetJob(ctx context.Context, jobID, tenantID string) (*models.Job, error) {
	query := `
		SELECT id, tenant_id, name, description, source_type, source_reference,
		       status, total_users, processed_users, successful_users, failed_users,
		       error_summary, created_by, started_at, completed_at, created_at, updated_at
		FROM onboarding_jobs
		WHERE id = $1 AND tenant_id = $2
	`

	job := &models.Job{}
	err := r.db.QueryRowContext(ctx, query, jobID, tenantID).Scan(
		&job.ID, &job.TenantID, &job.Name, &job.Description, &job.SourceType,
		&job.SourceReference, &job.Status, &job.TotalUsers, &job.ProcessedUsers,
		&job.SuccessfulUsers, &job.FailedUsers, &job.ErrorSummary, &job.CreatedBy,
		&job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found")
	}

	return job, err
}

func (r *Repository) UpdateJobStatus(ctx context.Context, jobID, status string, counts map[string]int) error {
	query := `
		UPDATE onboarding_jobs
		SET status = $1, processed_users = $2, successful_users = $3,
		    failed_users = $4, updated_at = $5
		WHERE id = $6
	`

	_, err := r.db.ExecContext(ctx, query,
		status, counts["processed"], counts["successful"], counts["failed"],
		time.Now(), jobID,
	)

	return err
}

func (r *Repository) ListJobs(ctx context.Context, tenantID, status string, page, pageSize int) ([]*models.Job, int, error) {
	offset := (page - 1) * pageSize

	countQuery := `SELECT COUNT(*) FROM onboarding_jobs WHERE tenant_id = $1`
	query := `
		SELECT id, tenant_id, name, description, source_type, source_reference,
		       status, total_users, processed_users, successful_users, failed_users,
		       error_summary, created_by, started_at, completed_at, created_at, updated_at
		FROM onboarding_jobs
		WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	if status != "" {
		countQuery += " AND status = $2"
		query += " AND status = $2"
		args = append(args, status)
	}

	// Get total count
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", len(args)+limitArgIndex) +
		" OFFSET $" + fmt.Sprintf("%d", len(args)+offsetArgIndex)
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{}
		err := rows.Scan(
			&job.ID, &job.TenantID, &job.Name, &job.Description, &job.SourceType,
			&job.SourceReference, &job.Status, &job.TotalUsers, &job.ProcessedUsers,
			&job.SuccessfulUsers, &job.FailedUsers, &job.ErrorSummary, &job.CreatedBy,
			&job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}

	return jobs, total, nil
}

// ===== Task Operations =====

func (r *Repository) CreateTask(ctx context.Context, task *models.Task) error {
	task.ID = uuid.New().String()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	query := `
		INSERT INTO onboarding_tasks (
			id, job_id, tenant_id, email, first_name, last_name, role, student_id,
			department, course_codes, graduation_year, phone, preferred_language,
			custom_fields, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := r.db.ExecContext(ctx, query,
		task.ID, task.JobID, task.TenantID, task.Email, task.FirstName, task.LastName,
		task.Role, task.StudentID, task.Department, pq.Array(task.CourseCodes),
		task.GraduationYear, task.Phone, task.PreferredLang, task.CustomFields,
		task.Status, task.CreatedAt, task.UpdatedAt,
	)

	return err
}

func (r *Repository) GetTask(ctx context.Context, taskID, tenantID string) (*models.Task, error) {
	query := `
		SELECT id, job_id, tenant_id, email, first_name, last_name, role, student_id,
		       department, course_codes, graduation_year, phone, preferred_language,
		       custom_fields, status, user_id, retry_count, error_message, error_details,
		       created_at, updated_at, processed_at
		FROM onboarding_tasks
		WHERE id = $1 AND tenant_id = $2
	`

	task := &models.Task{}
	err := r.db.QueryRowContext(ctx, query, taskID, tenantID).Scan(
		&task.ID, &task.JobID, &task.TenantID, &task.Email, &task.FirstName,
		&task.LastName, &task.Role, &task.StudentID, &task.Department,
		pq.Array(&task.CourseCodes), &task.GraduationYear, &task.Phone,
		&task.PreferredLang, &task.CustomFields, &task.Status, &task.UserID,
		&task.RetryCount, &task.ErrorMessage, &task.ErrorDetails,
		&task.CreatedAt, &task.UpdatedAt, &task.ProcessedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}

	return task, err
}

func (r *Repository) UpdateTaskStatus(ctx context.Context, taskID, status, userID, errorMsg string) error {
	query := `
		UPDATE onboarding_tasks
		SET status = $1, user_id = $2, error_message = $3,
		    processed_at = $4, updated_at = $5
		WHERE id = $6
	`

	_, err := r.db.ExecContext(ctx, query,
		status, sql.NullString{String: userID, Valid: userID != ""},
		errorMsg, time.Now(), time.Now(), taskID,
	)

	return err
}

func (r *Repository) ListPendingTasks(ctx context.Context, jobID string, limit int) ([]*models.Task, error) {
	query := `
		SELECT id, job_id, tenant_id, email, first_name, last_name, role, student_id,
		       department, course_codes, graduation_year, phone, preferred_language,
		       custom_fields, status, user_id, retry_count, error_message, error_details,
		       created_at, updated_at, processed_at
		FROM onboarding_tasks
		WHERE job_id = $1 AND status = $2
		ORDER BY created_at ASC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, jobID, models.TaskStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		err := rows.Scan(
			&task.ID, &task.JobID, &task.TenantID, &task.Email, &task.FirstName,
			&task.LastName, &task.Role, &task.StudentID, &task.Department,
			pq.Array(&task.CourseCodes), &task.GraduationYear, &task.Phone,
			&task.PreferredLang, &task.CustomFields, &task.Status, &task.UserID,
			&task.RetryCount, &task.ErrorMessage, &task.ErrorDetails,
			&task.CreatedAt, &task.UpdatedAt, &task.ProcessedAt,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// ===== Audit Log Operations =====

func (r *Repository) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	log.ID = uuid.New().String()
	log.CreatedAt = time.Now()

	query := `
		INSERT INTO onboarding_audit_logs (
			id, tenant_id, job_id, task_id, event_type, event_data,
			performed_by, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		log.ID, log.TenantID, log.JobID, log.TaskID, log.EventType,
		log.EventData, log.PerformedBy, log.IPAddress, log.UserAgent,
		log.CreatedAt,
	)

	return err
}

// ===== Progress Operations =====

func (r *Repository) UpdateJobProgress(ctx context.Context, progress *models.JobProgress) error {
	metricsJSON, err := json.Marshal(progress.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	query := `
		INSERT INTO job_progress (
			job_id, current_stage, progress_percentage, estimated_completion_time,
			current_task_id, metrics, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (job_id) DO UPDATE SET
			current_stage = EXCLUDED.current_stage,
			progress_percentage = EXCLUDED.progress_percentage,
			estimated_completion_time = EXCLUDED.estimated_completion_time,
			current_task_id = EXCLUDED.current_task_id,
			metrics = EXCLUDED.metrics,
			updated_at = EXCLUDED.updated_at
	`

	_, execErr := r.db.ExecContext(ctx, query,
		progress.JobID, progress.CurrentStage, progress.ProgressPercentage,
		progress.EstimatedCompletionTime, progress.CurrentTaskID,
		string(metricsJSON), time.Now(),
	)

	return execErr
}

func (r *Repository) GetJobProgress(ctx context.Context, jobID string) (*models.JobProgress, error) {
	query := `
		SELECT job_id, current_stage, progress_percentage, estimated_completion_time,
		       current_task_id, metrics, updated_at
		FROM job_progress
		WHERE job_id = $1
	`

	progress := &models.JobProgress{}
	err := r.db.QueryRowContext(ctx, query, jobID).Scan(
		&progress.JobID, &progress.CurrentStage, &progress.ProgressPercentage,
		&progress.EstimatedCompletionTime, &progress.CurrentTaskID,
		&progress.Metrics, &progress.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("progress not found")
	}

	return progress, err
}

// ===== Integration Config Operations =====

func (r *Repository) CreateIntegrationConfig(ctx context.Context, config *models.IntegrationConfig) error {
	config.ID = uuid.New().String()
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	query := `
		INSERT INTO integration_configs (
			id, tenant_id, integration_type, name, config, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.TenantID, config.IntegrationType, config.Name,
		config.Config, config.IsActive, config.CreatedAt, config.UpdatedAt,
	)

	return err
}

func (r *Repository) GetIntegrationConfig(ctx context.Context, configID, tenantID string) (*models.IntegrationConfig, error) {
	query := `
		SELECT id, tenant_id, integration_type, name, config, is_active,
		       last_sync_at, created_at, updated_at
		FROM integration_configs
		WHERE id = $1 AND tenant_id = $2
	`

	config := &models.IntegrationConfig{}
	err := r.db.QueryRowContext(ctx, query, configID, tenantID).Scan(
		&config.ID, &config.TenantID, &config.IntegrationType, &config.Name,
		&config.Config, &config.IsActive, &config.LastSyncAt,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("integration config not found")
	}

	return config, err
}

// ===== Batch Operations =====

func (r *Repository) CreateTasksBatch(ctx context.Context, tasks []*models.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		// Rollback returns error if already committed, which is expected
		//nolint:errcheck // rollback error expected after commit
		tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO onboarding_tasks (
			id, job_id, tenant_id, email, first_name, last_name, role, student_id,
			department, course_codes, graduation_year, phone, preferred_language,
			custom_fields, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, task := range tasks {
		task.ID = uuid.New().String()
		task.CreatedAt = time.Now()
		task.UpdatedAt = time.Now()

		_, err = stmt.ExecContext(ctx,
			task.ID, task.JobID, task.TenantID, task.Email, task.FirstName, task.LastName,
			task.Role, task.StudentID, task.Department, pq.Array(task.CourseCodes),
			task.GraduationYear, task.Phone, task.PreferredLang, task.CustomFields,
			task.Status, task.CreatedAt, task.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
