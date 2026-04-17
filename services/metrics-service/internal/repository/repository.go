package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"slate/services/metrics-service/internal/models"
)

// Repository provides database access for metrics data.
type Repository struct {
	db *sql.DB
}

// New creates a new Repository.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// InsertEvent logs an event to the event_log table.
func (r *Repository) InsertEvent(ctx context.Context, tenantID, eventType string, userID, courseID, assignmentID *string, metadata map[string]interface{}) error {
	metaJSON, _ := json.Marshal(metadata)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO event_log (tenant_id, event_type, user_id, course_id, assignment_id, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		tenantID, eventType,
		toNullString(userID), toNullString(courseID), toNullString(assignmentID),
		metaJSON,
	)
	return err
}

// UpsertStudentProgress inserts or updates student progress for a course.
func (r *Repository) UpsertStudentProgress(ctx context.Context, tenantID, studentID, courseID string, lessonsIncrement int, completionPct float64, durationMinutes int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO student_progress (tenant_id, student_id, course_id, lessons_completed, completion_pct, time_on_task_minutes, last_activity_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		 ON CONFLICT (tenant_id, student_id, course_id)
		 DO UPDATE SET
		   lessons_completed = student_progress.lessons_completed + $4,
		   completion_pct = GREATEST(student_progress.completion_pct, $5),
		   time_on_task_minutes = student_progress.time_on_task_minutes + $6,
		   last_activity_at = NOW(),
		   updated_at = NOW()`,
		tenantID, studentID, courseID, lessonsIncrement, completionPct, durationMinutes,
	)
	return err
}

// UpsertGradeStats inserts or updates grade statistics for a course/assignment.
func (r *Repository) UpsertGradeStats(ctx context.Context, tenantID, courseID string, assignmentID *string, stats *models.GradeStats) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO grade_stats (tenant_id, course_id, assignment_id, mean_score, median_score, p25, p75, std_dev, student_count, computed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		 ON CONFLICT DO NOTHING`,
		tenantID, courseID, toNullString(assignmentID),
		stats.MeanScore, stats.MedianScore, stats.P25, stats.P75, stats.StdDev, stats.StudentCount,
	)
	return err
}

// GetPlatformMetrics returns platform-wide statistics.
func (r *Repository) GetPlatformMetrics(ctx context.Context) (*models.PlatformMetrics, error) {
	m := &models.PlatformMetrics{}

	// DAU: distinct users in last 24h
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM event_log WHERE time > NOW() - INTERVAL '24 hours' AND user_id IS NOT NULL`,
	).Scan(&m.DAU)
	if err != nil {
		return nil, err
	}

	// MAU: distinct users in last 30d
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM event_log WHERE time > NOW() - INTERVAL '30 days' AND user_id IS NOT NULL`,
	).Scan(&m.MAU)
	if err != nil {
		return nil, err
	}

	// Active tenants: distinct tenants with events in last 24h
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT tenant_id) FROM event_log WHERE time > NOW() - INTERVAL '24 hours'`,
	).Scan(&m.ActiveTenants)
	if err != nil {
		return nil, err
	}

	// Total courses: distinct courses in student_progress
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT course_id) FROM student_progress`,
	).Scan(&m.TotalCourses)
	if err != nil {
		return nil, err
	}

	// Total students: distinct students in student_progress
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT student_id) FROM student_progress`,
	).Scan(&m.TotalStudents)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// GetTenantMetrics returns usage metrics for a specific tenant.
func (r *Repository) GetTenantMetrics(ctx context.Context, tenantID string) (*models.TenantMetrics, error) {
	m := &models.TenantMetrics{TenantID: tenantID}

	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM event_log WHERE tenant_id = $1 AND time > NOW() - INTERVAL '30 days'`,
		tenantID,
	).Scan(&m.EventCount)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM event_log WHERE tenant_id = $1 AND time > NOW() - INTERVAL '24 hours' AND user_id IS NOT NULL`,
		tenantID,
	).Scan(&m.ActiveUsers)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT student_id) FROM student_progress WHERE tenant_id = $1`,
		tenantID,
	).Scan(&m.TotalStudents)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT course_id) FROM student_progress WHERE tenant_id = $1`,
		tenantID,
	).Scan(&m.TotalCourses)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// GetCourseEngagement returns engagement metrics for a course.
func (r *Repository) GetCourseEngagement(ctx context.Context, courseID string) (*models.CourseEngagement, error) {
	m := &models.CourseEngagement{CourseID: courseID}

	err := r.db.QueryRowContext(ctx,
		`SELECT
			COUNT(*),
			COALESCE(AVG(completion_pct), 0),
			COALESCE(AVG(time_on_task_minutes), 0)
		 FROM student_progress WHERE course_id = $1`,
		courseID,
	).Scan(&m.TotalStudents, &m.AvgCompletionPct, &m.AvgTimeOnTask)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM event_log WHERE course_id = $1 AND event_type = 'submission.uploaded'`,
		courseID,
	).Scan(&m.TotalSubmissions)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM event_log WHERE course_id = $1 AND time > NOW() - INTERVAL '7 days' AND user_id IS NOT NULL`,
		courseID,
	).Scan(&m.ActiveLast7Days)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// GetStudentProgress returns progress for a specific student.
func (r *Repository) GetStudentProgress(ctx context.Context, tenantID, studentID string) ([]models.StudentProgress, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, student_id, course_id, completion_pct, time_on_task_minutes, lessons_completed, last_activity_at, updated_at
		 FROM student_progress WHERE tenant_id = $1 AND student_id = $2
		 ORDER BY updated_at DESC`,
		tenantID, studentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.StudentProgress
	for rows.Next() {
		var sp models.StudentProgress
		var lastActivity sql.NullTime
		if err := rows.Scan(&sp.ID, &sp.TenantID, &sp.StudentID, &sp.CourseID,
			&sp.CompletionPct, &sp.TimeOnTaskMinutes, &sp.LessonsCompleted,
			&lastActivity, &sp.UpdatedAt); err != nil {
			return nil, err
		}
		if lastActivity.Valid {
			sp.LastActivityAt = lastActivity.Time
		}
		results = append(results, sp)
	}
	return results, rows.Err()
}

