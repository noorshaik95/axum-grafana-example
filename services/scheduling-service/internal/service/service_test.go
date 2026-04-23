package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"slate/services/scheduling-service/internal/cache"
	"slate/services/scheduling-service/internal/repository"
)

// fakeStore is an in-memory ScheduleStore used by service-layer tests to avoid
// standing up Postgres.
type fakeStore struct {
	schedules   []repository.Schedule
	bookings    []repository.Booking
	nextID      int
	bookCalls   int
	listCalls   int
	bookedCalls int
}

func (f *fakeStore) Create(_ context.Context, s repository.Schedule) (*repository.Schedule, error) {
	f.nextID++
	s.ID = time.Now().Format("20060102150405") + "-" + itoa(f.nextID)
	s.IsActive = true
	s.CreatedAt = time.Now().UTC()
	f.schedules = append(f.schedules, s)
	return &s, nil
}

func (f *fakeStore) ListForInstructor(_ context.Context, instructorID string, activeOnly bool) ([]repository.Schedule, error) {
	f.listCalls++
	var out []repository.Schedule
	for _, s := range f.schedules {
		if s.InstructorID != instructorID {
			continue
		}
		if activeOnly && !s.IsActive {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

func (f *fakeStore) ListBookedSlots(_ context.Context, instructorID string, from, to time.Time) ([]repository.Booking, error) {
	f.bookedCalls++
	var out []repository.Booking
	for _, b := range f.bookings {
		if b.InstructorID != instructorID || b.Status != "confirmed" {
			continue
		}
		d, err := time.Parse("2006-01-02", b.SlotDate)
		if err != nil {
			continue
		}
		if d.Before(from) || d.After(to) {
			continue
		}
		out = append(out, b)
	}
	return out, nil
}

func (f *fakeStore) Book(_ context.Context, b repository.Booking) (*repository.Booking, error) {
	f.bookCalls++
	for _, existing := range f.bookings {
		if existing.Status != "confirmed" {
			continue
		}
		if existing.InstructorID == b.InstructorID && existing.SlotDate == b.SlotDate && existing.SlotStartTime == b.SlotStartTime {
			return nil, repository.ErrSlotTaken
		}
	}
	f.nextID++
	b.ID = "bk-" + itoa(f.nextID)
	b.Status = "confirmed"
	b.CreatedAt = time.Now().UTC()
	f.bookings = append(f.bookings, b)
	return &b, nil
}

func (f *fakeStore) Cancel(_ context.Context, bookingID string) (*repository.Booking, error) {
	for i := range f.bookings {
		if f.bookings[i].ID == bookingID && f.bookings[i].Status == "confirmed" {
			f.bookings[i].Status = "cancelled"
			b := f.bookings[i]
			return &b, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeStore) InstructorDay(_ context.Context, instructorID string, date time.Time) ([]repository.Booking, error) {
	wantDate := date.Format("2006-01-02")
	var out []repository.Booking
	for _, b := range f.bookings {
		if b.InstructorID == instructorID && b.SlotDate == wantDate && b.Status == "confirmed" {
			out = append(out, b)
		}
	}
	return out, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func newTestCache(t *testing.T) (cache.Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return cache.NewRedisCache(rc, 5*time.Minute), mr
}

func seedMondaySchedule(t *testing.T, f *fakeStore) {
	t.Helper()
	_, err := f.Create(context.Background(), repository.Schedule{
		InstructorID:        "inst-1",
		DayOfWeek:           1, // Monday
		StartTime:           "09:00",
		EndTime:             "10:00",
		SlotDurationMinutes: 15,
		Format:              "online",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

// Second identical call for the same (instructor, window) must hit the Redis
// cache and skip both schedules + bookings queries.
func TestAvailableSlots_CachesSecondCall(t *testing.T) {
	ctx := context.Background()
	f := &fakeStore{}
	seedMondaySchedule(t, f)
	c, _ := newTestCache(t)

	s := NewScheduler(f, c, "acme")

	_, hit, err := s.AvailableSlots(ctx, "inst-1", "2026-05-04", "2026-05-04")
	if err != nil || hit {
		t.Fatalf("first call: err=%v hit=%v (want miss)", err, hit)
	}
	firstList := f.listCalls

	slots, hit, err := s.AvailableSlots(ctx, "inst-1", "2026-05-04", "2026-05-04")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if !hit {
		t.Fatal("second call must hit cache")
	}
	if f.listCalls != firstList {
		t.Fatalf("cache hit still queried ListForInstructor: before=%d after=%d", firstList, f.listCalls)
	}
	if len(slots) != 4 {
		t.Fatalf("want 4 slots, got %d", len(slots))
	}
}

// After BookSlot, the cache for the instructor must be invalidated so the next
// ListAvailableSlots sees the freshly-booked slot as taken.
func TestBook_InvalidatesCache(t *testing.T) {
	ctx := context.Background()
	f := &fakeStore{}
	seedMondaySchedule(t, f)
	c, _ := newTestCache(t)
	s := NewScheduler(f, c, "acme")

	// Warm the cache.
	if _, _, err := s.AvailableSlots(ctx, "inst-1", "2026-05-04", "2026-05-04"); err != nil {
		t.Fatalf("warm: %v", err)
	}
	// Confirm cache is warm via a second call that hits.
	if _, hit, _ := s.AvailableSlots(ctx, "inst-1", "2026-05-04", "2026-05-04"); !hit {
		t.Fatal("expected warm cache before book")
	}

	if _, err := s.Book(ctx, "", "inst-1", "stu-1", "2026-05-04", "09:15", "Q1 draft"); err != nil {
		t.Fatalf("book: %v", err)
	}

	slots, hit, err := s.AvailableSlots(ctx, "inst-1", "2026-05-04", "2026-05-04")
	if err != nil {
		t.Fatalf("post-book: %v", err)
	}
	if hit {
		t.Fatal("cache should have been invalidated by Book")
	}
	for _, sl := range slots {
		if sl.SlotStartTime == "09:15" {
			t.Fatal("booked slot 09:15 should not appear in availability")
		}
	}
}

// Cancel must also invalidate the cache so the freed slot reappears.
func TestCancel_InvalidatesCache(t *testing.T) {
	ctx := context.Background()
	f := &fakeStore{}
	seedMondaySchedule(t, f)
	c, _ := newTestCache(t)
	s := NewScheduler(f, c, "acme")

	b, err := s.Book(ctx, "", "inst-1", "stu-1", "2026-05-04", "09:30", "context")
	if err != nil {
		t.Fatalf("book: %v", err)
	}
	// Warm cache after book.
	if _, _, err := s.AvailableSlots(ctx, "inst-1", "2026-05-04", "2026-05-04"); err != nil {
		t.Fatalf("warm: %v", err)
	}
	if _, hit, _ := s.AvailableSlots(ctx, "inst-1", "2026-05-04", "2026-05-04"); !hit {
		t.Fatal("expected warm cache before cancel")
	}

	if _, err := s.Cancel(ctx, b.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	slots, hit, err := s.AvailableSlots(ctx, "inst-1", "2026-05-04", "2026-05-04")
	if err != nil {
		t.Fatalf("post-cancel: %v", err)
	}
	if hit {
		t.Fatal("cache should have been invalidated by Cancel")
	}
	// 09:30 slot must now be available again.
	var restored bool
	for _, sl := range slots {
		if sl.SlotStartTime == "09:30" {
			restored = true
		}
	}
	if !restored {
		t.Fatal("cancelled slot did not reappear in availability")
	}
}

// BookSlot's pre_context must round-trip into GetInstructorDay as `questions`
// so instructors can read ahead (W6.4).
func TestInstructorDay_ReturnsPreContextAsQuestions(t *testing.T) {
	ctx := context.Background()
	f := &fakeStore{}
	seedMondaySchedule(t, f)
	s := NewScheduler(f, cache.NoopCache{}, "acme")

	preContext := "Assignment-42 draft: I'm stuck on problem 3 because..."
	if _, err := s.Book(ctx, "", "inst-1", "stu-1", "2026-05-04", "09:00", preContext); err != nil {
		t.Fatalf("book: %v", err)
	}

	bookings, err := s.InstructorDay(ctx, "inst-1", "2026-05-04")
	if err != nil {
		t.Fatalf("day: %v", err)
	}
	if len(bookings) != 1 {
		t.Fatalf("want 1 booking, got %d", len(bookings))
	}
	if bookings[0].Questions == nil || *bookings[0].Questions != preContext {
		t.Fatalf("questions not populated from pre_context: %+v", bookings[0].Questions)
	}
}

// Cancelled bookings must not appear in GetInstructorDay.
func TestInstructorDay_ExcludesCancelled(t *testing.T) {
	ctx := context.Background()
	f := &fakeStore{}
	seedMondaySchedule(t, f)
	s := NewScheduler(f, cache.NoopCache{}, "acme")

	b, err := s.Book(ctx, "", "inst-1", "stu-1", "2026-05-04", "09:00", "ctx")
	if err != nil {
		t.Fatalf("book: %v", err)
	}
	if _, err := s.Cancel(ctx, b.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	bookings, err := s.InstructorDay(ctx, "inst-1", "2026-05-04")
	if err != nil {
		t.Fatalf("day: %v", err)
	}
	if len(bookings) != 0 {
		t.Fatalf("cancelled booking must be excluded, got %d", len(bookings))
	}
}

// Double-book same (instructor, date, start_time) while first is confirmed
// must return ErrSlotTaken.
func TestBook_DoubleBookRejected(t *testing.T) {
	ctx := context.Background()
	f := &fakeStore{}
	seedMondaySchedule(t, f)
	s := NewScheduler(f, cache.NoopCache{}, "acme")

	if _, err := s.Book(ctx, "", "inst-1", "stu-1", "2026-05-04", "09:00", ""); err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err := s.Book(ctx, "", "inst-1", "stu-2", "2026-05-04", "09:00", "")
	if !errors.Is(err, repository.ErrSlotTaken) {
		t.Fatalf("want ErrSlotTaken, got %v", err)
	}
}

// Cancel on an unknown booking returns ErrNotFound.
func TestCancel_NotFound(t *testing.T) {
	ctx := context.Background()
	f := &fakeStore{}
	s := NewScheduler(f, cache.NoopCache{}, "acme")
	_, err := s.Cancel(ctx, "does-not-exist")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// ListAvailableSlots with a different window must miss cache (distinct range).
func TestAvailableSlots_DifferentWindowMissesCache(t *testing.T) {
	ctx := context.Background()
	f := &fakeStore{}
	seedMondaySchedule(t, f)
	c, _ := newTestCache(t)
	s := NewScheduler(f, c, "acme")

	if _, _, err := s.AvailableSlots(ctx, "inst-1", "2026-05-04", "2026-05-04"); err != nil {
		t.Fatalf("warm: %v", err)
	}
	_, hit, err := s.AvailableSlots(ctx, "inst-1", "2026-05-04", "2026-05-11")
	if err != nil {
		t.Fatalf("wider: %v", err)
	}
	if hit {
		t.Fatal("wider window must not satisfy narrow-window cache entry")
	}
}
