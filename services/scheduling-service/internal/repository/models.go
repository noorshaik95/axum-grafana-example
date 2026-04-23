package repository

import "time"

// Schedule is an instructor's recurring weekly office-hours window.
type Schedule struct {
	ID                  string
	InstructorID        string
	DayOfWeek           int // 0=Sunday ... 6=Saturday
	StartTime           string
	EndTime             string
	SlotDurationMinutes int
	Format              string
	Location            *string
	IsActive            bool
	CreatedAt           time.Time
}

// Booking is a confirmed student appointment against a Schedule.
type Booking struct {
	ID            string
	ScheduleID    *string
	InstructorID  string
	StudentID     string
	SlotDate      string // "YYYY-MM-DD"
	SlotStartTime string // "HH:MM"
	Questions     *string
	Status        string
	CreatedAt     time.Time
}
