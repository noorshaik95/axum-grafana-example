// Package service holds the slot-availability algorithm and booking orchestration
// for scheduling-service (W6.3, W6.4). Pure logic where possible so unit tests
// don't need Postgres/Redis.
package service

import (
	"fmt"
	"time"

	"slate/services/scheduling-service/internal/cache"
	"slate/services/scheduling-service/internal/repository"
)

// GenerateSlots expands a set of weekly Schedule windows into concrete slots
// across [dateFrom, dateTo] inclusive, then subtracts any already-booked
// (slot_date, slot_start_time) pairs from `booked`.
//
// Pure function, no I/O — callers are responsible for fetching schedules and
// bookings first.
func GenerateSlots(schedules []repository.Schedule, booked []repository.Booking, dateFrom, dateTo time.Time) ([]cache.Slot, error) {
	if dateTo.Before(dateFrom) {
		return nil, fmt.Errorf("date_to (%s) is before date_from (%s)",
			dateTo.Format("2006-01-02"), dateFrom.Format("2006-01-02"))
	}

	// Set of "YYYY-MM-DD|HH:MM" already taken.
	taken := make(map[string]struct{}, len(booked))
	for _, b := range booked {
		taken[b.SlotDate+"|"+b.SlotStartTime] = struct{}{}
	}

	// Index active schedules by day-of-week for O(1) lookup per date.
	byDOW := make(map[int][]repository.Schedule, 7)
	for _, s := range schedules {
		if !s.IsActive {
			continue
		}
		if s.DayOfWeek < 0 || s.DayOfWeek > 6 {
			continue
		}
		byDOW[s.DayOfWeek] = append(byDOW[s.DayOfWeek], s)
	}

	var out []cache.Slot
	for day := truncateToDay(dateFrom); !day.After(truncateToDay(dateTo)); day = day.AddDate(0, 0, 1) {
		dow := int(day.Weekday()) // time.Weekday: 0=Sunday ... 6=Saturday — matches schema.
		daySchedules, ok := byDOW[dow]
		if !ok {
			continue
		}
		for _, s := range daySchedules {
			slots, err := expandSchedule(s, day)
			if err != nil {
				return nil, err
			}
			for _, sl := range slots {
				if _, isTaken := taken[sl.SlotDate+"|"+sl.SlotStartTime]; isTaken {
					continue
				}
				out = append(out, sl)
			}
		}
	}
	return out, nil
}

// expandSchedule enumerates [start_time, end_time) in slot_duration_minutes steps
// for a given concrete date. Half-open interval: a window 09:00–10:00 with 15m
// slots yields 09:00, 09:15, 09:30, 09:45 (four slots), not five.
func expandSchedule(s repository.Schedule, date time.Time) ([]cache.Slot, error) {
	start, err := parseHM(s.StartTime)
	if err != nil {
		return nil, fmt.Errorf("schedule %s start_time: %w", s.ID, err)
	}
	end, err := parseHM(s.EndTime)
	if err != nil {
		return nil, fmt.Errorf("schedule %s end_time: %w", s.ID, err)
	}
	if s.SlotDurationMinutes <= 0 {
		return nil, fmt.Errorf("schedule %s has invalid slot_duration %d", s.ID, s.SlotDurationMinutes)
	}
	if !start.Before(end) {
		return nil, fmt.Errorf("schedule %s has start_time >= end_time", s.ID)
	}
	dateStr := date.Format("2006-01-02")
	var loc string
	if s.Location != nil {
		loc = *s.Location
	}
	dur := time.Duration(s.SlotDurationMinutes) * time.Minute
	var slots []cache.Slot
	for t := start; t.Add(dur).Compare(end) <= 0; t = t.Add(dur) {
		slots = append(slots, cache.Slot{
			ScheduleID:    s.ID,
			InstructorID:  s.InstructorID,
			SlotDate:      dateStr,
			SlotStartTime: formatHM(t),
			SlotEndTime:   formatHM(t.Add(dur)),
			Format:        s.Format,
			Location:      loc,
		})
	}
	return slots, nil
}

// parseHM parses "HH:MM" into a time within an arbitrary fixed day so arithmetic
// with time.Duration stays timezone-neutral.
func parseHM(hm string) (time.Time, error) {
	t, err := time.Parse("15:04", hm)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func formatHM(t time.Time) string {
	return t.Format("15:04")
}

func truncateToDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// ParseDate parses "YYYY-MM-DD" into a UTC time at midnight.
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, time.UTC)
}
