package grading

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
	"slate/services/assignment-grading-service/pkg/kafka"
)

// Engine handles grading calculations
type Engine struct {
	gradeRepo repository.GradeRepository
	producer  kafka.EventPublisher
}

// NewEngine creates a new grading engine
func NewEngine(gradeRepo repository.GradeRepository, producer kafka.EventPublisher) *Engine {
	return &Engine{
		gradeRepo: gradeRepo,
		producer:  producer,
	}
}

// AutoGrade calculates percentile-based grades for all submissions of an assignment
func (e *Engine) AutoGrade(ctx context.Context, db *sql.DB, assignmentID string) error {
	scores, err := e.gradeRepo.GetScoresForAssignment(ctx, assignmentID)
	if err != nil {
		return fmt.Errorf("failed to get scores: %w", err)
	}

	if len(scores) == 0 {
		return nil
	}

	// Extract all score values for percentile calculation
	allScores := make([]float64, len(scores))
	for i, s := range scores {
		allScores[i] = s.Score
	}

	// Calculate and update percentile for each grade
	for _, gs := range scores {
		percentile := ComputePercentile(gs.Score, allScores)
		if err := e.gradeRepo.UpdatePercentile(ctx, gs.GradeID, percentile); err != nil {
			return fmt.Errorf("failed to update percentile for grade %s: %w", gs.GradeID, err)
		}
	}

	// Emit kafka event
	event := kafka.Event{
		Type:        kafka.EventTypeAutoGraded,
		AggregateID: assignmentID,
		Timestamp:   time.Now(),
		Data: map[string]interface{}{
			"assignment_id": assignmentID,
			"grades_count":  len(scores),
		},
	}
	if err := e.producer.PublishEvent(ctx, event); err != nil {
		fmt.Printf("Failed to publish assignment.auto_graded event: %v\n", err)
	}

	return nil
}

// ComputePercentile returns the percentile rank of a score within a set
func ComputePercentile(score float64, allScores []float64) float64 {
	if len(allScores) == 0 {
		return 0
	}
	if len(allScores) == 1 {
		return 100
	}

	below := 0
	for _, s := range allScores {
		if s < score {
			below++
		}
	}
	return float64(below) / float64(len(allScores)) * 100
}

// LetterGrade converts a percentage to letter grade using the grading scale
func LetterGrade(pct float64, scale []models.GradeScale) string {
	if len(scale) == 0 {
		scale = models.DefaultGradeScale
	}

	// Sort scale by Min descending so we match highest threshold first
	sorted := make([]models.GradeScale, len(scale))
	copy(sorted, scale)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Min > sorted[j].Min
	})

	for _, s := range sorted {
		if pct >= s.Min {
			return s.Grade
		}
	}
	return "F"
}

// ApplyLatePenalty applies a late penalty to a score based on grading rules
func ApplyLatePenalty(score float64, submittedAt, dueDate time.Time, rule models.GradingRule) float64 {
	if submittedAt.Before(dueDate) || dueDate.IsZero() {
		return score
	}
	daysLate := int(math.Ceil(submittedAt.Sub(dueDate).Hours() / 24))
	if daysLate <= 0 {
		return score
	}
	penalty := math.Min(float64(daysLate)*rule.LatePenaltyPerDay, rule.MaxLatePenalty)
	return math.Max(0, score-penalty)
}

// GradeDistribution represents a grade distribution bucket
type GradeDistribution struct {
	LetterGrade string  `json:"letter_grade"`
	Count       int     `json:"count"`
	Percentage  float64 `json:"percentage"`
	MinScore    float64 `json:"min_score"`
	MaxScore    float64 `json:"max_score"`
}

// ComputeDistribution calculates grade distribution for a set of grades
func ComputeDistribution(grades []*models.Grade, scale []models.GradeScale) []GradeDistribution {
	if len(scale) == 0 {
		scale = models.DefaultGradeScale
	}

	// Initialize buckets
	buckets := make(map[string]*GradeDistribution)
	for _, s := range scale {
		buckets[s.Grade] = &GradeDistribution{
			LetterGrade: s.Grade,
			MinScore:    math.MaxFloat64,
			MaxScore:    0,
		}
	}

	total := len(grades)
	for _, g := range grades {
		letter := LetterGrade(g.Percentage, scale)
		if bucket, ok := buckets[letter]; ok {
			bucket.Count++
			if g.Percentage < bucket.MinScore {
				bucket.MinScore = g.Percentage
			}
			if g.Percentage > bucket.MaxScore {
				bucket.MaxScore = g.Percentage
			}
		}
	}

	// Convert to slice and compute percentages
	result := make([]GradeDistribution, 0, len(scale))
	for _, s := range scale {
		bucket := buckets[s.Grade]
		if total > 0 {
			bucket.Percentage = float64(bucket.Count) / float64(total) * 100
		}
		if bucket.Count == 0 {
			bucket.MinScore = 0
			bucket.MaxScore = 0
		}
		result = append(result, *bucket)
	}

	return result
}
