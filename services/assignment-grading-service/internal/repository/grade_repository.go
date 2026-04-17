package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"slate/services/assignment-grading-service/internal/models"

	"github.com/google/uuid"
)

// GradeRepository defines the interface for grade data access
type GradeRepository interface {
	Create(ctx context.Context, grade *models.Grade) error
	GetByID(ctx context.Context, id string) (*models.Grade, error)
	GetBySubmission(ctx context.Context, submissionID string) (*models.Grade, error)
	Update(ctx context.Context, grade *models.Grade) error
	ListByStudent(ctx context.Context, studentID, courseID string) ([]*models.Grade, error)
	ListByStudentTenant(ctx context.Context, tenantID, studentID string) ([]*models.Grade, error)
	ListByCourse(ctx context.Context, courseID string) ([]*models.Grade, error)
	ListByAssignment(ctx context.Context, assignmentID string) ([]*models.Grade, error)
	GetStatistics(ctx context.Context, assignmentID string) (*GradeStatistics, error)
	GetScoresForAssignment(ctx context.Context, assignmentID string) ([]GradeScore, error)
	UpdatePercentile(ctx context.Context, gradeID string, percentile float64) error
	CreateEmpty(ctx context.Context, tenantID, assignmentID, studentID, courseID string) error
}

type GradeScore struct {
	GradeID      string
	SubmissionID string
	Score        float64
}

type GradeStatistics struct {
	TotalSubmissions int
	GradedCount      int
	Mean             float64
	Median           float64
	StdDeviation     float64
	MinScore         float64
	MaxScore         float64
}

type gradeRepository struct {
	db *sql.DB
}

// NewGradeRepository creates a new grade repository
func NewGradeRepository(db *sql.DB) GradeRepository {
	return &gradeRepository{db: db}
}

// Create creates a new grade
func (r *gradeRepository) Create(ctx context.Context, grade *models.Grade) error {
	grade.ID = uuid.New().String()
	now := time.Now()
	grade.CreatedAt = now
	grade.UpdatedAt = now
	grade.GradedAt = &now

	rubricScoresJSON, err := grade.RubricScoresJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal rubric scores: %w", err)
	}

	query := `
		INSERT INTO grades (
			id, tenant_id, submission_id, student_id, assignment_id, course_id,
			score, max_score, adjusted_score, percentage, letter_grade,
			rubric_scores, feedback, status, graded_at, graded_by,
			override_justification, percentile, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	_, err = r.db.ExecContext(ctx, query,
		grade.ID, nilIfEmpty(grade.TenantID), grade.SubmissionID, grade.StudentID,
		grade.AssignmentID, nilIfEmpty(grade.CourseID),
		grade.Score, grade.MaxScore, grade.AdjustedScore, grade.Percentage,
		nilIfEmpty(grade.LetterGrade), rubricScoresJSON,
		grade.Feedback, grade.Status, grade.GradedAt, grade.GradedBy,
		nilIfEmpty(grade.OverrideJustification), grade.Percentile,
		grade.CreatedAt, grade.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create grade: %w", err)
	}

	return nil
}

// GetByID retrieves a grade by ID
func (r *gradeRepository) GetByID(ctx context.Context, id string) (*models.Grade, error) {
	query := `
		SELECT id, tenant_id, submission_id, student_id, assignment_id, course_id,
			   score, COALESCE(max_score, 0), adjusted_score, COALESCE(percentage, 0),
			   letter_grade, rubric_scores, feedback, status, graded_at, published_at,
			   graded_by, override_justification, COALESCE(percentile, 0),
			   created_at, updated_at
		FROM grades
		WHERE id = $1
	`

	return r.scanGrade(r.db.QueryRowContext(ctx, query, id))
}

// GetBySubmission retrieves a grade by submission ID
func (r *gradeRepository) GetBySubmission(ctx context.Context, submissionID string) (*models.Grade, error) {
	query := `
		SELECT id, tenant_id, submission_id, student_id, assignment_id, course_id,
			   score, COALESCE(max_score, 0), adjusted_score, COALESCE(percentage, 0),
			   letter_grade, rubric_scores, feedback, status, graded_at, published_at,
			   graded_by, override_justification, COALESCE(percentile, 0),
			   created_at, updated_at
		FROM grades
		WHERE submission_id = $1
	`

	grade, err := r.scanGrade(r.db.QueryRowContext(ctx, query, submissionID))
	if err != nil && err.Error() == "grade not found" {
		return nil, nil
	}
	return grade, err
}

// Update updates an existing grade
func (r *gradeRepository) Update(ctx context.Context, grade *models.Grade) error {
	grade.UpdatedAt = time.Now()

	rubricScoresJSON, err := grade.RubricScoresJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal rubric scores: %w", err)
	}

	query := `
		UPDATE grades
		SET score = $2, adjusted_score = $3, feedback = $4, status = $5,
			published_at = $6, updated_at = $7, percentage = $8,
			letter_grade = $9, rubric_scores = $10, percentile = $11,
			override_justification = $12, max_score = $13
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		grade.ID, grade.Score, grade.AdjustedScore, grade.Feedback,
		grade.Status, grade.PublishedAt, grade.UpdatedAt, grade.Percentage,
		nilIfEmpty(grade.LetterGrade), rubricScoresJSON, grade.Percentile,
		nilIfEmpty(grade.OverrideJustification), grade.MaxScore,
	)

	if err != nil {
		return fmt.Errorf("failed to update grade: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("grade not found")
	}

	return nil
}

