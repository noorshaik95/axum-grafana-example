package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"slate/services/scheduling-service/internal/cache"
	"slate/services/scheduling-service/internal/repository"
)

// Scheduler wires the repository + cache + tenant identity for use by the
// gRPC server. Kept on the service layer so we can test booking/cancellation
// orchestration independently of the wire protocol.
type Scheduler struct {
	repo       ScheduleStore
	cache      cache.Cache
	tenantSlug string
}

// ScheduleStore is the subset of the repository the service depends on.
// Interface widens to allow injecting in-memory test doubles without lib/pq.
type ScheduleStore interface {
	Create(ctx context.Context, s repository.Schedule) (*repository.Schedule, error)
	ListForInstructor(ctx context.Context, instructorID string, activeOnly bool) ([]repository.Schedule, error)
	ListBookedSlots(ctx context.Context, instructorID string, dateFrom, dateTo time.Time) ([]repository.Booking, error)
	Book(ctx context.Context, b repository.Booking) (*repository.Booking, error)
	Cancel(ctx context.Context, bookingID string) (*repository.Booking, error)
	InstructorDay(ctx context.Context, instructorID string, date time.Time) ([]repository.Booking, error)
}

func NewScheduler(repo ScheduleStore, c cache.Cache, tenantSlug string) *Scheduler {
	if c == nil {
		c = cache.NoopCache{}
	}
	return &Scheduler{repo: repo, cache: c, tenantSlug: tenantSlug}
}

// AvailableSlots returns slots for [dateFrom, dateTo] with a Redis read-through.
// `fromCache` is true when the response was served without hitting Postgres.
func (s *Scheduler) AvailableSlots(ctx context.Context, instructorID, dateFrom, dateTo string) ([]cache.Slot, bool, error) {
	if instructorID == "" {
		return nil, false, errors.New("instructor_id required")
	}
	from, err := ParseDate(dateFrom)
	if err != nil {
		return nil, false, fmt.Errorf("date_from: %w", err)
	}
	to, err := ParseDate(dateTo)
	if err != nil {
		return nil, false, fmt.Errorf("date_to: %w", err)
	}

	if slots, hit, err := s.cache.Get(ctx, s.tenantSlug, instructorID, dateFrom, dateTo); err == nil && hit {
		return slots, true, nil
	}

	schedules, err := s.repo.ListForInstructor(ctx, instructorID, true)
	if err != nil {
		return nil, false, err
	}
	booked, err := s.repo.ListBookedSlots(ctx, instructorID, from, to)
	if err != nil {
		return nil, false, err
	}
	slots, err := GenerateSlots(schedules, booked, from, to)
	if err != nil {
		return nil, false, err
	}
	_ = s.cache.Set(ctx, s.tenantSlug, instructorID, dateFrom, dateTo, slots)
	return slots, false, nil
}

// Book writes a confirmed booking, storing `preContext` as the questions field
// so GetInstructorDay pre-populates it for the instructor (W6.4).
// Invalidates the cached slot list on success so the next ListAvailableSlots
// re-reads Postgres.
func (s *Scheduler) Book(ctx context.Context, scheduleID, instructorID, studentID, slotDate, slotStart, preContext string) (*repository.Booking, error) {
	if instructorID == "" || studentID == "" || slotDate == "" || slotStart == "" {
		return nil, errors.New("instructor_id, student_id, slot_date, slot_start_time required")
	}
	if _, err := ParseDate(slotDate); err != nil {
		return nil, fmt.Errorf("slot_date: %w", err)
	}
	if _, err := parseHM(slotStart); err != nil {
		return nil, fmt.Errorf("slot_start_time: %w", err)
	}
	var schedIDPtr *string
	if scheduleID != "" {
		schedIDPtr = &scheduleID
	}
	var qPtr *string
	if preContext != "" {
		qPtr = &preContext
	}
	b, err := s.repo.Book(ctx, repository.Booking{
		ScheduleID:    schedIDPtr,
		InstructorID:  instructorID,
		StudentID:     studentID,
		SlotDate:      slotDate,
		SlotStartTime: slotStart,
		Questions:     qPtr,
	})
	if err != nil {
		return nil, err
	}
	_ = s.cache.InvalidateInstructor(ctx, s.tenantSlug, instructorID)
	return b, nil
}

// Cancel transitions a confirmed booking to "cancelled". Invalidates the slot
// cache so the freed slot reappears in ListAvailableSlots.
func (s *Scheduler) Cancel(ctx context.Context, bookingID string) (*repository.Booking, error) {
	if bookingID == "" {
		return nil, errors.New("booking_id required")
	}
	b, err := s.repo.Cancel(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	_ = s.cache.InvalidateInstructor(ctx, s.tenantSlug, b.InstructorID)
	return b, nil
}

func (s *Scheduler) CreateSchedule(ctx context.Context, sched repository.Schedule) (*repository.Schedule, error) {
	if sched.InstructorID == "" {
		return nil, errors.New("instructor_id required")
	}
	if sched.DayOfWeek < 0 || sched.DayOfWeek > 6 {
		return nil, errors.New("day_of_week must be 0..6")
	}
	if sched.Format == "" {
		sched.Format = "online"
	}
	if sched.SlotDurationMinutes == 0 {
		sched.SlotDurationMinutes = 15
	}
	out, err := s.repo.Create(ctx, sched)
	if err != nil {
		return nil, err
	}
	_ = s.cache.InvalidateInstructor(ctx, s.tenantSlug, sched.InstructorID)
	return out, nil
}

func (s *Scheduler) ListSchedules(ctx context.Context, instructorID string, activeOnly bool) ([]repository.Schedule, error) {
	return s.repo.ListForInstructor(ctx, instructorID, activeOnly)
}

func (s *Scheduler) InstructorDay(ctx context.Context, instructorID, slotDate string) ([]repository.Booking, error) {
	d, err := ParseDate(slotDate)
	if err != nil {
		return nil, fmt.Errorf("slot_date: %w", err)
	}
	return s.repo.InstructorDay(ctx, instructorID, d)
}
