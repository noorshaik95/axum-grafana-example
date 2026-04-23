package service

import (
	"testing"
	"time"

	"slate/services/scheduling-service/internal/repository"
)

// helper — fixed date: 2026-05-04 is a Monday (Weekday = 1).
// Use within-range dates for generation so day-of-week coverage is exercised.
func parse(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return d
}

// monday / tuesday / wednesday — confirm Weekday matches schema (0=Sunday..6=Saturday).
func TestWeekdayMapping(t *testing.T) {
	cases := map[string]int{
		"2026-05-03": 0, // Sunday
		"2026-05-04": 1, // Monday
		"2026-05-05": 2, // Tuesday
		"2026-05-06": 3,
		"2026-05-09": 6, // Saturday
	}
	for date, want := range cases {
		got := int(parse(t, date).Weekday())
		if got != want {
			t.Errorf("%s: want weekday %d, got %d", date, want, got)
		}
	}
}

// Schedule window 09:00–10:00 on Monday with 15-minute slots → 4 slots / Monday.
// Range 2026-05-04 (Mon) to 2026-05-10 (Sun, inclusive) → one Monday, one day that
// matches a second schedule, others skipped.
func TestGenerateSlots_DayOfWeekAndWindow(t *testing.T) {
	schedules := []repository.Schedule{
		{
			ID: "sched-mon", InstructorID: "inst-1",
			DayOfWeek: 1, // Monday
			StartTime: "09:00", EndTime: "10:00",
			SlotDurationMinutes: 15,
			Format:              "online", IsActive: true,
		},
		{
			ID: "sched-fri", InstructorID: "inst-1",
			DayOfWeek: 5, // Friday
			StartTime: "13:00", EndTime: "14:30",
			SlotDurationMinutes: 30,
			Format:              "in_person", IsActive: true,
		},
		// Inactive schedule — must be ignored even though DOW matches.
		{
			ID: "sched-tue", InstructorID: "inst-1",
			DayOfWeek: 2,
			StartTime: "08:00", EndTime: "09:00",
			SlotDurationMinutes: 15,
			Format:              "online", IsActive: false,
		},
	}

	slots, err := GenerateSlots(schedules, nil, parse(t, "2026-05-04"), parse(t, "2026-05-10"))
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Monday: 09:00, 09:15, 09:30, 09:45 (4 slots) — Friday: 13:00, 13:30, 14:00 (3 slots).
	// Tuesday inactive schedule contributes 0.
	if got := len(slots); got != 7 {
		t.Fatalf("want 7 slots, got %d: %+v", got, slots)
	}

	// Spot-check Monday slot contents.
	want := map[string]string{
		"2026-05-04|09:00": "09:15",
		"2026-05-04|09:45": "10:00",
		"2026-05-08|14:00": "14:30",
	}
	got := map[string]string{}
	for _, s := range slots {
		got[s.SlotDate+"|"+s.SlotStartTime] = s.SlotEndTime
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("slot %q: want end %q, got %q", k, v, got[k])
		}
	}
}

// An existing confirmed booking at (date, start_time) must be subtracted from
// the generated slot list for that same instructor.
func TestGenerateSlots_SubtractsBooked(t *testing.T) {
	schedules := []repository.Schedule{{
		ID: "s1", InstructorID: "inst-1",
		DayOfWeek: 1, StartTime: "09:00", EndTime: "10:00",
		SlotDurationMinutes: 15, Format: "online", IsActive: true,
	}}
	booked := []repository.Booking{
		{InstructorID: "inst-1", SlotDate: "2026-05-04", SlotStartTime: "09:15", Status: "confirmed"},
		{InstructorID: "inst-1", SlotDate: "2026-05-04", SlotStartTime: "09:45", Status: "confirmed"},
	}
	slots, err := GenerateSlots(schedules, booked, parse(t, "2026-05-04"), parse(t, "2026-05-04"))
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("want 2 remaining slots, got %d: %+v", len(slots), slots)
	}
	for _, s := range slots {
		if s.SlotStartTime == "09:15" || s.SlotStartTime == "09:45" {
			t.Errorf("booked slot %q was not subtracted", s.SlotStartTime)
		}
	}
}

// Window that doesn't divide evenly by slot_duration must drop the partial tail.
// 09:00–09:40 with 15m → 09:00, 09:15 (two full slots, 09:30 would overflow to 09:45 > 09:40).
func TestGenerateSlots_PartialTailDropped(t *testing.T) {
	schedules := []repository.Schedule{{
		ID: "s1", InstructorID: "inst-1",
		DayOfWeek: 1, StartTime: "09:00", EndTime: "09:40",
		SlotDurationMinutes: 15, Format: "online", IsActive: true,
	}}
	slots, err := GenerateSlots(schedules, nil, parse(t, "2026-05-04"), parse(t, "2026-05-04"))
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("want 2 slots, got %d: %+v", len(slots), slots)
	}
}

func TestGenerateSlots_InvalidRange(t *testing.T) {
	_, err := GenerateSlots(nil, nil, parse(t, "2026-05-10"), parse(t, "2026-05-04"))
	if err == nil {
		t.Fatal("expected error for inverted range")
	}
}