// ListByStudent lists grades for a student in a course
func (r *gradeRepository) ListByStudent(ctx context.Context, studentID, courseID string) ([]*models.Grade, error) {
	query := `
		SELECT g.id, g.tenant_id, g.submission_id, g.student_id, g.assignment_id, g.course_id,
			   g.score, COALESCE(g.max_score, 0), g.adjusted_score, COALESCE(g.percentage, 0),
			   g.letter_grade, g.rubric_scores, g.feedback, g.status, g.graded_at, g.published_at,
			   g.graded_by, g.override_justification, COALESCE(g.percentile, 0),
			   g.created_at, g.updated_at
		FROM grades g
		JOIN assignments a ON g.assignment_id = a.id
		WHERE g.student_id = $1 AND a.course_id = $2 AND g.status = 'published'
		ORDER BY a.due_date DESC
	`

	return r.scanGrades(ctx, query, studentID, courseID)
}

// ListByStudentTenant lists all grades for a student in a tenant
func (r *gradeRepository) ListByStudentTenant(ctx context.Context, tenantID, studentID string) ([]*models.Grade, error) {
	query := `
		SELECT g.id, g.tenant_id, g.submission_id, g.student_id, g.assignment_id, g.course_id,
			   g.score, COALESCE(g.max_score, 0), g.adjusted_score, COALESCE(g.percentage, 0),
			   g.letter_grade, g.rubric_scores, g.feedback, g.status, g.graded_at, g.published_at,
			   g.graded_by, g.override_justification, COALESCE(g.percentile, 0),
			   g.created_at, g.updated_at
		FROM grades g
		WHERE g.tenant_id = $1 AND g.student_id = $2 AND g.status = 'published'
		ORDER BY g.graded_at DESC
	`

	return r.scanGrades(ctx, query, tenantID, studentID)
}

// ListByCourse lists all grades for a course
func (r *gradeRepository) ListByCourse(ctx context.Context, courseID string) ([]*models.Grade, error) {
	query := `
		SELECT g.id, g.tenant_id, g.submission_id, g.student_id, g.assignment_id, g.course_id,
			   g.score, COALESCE(g.max_score, 0), g.adjusted_score, COALESCE(g.percentage, 0),
			   g.letter_grade, g.rubric_scores, g.feedback, g.status, g.graded_at, g.published_at,
			   g.graded_by, g.override_justification, COALESCE(g.percentile, 0),
			   g.created_at, g.updated_at
		FROM grades g
		JOIN assignments a ON g.assignment_id = a.id
		WHERE a.course_id = $1 AND g.status = 'published'
		ORDER BY g.student_id, a.due_date
	`

	return r.scanGrades(ctx, query, courseID)
}

