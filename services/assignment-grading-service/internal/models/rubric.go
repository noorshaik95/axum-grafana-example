package models

import (
	"errors"
	"time"
)

// RubricRow is a single row in an assignment's rubric (W9.1).
type RubricRow struct {
	ID           string    `json:"id"`
	AssignmentID string    `json:"assignment_id"`
	Title        string    `json:"title"`
	MaxPoints    float64   `json:"max_points"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Validate checks the rubric row for required fields.
func (r *RubricRow) Validate() error {
	if r.AssignmentID == "" {
		return errors.New("assignment_id is required")
	}
	if r.Title == "" {
		return errors.New("title is required")
	}
	if len(r.Title) > 500 {
		return errors.New("title must be 500 characters or less")
	}
	if r.MaxPoints <= 0 {
		return errors.New("max_points must be greater than 0")
	}
	return nil
}
