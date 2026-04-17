package models

import (
	"encoding/json"
	"errors"
	"time"
)

// RubricCriterion defines a single rubric criterion
type RubricCriterion struct {
	Criterion   string  `json:"criterion"`
	MaxPoints   float64 `json:"max_points"`
	Description string  `json:"description"`
}

// LatePolicy defines the late submission policy for an assignment
type LatePolicy struct {
	PenaltyPercentPerDay int `json:"penalty_percent_per_day"` // Percentage deduction per day late (0-100)
	MaxLateDays          int `json:"max_late_days"`           // Maximum days late before zero (0 = no late submissions)
}

// Assignment represents a course assignment
type Assignment struct {
	ID               string            `json:"id"`
	TenantID         string            `json:"tenant_id"`
	CourseID         string            `json:"course_id"`
	InstructorID     string            `json:"instructor_id"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	MaxPoints        float64           `json:"max_points"`
	Rubric           []RubricCriterion `json:"rubric,omitempty"`
	AssignmentType   string            `json:"assignment_type,omitempty"`
	DueDate          time.Time         `json:"due_date"`
	MaxFileSizeMB    int               `json:"max_file_size_mb"`
	AllowedFileTypes []string          `json:"allowed_file_types,omitempty"`
	IsDeleted        bool              `json:"is_deleted"`
	LatePolicy       LatePolicy        `json:"late_policy"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// RubricJSON returns the rubric as a JSON byte slice for database storage
func (a *Assignment) RubricJSON() ([]byte, error) {
	if a.Rubric == nil {
		return nil, nil
	}
	return json.Marshal(a.Rubric)
}

// SetRubricFromJSON parses rubric JSON from database
func (a *Assignment) SetRubricFromJSON(data []byte) error {
	if data == nil {
		return nil
	}
	return json.Unmarshal(data, &a.Rubric)
}

// Validate checks if the assignment has valid data
func (a *Assignment) Validate() error {
	if a.CourseID == "" {
		return errors.New("course_id is required")
	}

	if a.Title == "" {
		return errors.New("title is required")
	}

	if len(a.Title) > 500 {
		return errors.New("title must be 500 characters or less")
	}

	if a.MaxPoints <= 0 {
		return errors.New("max_points must be greater than 0")
	}

	if a.DueDate.IsZero() {
		return errors.New("due_date is required")
	}

	if a.AssignmentType != "" {
		validTypes := map[string]bool{"exam": true, "homework": true, "quiz": true, "project": true}
		if !validTypes[a.AssignmentType] {
			return errors.New("assignment_type must be one of: exam, homework, quiz, project")
		}
	}

	return a.LatePolicy.Validate()
}

// Validate checks if the late policy has valid data
func (lp *LatePolicy) Validate() error {
	if lp.PenaltyPercentPerDay < 0 || lp.PenaltyPercentPerDay > 100 {
		return errors.New("penalty_percent_per_day must be between 0 and 100")
	}

	if lp.MaxLateDays < 0 {
		return errors.New("max_late_days must be non-negative")
	}

	return nil
}

// AllowsLateSubmissions returns true if the assignment accepts late submissions
func (lp *LatePolicy) AllowsLateSubmissions() bool {
	return lp.MaxLateDays > 0
}