// ListByAssignment lists all grades for an assignment
func (r *gradeRepository) ListByAssignment(ctx context.Context, assignmentID string) ([]*models.Grade, error) {
	query := `
		SELECT g.id, g.tenant_id, g.submission_id, g.student_id, g.assignment_id, g.course_id,
			   g.score, COALESCE(g.max_score, 0), g.adjusted_score, COALESCE(g.percentage, 0),
			   g.letter_grade, g.rubric_scores, g.feedback, g.status, g.graded_at, g.published_at,
			   g.graded_by, g.override_justification, COALESCE(g.percentile, 0),
			   g.created_at, g.updated_at
		FROM grades g
		WHERE g.assignment_id = $1
		ORDER BY g.score DESC
	`

	return r.scanGrades(ctx, query, assignmentID)
}

// GetScoresForAssignment returns all scores for percentile calculation
func (r *gradeRepository) GetScoresForAssignment(ctx context.Context, assignmentID string) ([]GradeScore, error) {
	query := `
		SELECT id, submission_id, adjusted_score
		FROM grades
		WHERE assignment_id = $1 AND score > 0
		ORDER BY adjusted_score
	`

	rows, err := r.db.QueryContext(ctx, query, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scores: %w", err)
	}
	defer rows.Close()

	var scores []GradeScore
	for rows.Next() {
		var gs GradeScore
		if err := rows.Scan(&gs.GradeID, &gs.SubmissionID, &gs.Score); err != nil {
			return nil, fmt.Errorf("failed to scan score: %w", err)
		}
		scores = append(scores, gs)
	}

	return scores, nil
}

// UpdatePercentile updates just the percentile field
func (r *gradeRepository) UpdatePercentile(ctx context.Context, gradeID string, percentile float64) error {
	query := `UPDATE grades SET percentile = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, gradeID, percentile)
	if err != nil {
		return fmt.Errorf("failed to update percentile: %w", err)
	}
	return nil
}

// CreateEmpty creates an empty grade record (used when student enrolls in course)
func (r *gradeRepository) CreateEmpty(ctx context.Context, tenantID, assignmentID, studentID, courseID string) error {
	query := `
		INSERT INTO grades (id, tenant_id, assignment_id, student_id, course_id, submission_id,
			score, adjusted_score, status, graded_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, '', 0, 0, 'draft', 'system', NOW(), NOW())
		ON CONFLICT DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query,
		uuid.New().String(), nilIfEmpty(tenantID), assignmentID, studentID, nilIfEmpty(courseID),
	)
	if err != nil {
		return fmt.Errorf("failed to create empty grade: %w", err)
	}
	return nil
}

