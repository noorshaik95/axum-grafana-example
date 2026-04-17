package models

import (
	"encoding/json"
	"errors"
	"time"
)

// GradeScale represents a single grade threshold in the grading scale
type GradeScale struct {
	Grade string  `json:"grade"`
	Min   float64 `json:"min"`
}

// DefaultGradeScale is the default A-F grading scale
var DefaultGradeScale = []GradeScale{
	{Grade: "A", Min: 90},
	{Grade: "B", Min: 80},
	{Grade: "C", Min: 70},
	{Grade: "D", Min: 60},
	{Grade: "F", Min: 0},
}

// GradingRule defines grading rules for a course or tenant
type GradingRule struct {
	ID                 string       `json:"id"`
	TenantID           string       `json:"tenant_id"`
	CourseID           string       `json:"course_id,omitempty"`
	AssignmentType     string       `json:"assignment_type,omitempty"`
	Weight             float64      `json:"weight"`
	LatePenaltyPerDay  float64      `json:"late_penalty_per_day"`
	MaxLatePenalty     float64      `json:"max_late_penalty"`
	GradeScale         []GradeScale `json:"grade_scale"`
	CreatedAt          time.Time    `json:"created_at"`
}

// GradeScaleJSON returns grade scale as JSON bytes
func (r *GradingRule) GradeScaleJSON() ([]byte, error) {
	if r.GradeScale == nil {
		return json.Marshal(DefaultGradeScale)
	}
	return json.Marshal(r.GradeScale)
}

// SetGradeScaleFromJSON parses grade scale from JSON
func (r *GradingRule) SetGradeScaleFromJSON(data []byte) error {
	if data == nil {
		r.GradeScale = DefaultGradeScale
		return nil
	}
	return json.Unmarshal(data, &r.GradeScale)
}

// Validate checks if the grading rule has valid data
func (r *GradingRule) Validate() error {
	if r.TenantID == "" {
		return errors.New("tenant_id is required")
	}

	if r.Weight < 0 || r.Weight > 1 {
		return errors.New("weight must be between 0 and 1")
	}

	if r.LatePenaltyPerDay < 0 {
		return errors.New("late_penalty_per_day must be non-negative")
	}

	if r.MaxLatePenalty < 0 {
		return errors.New("max_late_penalty must be non-negative")
	}

	if r.AssignmentType != "" {
		validTypes := map[string]bool{"exam": true, "homework": true, "quiz": true, "project": true}
		if !validTypes[r.AssignmentType] {
			return errors.New("assignment_type must be one of: exam, homework, quiz, project")
		}
	}

	return nil
}
