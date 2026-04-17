package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"slate/services/assignment-grading-service/internal/models"

	"github.com/google/uuid"
)

// GradingRuleRepository defines the interface for grading rule data access
type GradingRuleRepository interface {
	Create(ctx context.Context, rule *models.GradingRule) error
	GetByID(ctx context.Context, id string) (*models.GradingRule, error)
	Update(ctx context.Context, rule *models.GradingRule) error
	Delete(ctx context.Context, id string) error
	ListByTenant(ctx context.Context, tenantID string) ([]*models.GradingRule, error)
	ListByCourse(ctx context.Context, tenantID, courseID string) ([]*models.GradingRule, error)
	GetForCourseOrTenant(ctx context.Context, tenantID, courseID string) ([]*models.GradingRule, error)
}

type gradingRuleRepository struct {
	db *sql.DB
}

// NewGradingRuleRepository creates a new grading rule repository
func NewGradingRuleRepository(db *sql.DB) GradingRuleRepository {
	return &gradingRuleRepository{db: db}
}

// Create creates a new grading rule
func (r *gradingRuleRepository) Create(ctx context.Context, rule *models.GradingRule) error {
	rule.ID = uuid.New().String()
	rule.CreatedAt = time.Now()

	gradeScaleJSON, err := rule.GradeScaleJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal grade scale: %w", err)
	}

	query := `
		INSERT INTO grading_rules (
			id, tenant_id, course_id, assignment_type, weight,
			late_penalty_per_day, max_late_penalty, grade_scale, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.ExecContext(ctx, query,
		rule.ID, rule.TenantID, nilIfEmpty(rule.CourseID),
		nilIfEmpty(rule.AssignmentType), rule.Weight,
		rule.LatePenaltyPerDay, rule.MaxLatePenalty,
		gradeScaleJSON, rule.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create grading rule: %w", err)
	}

	return nil
}

// GetByID retrieves a grading rule by ID
func (r *gradingRuleRepository) GetByID(ctx context.Context, id string) (*models.GradingRule, error) {
	query := `
		SELECT id, tenant_id, course_id, assignment_type, weight,
			   late_penalty_per_day, max_late_penalty, grade_scale, created_at
		FROM grading_rules
		WHERE id = $1
	`

	return r.scanRule(r.db.QueryRowContext(ctx, query, id))
}

// Update updates an existing grading rule
func (r *gradingRuleRepository) Update(ctx context.Context, rule *models.GradingRule) error {
	gradeScaleJSON, err := rule.GradeScaleJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal grade scale: %w", err)
	}

	query := `
		UPDATE grading_rules
		SET course_id = $2, assignment_type = $3, weight = $4,
			late_penalty_per_day = $5, max_late_penalty = $6, grade_scale = $7
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		rule.ID, nilIfEmpty(rule.CourseID), nilIfEmpty(rule.AssignmentType),
		rule.Weight, rule.LatePenaltyPerDay, rule.MaxLatePenalty, gradeScaleJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update grading rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("grading rule not found")
	}

	return nil
}

// Delete deletes a grading rule
func (r *gradingRuleRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM grading_rules WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete grading rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("grading rule not found")
	}

	return nil
}

// ListByTenant lists all grading rules for a tenant
func (r *gradingRuleRepository) ListByTenant(ctx context.Context, tenantID string) ([]*models.GradingRule, error) {
	query := `
		SELECT id, tenant_id, course_id, assignment_type, weight,
			   late_penalty_per_day, max_late_penalty, grade_scale, created_at
		FROM grading_rules
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	return r.scanRules(ctx, query, tenantID)
}

// ListByCourse lists grading rules for a specific course
func (r *gradingRuleRepository) ListByCourse(ctx context.Context, tenantID, courseID string) ([]*models.GradingRule, error) {
	query := `
		SELECT id, tenant_id, course_id, assignment_type, weight,
			   late_penalty_per_day, max_late_penalty, grade_scale, created_at
		FROM grading_rules
		WHERE tenant_id = $1 AND course_id = $2
		ORDER BY created_at DESC
	`

	return r.scanRules(ctx, query, tenantID, courseID)
}

// GetForCourseOrTenant returns course-specific rules, falling back to tenant-wide defaults
func (r *gradingRuleRepository) GetForCourseOrTenant(ctx context.Context, tenantID, courseID string) ([]*models.GradingRule, error) {
	query := `
		SELECT id, tenant_id, course_id, assignment_type, weight,
			   late_penalty_per_day, max_late_penalty, grade_scale, created_at
		FROM grading_rules
		WHERE tenant_id = $1 AND (course_id = $2 OR course_id IS NULL)
		ORDER BY course_id NULLS LAST, created_at DESC
	`

	return r.scanRules(ctx, query, tenantID, courseID)
}

// scanRule scans a single row
func (r *gradingRuleRepository) scanRule(row *sql.Row) (*models.GradingRule, error) {
	rule := &models.GradingRule{}
	var courseID, assignmentType sql.NullString
	var gradeScaleJSON []byte

	err := row.Scan(
		&rule.ID, &rule.TenantID, &courseID, &assignmentType, &rule.Weight,
		&rule.LatePenaltyPerDay, &rule.MaxLatePenalty, &gradeScaleJSON, &rule.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("grading rule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get grading rule: %w", err)
	}

	rule.CourseID = nullStringVal(courseID)
	rule.AssignmentType = nullStringVal(assignmentType)
	if err := rule.SetGradeScaleFromJSON(gradeScaleJSON); err != nil {
		return nil, fmt.Errorf("failed to parse grade scale: %w", err)
	}

	return rule, nil
}

// scanRules scans multiple rows
func (r *gradingRuleRepository) scanRules(ctx context.Context, query string, args ...interface{}) ([]*models.GradingRule, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list grading rules: %w", err)
	}
	defer rows.Close()

	var rules []*models.GradingRule
	for rows.Next() {
		rule := &models.GradingRule{}
		var courseID, assignmentType sql.NullString
		var gradeScaleJSON []byte

		err := rows.Scan(
			&rule.ID, &rule.TenantID, &courseID, &assignmentType, &rule.Weight,
			&rule.LatePenaltyPerDay, &rule.MaxLatePenalty, &gradeScaleJSON, &rule.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan grading rule: %w", err)
		}

		rule.CourseID = nullStringVal(courseID)
		rule.AssignmentType = nullStringVal(assignmentType)
		if err := rule.SetGradeScaleFromJSON(gradeScaleJSON); err != nil {
			return nil, fmt.Errorf("failed to parse grade scale: %w", err)
		}

		rules = append(rules, rule)
	}

	return rules, nil
}
