package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

// --- Extensions for W12 (roster health, export, platform stats) ---

// RosterSignalRow is the raw per-student input used to classify roster risk.
type RosterSignalRow struct {
	UserID            string
	DisplayName       string
	MissedAssignments int
	DaysSinceActive   int
	GradeTrend        []float64
}

// GetRosterSignals returns the per-student signals needed to compute risk for
// every student enrolled in a course. Fields that the schema does not yet
// track (display name) fall back to the student ID.
func (r *Repository) GetRosterSignals(ctx context.Context, tenantID, courseID string) ([]RosterSignalRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT
			sp.student_id::text,
			sp.student_id::text AS display_name,
			(EXTRACT(EPOCH FROM (NOW() - COALESCE(sp.last_activity_at, NOW()))) / 86400)::int AS days_since_active
		 FROM student_progress sp
		 WHERE sp.tenant_id = $1 AND sp.course_id = $2`,
		tenantID, courseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RosterSignalRow
	for rows.Next() {
		var row RosterSignalRow
		if err := rows.Scan(&row.UserID, &row.DisplayName, &row.DaysSinceActive); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		missed, trend, err := r.studentRiskSignals(ctx, tenantID, courseID, out[i].UserID)
		if err != nil {
			return nil, err
		}
		out[i].MissedAssignments = missed
		out[i].GradeTrend = trend
	}
	return out, nil
}

func (r *Repository) studentRiskSignals(ctx context.Context, tenantID, courseID, studentID string) (int, []float64, error) {
	var missed int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM event_log
		 WHERE tenant_id = $1 AND course_id = $2 AND user_id = $3
		   AND event_type = 'assignment.missed'`,
		tenantID, courseID, studentID,
	).Scan(&missed)
	if err != nil {
		return 0, nil, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT (metadata->>'score')::float8
		 FROM event_log
		 WHERE tenant_id = $1 AND course_id = $2 AND user_id = $3
		   AND event_type = 'submission.graded'
		   AND metadata->>'score' IS NOT NULL
		 ORDER BY time DESC
		 LIMIT 5`,
		tenantID, courseID, studentID,
	)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var desc []float64
	for rows.Next() {
		var s float64
		if err := rows.Scan(&s); err != nil {
			return 0, nil, err
		}
		desc = append(desc, s)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	trend := make([]float64, len(desc))
	for i, v := range desc {
		trend[len(desc)-1-i] = v
	}
	return missed, trend, nil
}

// TenantMAURow captures monthly-active-user counts per tenant.
type TenantMAURow struct {
	TenantID string
	MAU      int
}

// GetTenantMAU returns monthly-active-user counts grouped by tenant.
func (r *Repository) GetTenantMAU(ctx context.Context) ([]TenantMAURow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT tenant_id::text, COUNT(DISTINCT user_id) AS mau
		 FROM event_log
		 WHERE time > NOW() - INTERVAL '30 days' AND user_id IS NOT NULL
		 GROUP BY tenant_id
		 ORDER BY mau DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TenantMAURow
	for rows.Next() {
		var t TenantMAURow
		if err := rows.Scan(&t.TenantID, &t.MAU); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetSignupsLast30d counts user-creation events in the last 30 days.
func (r *Repository) GetSignupsLast30d(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM event_log
		 WHERE event_type = 'user.created' AND time > NOW() - INTERVAL '30 days'`,
	).Scan(&n)
	return n, err
}

// GetUptimePct derives an uptime percent from `health.ok` / `health.fail`
// events in the last 24 hours. Returns 100.0 when there is no data — an
// empty window should not be reported as 0% uptime.
func (r *Repository) GetUptimePct(ctx context.Context) (float64, error) {
	var total, failures int
	err := r.db.QueryRowContext(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE event_type IN ('health.ok','health.fail')),
			COUNT(*) FILTER (WHERE event_type = 'health.fail')
		 FROM event_log
		 WHERE time > NOW() - INTERVAL '24 hours'`,
	).Scan(&total, &failures)
	if err != nil {
		return 0, err
	}
	if total == 0 {
		return 100.0, nil
	}
	return float64(total-failures) / float64(total) * 100.0, nil
}

// GradebookRows returns CSV-ready rows for a course gradebook export.
func (r *Repository) GradebookRows(ctx context.Context, tenantID, courseID string) ([][]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT ON (user_id)
			user_id::text,
			COALESCE((metadata->>'score'), ''),
			COALESCE(assignment_id::text, ''),
			time::text
		 FROM event_log
		 WHERE tenant_id = $1 AND course_id = $2
		   AND event_type = 'submission.graded'
		 ORDER BY user_id, time DESC`,
		tenantID, courseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := [][]string{{"student_id", "score", "assignment_id", "graded_at"}}
	for rows.Next() {
		var studentID, score, assignmentID, gradedAt string
		if err := rows.Scan(&studentID, &score, &assignmentID, &gradedAt); err != nil {
			return nil, err
		}
		out = append(out, []string{studentID, score, assignmentID, gradedAt})
	}
	return out, rows.Err()
}

// RosterRows returns CSV-ready rows for a roster export.
func (r *Repository) RosterRows(ctx context.Context, tenantID, courseID string) ([][]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT
			student_id::text,
			COALESCE(completion_pct::text, '0'),
			COALESCE(time_on_task_minutes::text, '0'),
			COALESCE(last_activity_at::text, '')
		 FROM student_progress
		 WHERE tenant_id = $1 AND course_id = $2
		 ORDER BY student_id`,
		tenantID, courseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := [][]string{{"student_id", "completion_pct", "time_on_task_minutes", "last_activity_at"}}
	for rows.Next() {
		var studentID, completion, tot, last string
		if err := rows.Scan(&studentID, &completion, &tot, &last); err != nil {
			return nil, err
		}
		out = append(out, []string{studentID, completion, tot, last})
	}
	return out, rows.Err()
}

// PlatformRows returns CSV-ready rows for a platform export.
func (r *Repository) PlatformRows(ctx context.Context) ([][]string, error) {
	m, err := r.GetPlatformMetrics(ctx)
	if err != nil {
		return nil, err
	}
	return [][]string{
		{"metric", "value"},
		{"dau", fmt.Sprintf("%d", m.DAU)},
		{"mau", fmt.Sprintf("%d", m.MAU)},
		{"active_tenants", fmt.Sprintf("%d", m.ActiveTenants)},
		{"total_courses", fmt.Sprintf("%d", m.TotalCourses)},
		{"total_students", fmt.Sprintf("%d", m.TotalStudents)},
	}, nil
}