// GetStatistics calculates statistics for an assignment
func (r *gradeRepository) GetStatistics(ctx context.Context, assignmentID string) (*GradeStatistics, error) {
	var totalSubmissions int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM submissions WHERE assignment_id = $1`,
		assignmentID,
	).Scan(&totalSubmissions)
	if err != nil {
		return nil, fmt.Errorf("failed to count submissions: %w", err)
	}

	query := `
		SELECT COUNT(*), COALESCE(MIN(adjusted_score), 0), COALESCE(MAX(adjusted_score), 0), COALESCE(AVG(adjusted_score), 0)
		FROM grades
		WHERE assignment_id = $1 AND status = 'published'
	`

	var gradedCount int
	var minScore, maxScore, mean float64
	err = r.db.QueryRowContext(ctx, query, assignmentID).Scan(&gradedCount, &minScore, &maxScore, &mean)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic stats: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT adjusted_score FROM grades WHERE assignment_id = $1 AND status = 'published' ORDER BY adjusted_score`,
		assignmentID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get scores: %w", err)
	}
	defer rows.Close()

	var scores []float64
	for rows.Next() {
		var score float64
		if err := rows.Scan(&score); err != nil {
			return nil, fmt.Errorf("failed to scan score: %w", err)
		}
		scores = append(scores, score)
	}

	median := 0.0
	if len(scores) > 0 {
		mid := len(scores) / 2
		if len(scores)%2 == 0 {
			median = (scores[mid-1] + scores[mid]) / 2
		} else {
			median = scores[mid]
		}
	}

	stdDev := 0.0
	if len(scores) > 0 {
		variance := 0.0
		for _, score := range scores {
			variance += math.Pow(score-mean, 2)
		}
		variance /= float64(len(scores))
		stdDev = math.Sqrt(variance)
	}

	return &GradeStatistics{
		TotalSubmissions: totalSubmissions,
		GradedCount:      gradedCount,
		Mean:             mean,
		Median:           median,
		StdDeviation:     stdDev,
		MinScore:         minScore,
		MaxScore:         maxScore,
	}, nil
}

// scanGrade scans a single row into a Grade
func (r *gradeRepository) scanGrade(row *sql.Row) (*models.Grade, error) {
	grade := &models.Grade{}
	var tenantID, courseID, letterGrade, overrideJustification sql.NullString
	var rubricScoresJSON []byte

	err := row.Scan(
		&grade.ID, &tenantID, &grade.SubmissionID, &grade.StudentID,
		&grade.AssignmentID, &courseID,
		&grade.Score, &grade.MaxScore, &grade.AdjustedScore, &grade.Percentage,
		&letterGrade, &rubricScoresJSON, &grade.Feedback, &grade.Status,
		&grade.GradedAt, &grade.PublishedAt, &grade.GradedBy,
		&overrideJustification, &grade.Percentile,
		&grade.CreatedAt, &grade.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("grade not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get grade: %w", err)
	}

	grade.TenantID = nullStringVal(tenantID)
	grade.CourseID = nullStringVal(courseID)
	grade.LetterGrade = nullStringVal(letterGrade)
	grade.OverrideJustification = nullStringVal(overrideJustification)
	if rubricScoresJSON != nil {
		if err := grade.SetRubricScoresFromJSON(rubricScoresJSON); err != nil {
			return nil, fmt.Errorf("failed to parse rubric scores: %w", err)
		}
	}

	return grade, nil
}

// scanGrades scans multiple rows
func (r *gradeRepository) scanGrades(ctx context.Context, query string, args ...interface{}) ([]*models.Grade, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list grades: %w", err)
	}
	defer rows.Close()

	var grades []*models.Grade
	for rows.Next() {
		grade := &models.Grade{}
		var tenantID, courseID, letterGrade, overrideJustification sql.NullString
		var rubricScoresJSON []byte

		err := rows.Scan(
			&grade.ID, &tenantID, &grade.SubmissionID, &grade.StudentID,
			&grade.AssignmentID, &courseID,
			&grade.Score, &grade.MaxScore, &grade.AdjustedScore, &grade.Percentage,
			&letterGrade, &rubricScoresJSON, &grade.Feedback, &grade.Status,
			&grade.GradedAt, &grade.PublishedAt, &grade.GradedBy,
			&overrideJustification, &grade.Percentile,
			&grade.CreatedAt, &grade.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan grade: %w", err)
		}

		grade.TenantID = nullStringVal(tenantID)
		grade.CourseID = nullStringVal(courseID)
		grade.LetterGrade = nullStringVal(letterGrade)
		grade.OverrideJustification = nullStringVal(overrideJustification)
		if rubricScoresJSON != nil {
			if err := grade.SetRubricScoresFromJSON(rubricScoresJSON); err != nil {
				return nil, fmt.Errorf("failed to parse rubric scores: %w", err)
			}
		}

		grades = append(grades, grade)
	}

	return grades, nil
}