// GetStudentTimeOnTask returns time-on-task data per course for a student.
func (r *Repository) GetStudentTimeOnTask(ctx context.Context, tenantID, studentID string) ([]models.TimeOnTask, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT course_id, time_on_task_minutes, lessons_completed
		 FROM student_progress WHERE tenant_id = $1 AND student_id = $2
		 ORDER BY time_on_task_minutes DESC`,
		tenantID, studentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.TimeOnTask
	for rows.Next() {
		var t models.TimeOnTask
		if err := rows.Scan(&t.CourseID, &t.TotalMinutes, &t.LessonsCompleted); err != nil {
			return nil, err
		}
		results = append(results, t)
	}
	return results, rows.Err()
}

// GetLatestGradeStats returns the latest grade stats for a course.
func (r *Repository) GetLatestGradeStats(ctx context.Context, tenantID, courseID string) (*models.GradeStats, error) {
	gs := &models.GradeStats{}
	var assignmentID sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, course_id, assignment_id, mean_score, median_score, p25, p75, std_dev, student_count, computed_at
		 FROM grade_stats WHERE tenant_id = $1 AND course_id = $2 AND assignment_id IS NULL
		 ORDER BY computed_at DESC LIMIT 1`,
		tenantID, courseID,
	).Scan(&gs.ID, &gs.TenantID, &gs.CourseID, &assignmentID,
		&gs.MeanScore, &gs.MedianScore, &gs.P25, &gs.P75, &gs.StdDev,
		&gs.StudentCount, &gs.ComputedAt)
	if err != nil {
		return nil, err
	}
	if assignmentID.Valid {
		gs.AssignmentID = assignmentID.String
	}
	return gs, nil
}

// GetScoresForAssignment returns all scores for an assignment from event_log.
func (r *Repository) GetScoresForAssignment(ctx context.Context, tenantID, courseID, assignmentID string) ([]float64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT ON (user_id)
			(metadata->>'score')::float8
		 FROM event_log
		 WHERE tenant_id = $1 AND course_id = $2 AND assignment_id = $3
		   AND event_type = 'submission.graded'
		   AND metadata->>'score' IS NOT NULL
		 ORDER BY user_id, time DESC`,
		tenantID, courseID, assignmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []float64
	for rows.Next() {
		var s float64
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	return scores, rows.Err()
}

// GetScoresForCourse returns all latest scores per student across assignments.
func (r *Repository) GetScoresForCourse(ctx context.Context, tenantID, courseID string) ([]float64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT ON (user_id)
			(metadata->>'score')::float8
		 FROM event_log
		 WHERE tenant_id = $1 AND course_id = $2
		   AND event_type = 'submission.graded'
		   AND metadata->>'score' IS NOT NULL
		 ORDER BY user_id, time DESC`,
		tenantID, courseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []float64
	for rows.Next() {
		var s float64
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	return scores, rows.Err()
}

// GetStudentScoreForCourse returns the latest score for a student in a course.
func (r *Repository) GetStudentScoreForCourse(ctx context.Context, tenantID, studentID, courseID string) (float64, error) {
	var score float64
	err := r.db.QueryRowContext(ctx,
		`SELECT (metadata->>'score')::float8
		 FROM event_log
		 WHERE tenant_id = $1 AND user_id = $2 AND course_id = $3
		   AND event_type = 'submission.graded'
		   AND metadata->>'score' IS NOT NULL
		 ORDER BY time DESC LIMIT 1`,
		tenantID, studentID, courseID,
	).Scan(&score)
	return score, err
}

// GetTenantIDForCourse attempts to find the tenant for a course from existing data.
func (r *Repository) GetTenantIDForCourse(ctx context.Context, courseID string) (string, error) {
	var tenantID string
	err := r.db.QueryRowContext(ctx,
		`SELECT tenant_id FROM student_progress WHERE course_id = $1 LIMIT 1`,
		courseID,
	).Scan(&tenantID)
	if err != nil {
		err = r.db.QueryRowContext(ctx,
			`SELECT tenant_id FROM event_log WHERE course_id = $1 ORDER BY time DESC LIMIT 1`,
			courseID,
		).Scan(&tenantID)
	}
	return tenantID, err
}

// GetTenantIDForStudent attempts to find the tenant for a student from existing data.
func (r *Repository) GetTenantIDForStudent(ctx context.Context, studentID string) (string, error) {
	var tenantID string
	err := r.db.QueryRowContext(ctx,
		`SELECT tenant_id FROM student_progress WHERE student_id = $1 LIMIT 1`,
		studentID,
	).Scan(&tenantID)
	return tenantID, err
}

func toNullString(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// InsertEventWithTime logs an event with a specific timestamp.
func (r *Repository) InsertEventWithTime(ctx context.Context, t time.Time, tenantID, eventType string, userID, courseID, assignmentID *string, metadata map[string]interface{}) error {
	metaJSON, _ := json.Marshal(metadata)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO event_log (time, tenant_id, event_type, user_id, course_id, assignment_id, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		t, tenantID, eventType,
		toNullString(userID), toNullString(courseID), toNullString(assignmentID),
		metaJSON,
	)
	return err
}
