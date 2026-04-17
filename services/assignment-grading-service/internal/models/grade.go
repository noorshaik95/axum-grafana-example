package models

import (
	"encoding/json"
	"errors"
	"time"
)

// Grade status constants
const (
	GradeStatusDraft     = "draft"
	GradeStatusPublished = "published"
)

// Grade represents a grade for a submission
type Grade struct {
	ID                    string                 `json:"id"`
	TenantID              string                 `json:"tenant_id"`
	SubmissionID          string                 `json:"submission_id"`
	StudentID             string                 `json:"student_id"`
	AssignmentID          string                 `json:"assignment_id"`
	CourseID              string                 `json:"course_id"`
	Score                 float64                `json:"score"`
	MaxScore              float64                `json:"max_score"`
	AdjustedScore         float64                `json:"adjusted_score"`
	Percentage            float64                `json:"percentage"`
	LetterGrade           string                 `json:"letter_grade,omitempty"`
	RubricScores          map[string]interface{} `json:"rubric_scores,omitempty"`
	Feedback              string                 `json:"feedback"`
	Status                string                 `json:"status"`
	GradedAt              *time.Time             `json:"graded_at,omitempty"`
	PublishedAt           *time.Time             `json:"published_at,omitempty"`
	GradedBy              string                 `json:"graded_by"`
	OverrideJustification string                 `json:"override_justification,omitempty"`
	Percentile            float64                `json:"percentile"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// RubricScoresJSON returns rubric scores as JSON bytes
func (g *Grade) RubricScoresJSON() ([]byte, error) {
	if g.RubricScores == nil {
		return nil, nil
	}
	return json.Marshal(g.RubricScores)
}

// SetRubricScoresFromJSON parses rubric scores from JSON
func (g *Grade) SetRubricScoresFromJSON(data []byte) error {
	if data == nil {
		return nil
	}
	return json.Unmarshal(data, &g.RubricScores)
}

// Validate checks if the grade has valid data
func (g *Grade) Validate() error {
	if g.SubmissionID == "" {
		return errors.New("submission_id is required")
	}

	if g.StudentID == "" {
		return errors.New("student_id is required")
	}

	if g.AssignmentID == "" {
		return errors.New("assignment_id is required")
	}

	if g.Score < 0 {
		return errors.New("score must be non-negative")
	}

	if g.AdjustedScore < 0 {
		return errors.New("adjusted_score must be non-negative")
	}

	if g.GradedBy == "" {
		return errors.New("graded_by is required")
	}

	if !isValidGradeStatus(g.Status) {
		return errors.New("invalid status")
	}

	return nil
}

// ValidateScore checks if the score is within the allowed range
func (g *Grade) ValidateScore(maxPoints float64) error {
	if g.Score > maxPoints {
		return errors.New("score exceeds max_points")
	}

	if g.AdjustedScore > maxPoints {
		return errors.New("adjusted_score exceeds max_points")
	}

	return nil
}

// isValidGradeStatus checks if the grade status is valid
func isValidGradeStatus(status string) bool {
	switch status {
	case GradeStatusDraft, GradeStatusPublished:
		return true
	default:
		return false
	}
}

// Publish marks the grade as published
func (g *Grade) Publish() error {
	if g.Status == GradeStatusPublished {
		return errors.New("grade is already published")
	}

	now := time.Now()
	g.Status = GradeStatusPublished
	g.PublishedAt = &now
	g.UpdatedAt = now

	return nil
}

// IsDraft returns true if the grade is in draft status
func (g *Grade) IsDraft() bool {
	return g.Status == GradeStatusDraft
}

// IsPublished returns true if the grade is published
func (g *Grade) IsPublished() bool {
	return g.Status == GradeStatusPublished
}
