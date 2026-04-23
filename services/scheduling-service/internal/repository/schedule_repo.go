// Package repository holds PostgreSQL access for scheduling-service.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrSlotTaken = errors.New("slot already booked")

type ScheduleRepository struct {
	db *sql.DB
}

func NewScheduleRepository(db *sql.DB) *ScheduleRepository {
	return &ScheduleRepository{db: db}
}

func (r *ScheduleRepository) Create(ctx context.Context, s Schedule) (*Schedule, error) {
	const q = `
		INSERT INTO oh_schedules
		  (instructor_id, day_of_week, start_time, end_time,
		   slot_duration_minutes, format, location, is_active)
		VALUES ($1, $2, $3::time, $4::time, $5, $6, $7, true)
		RETURNING id, created_at
	`
	row := r.db.QueryRowContext(ctx, q,
		s.InstructorID, s.DayOfWeek, s.StartTime, s.EndTime,
		s.SlotDurationMinutes, s.Format, nullableString(s.Location))
	if err := row.Scan(&s.ID, &s.CreatedAt); err != nil {
		return nil, fmt.Errorf("create schedule: %w", err)
	}
	s.IsActive = true
	return &s, nil
}

func (r *ScheduleRepository) ListForInstructor(ctx context.Context, instructorID string, activeOnly bool) ([]Schedule, error) {
	q := `SELECT id, instructor_id, day_of_week,
	             to_char(start_time,'HH24:MI'), to_char(end_time,'HH24:MI'),
	             slot_duration_minutes, format, location, is_active, created_at
	      FROM oh_schedules WHERE instructor_id = $1`
	if activeOnly {
		q += " AND is_active = true"
	}
	q += " ORDER BY day_of_week, start_time"
	rows, err := r.db.QueryContext(ctx, q, instructorID)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()
	var out []Schedule
	for rows.Next() {
		var s Schedule
		var loc sql.NullString
		if err := rows.Scan(&s.ID, &s.InstructorID, &s.DayOfWeek,
			&s.StartTime, &s.EndTime,
			&s.SlotDurationMinutes, &s.Format, &loc, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		if loc.Valid {
			v := loc.String
			s.Location = &v
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListBookedSlots returns confirmed bookings for an instructor in [dateFrom, dateTo].
// Dates inclusive. Only 'confirmed' bookings count as "taken".
func (r *ScheduleRepository) ListBookedSlots(ctx context.Context, instructorID string, dateFrom, dateTo time.Time) ([]Booking, error) {
	const q = `
		SELECT id, schedule_id, instructor_id, student_id,
		       to_char(slot_date,'YYYY-MM-DD'), to_char(slot_start_time,'HH24:MI'),
		       questions, status, created_at
		FROM oh_bookings
		WHERE instructor_id = $1
		  AND slot_date >= $2 AND slot_date <= $3
		  AND status = 'confirmed'
	`
	rows, err := r.db.QueryContext(ctx, q, instructorID, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()
	var out []Booking
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Book inserts a confirmed booking. Relies on the partial unique index on
// (instructor_id, slot_date, slot_start_time) WHERE status='confirmed' to reject
// double-booking atomically. Translates a pq unique-violation into ErrSlotTaken
// without introducing a lib/pq dependency in the repo surface.
func (r *ScheduleRepository) Book(ctx context.Context, b Booking) (*Booking, error) {
	const q = `
		INSERT INTO oh_bookings
		  (schedule_id, instructor_id, student_id, slot_date, slot_start_time, questions, status)
		VALUES ($1, $2, $3, $4::date, $5::time, $6, 'confirmed')
		RETURNING id, created_at
	`
	row := r.db.QueryRowContext(ctx, q,
		nullableString(b.ScheduleID), b.InstructorID, b.StudentID,
		b.SlotDate, b.SlotStartTime, nullableString(b.Questions))
	if err := row.Scan(&b.ID, &b.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSlotTaken
		}
		return nil, fmt.Errorf("book slot: %w", err)
	}
	b.Status = "confirmed"
	return &b, nil
}

// Cancel marks a booking cancelled; returns ErrNotFound if no confirmed row matches.
func (r *ScheduleRepository) Cancel(ctx context.Context, bookingID string) (*Booking, error) {
	const q = `
		UPDATE oh_bookings SET status = 'cancelled'
		WHERE id = $1 AND status = 'confirmed'
		RETURNING id, schedule_id, instructor_id, student_id,
		          to_char(slot_date,'YYYY-MM-DD'), to_char(slot_start_time,'HH24:MI'),
		          questions, status, created_at
	`
	row := r.db.QueryRowContext(ctx, q, bookingID)
	b, err := scanBooking(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("cancel booking: %w", err)
	}
	return &b, nil
}

// InstructorDay returns all confirmed bookings for an instructor on a single date,
// ordered by slot_start_time. Includes `questions` so instructors see context.
func (r *ScheduleRepository) InstructorDay(ctx context.Context, instructorID string, date time.Time) ([]Booking, error) {
	const q = `
		SELECT id, schedule_id, instructor_id, student_id,
		       to_char(slot_date,'YYYY-MM-DD'), to_char(slot_start_time,'HH24:MI'),
		       questions, status, created_at
		FROM oh_bookings
		WHERE instructor_id = $1 AND slot_date = $2 AND status = 'confirmed'
		ORDER BY slot_start_time
	`
	rows, err := r.db.QueryContext(ctx, q, instructorID, date)
	if err != nil {
		return nil, fmt.Errorf("instructor day: %w", err)
	}
	defer rows.Close()
	var out []Booking
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// rowScanner matches *sql.Rows and *sql.Row so scanBooking works for both.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanBooking(r rowScanner) (Booking, error) {
	var b Booking
	var schedID sql.NullString
	var questions sql.NullString
	if err := r.Scan(&b.ID, &schedID, &b.InstructorID, &b.StudentID,
		&b.SlotDate, &b.SlotStartTime, &questions, &b.Status, &b.CreatedAt); err != nil {
		return Booking{}, err
	}
	if schedID.Valid {
		v := schedID.String
		b.ScheduleID = &v
	}
	if questions.Valid {
		v := questions.String
		b.Questions = &v
	}
	return b, nil
}

func nullableString(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

// isUniqueViolation recognises Postgres SQLSTATE 23505 without pulling pq into the
// repo surface. Works for lib/pq errors whose Error() starts with "pq: duplicate".
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// PgError from pgx/pq both expose SQLSTATE via a "Code" field; string match
	// is good enough since tests use the same driver as prod (lib/pq).
	msg := err.Error()
	return contains(msg, "duplicate key value") || contains(msg, "SQLSTATE 23505")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	n := len(sub)
	for i := 0; i+n <= len(s); i++ {
		if s[i:i+n] == sub {
			return i
		}
	}
	return -1
}
