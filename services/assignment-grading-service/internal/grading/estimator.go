package grading

import (
	"context"
	"fmt"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
)

// TypeBreakdown contains grade breakdown for a single assignment type
type TypeBreakdown struct {
	AssignmentType string  `json:"assignment_type"`
	Average        float64 `json:"average"`
	Weight         float64 `json:"weight"`
	GradedCount    int     `json:"graded_count"`
	TotalCount     int     `json:"total_count"`
}

// GradeEstimate contains the estimated final grade for a student
type GradeEstimate struct {
	Percentage  float64                  `json:"percentage"`
	LetterGrade string                   `json:"letter_grade"`
	Breakdown   map[string]TypeBreakdown `json:"breakdown"`
	Confidence  float64                  `json:"confidence"` // 0-1
}

// Estimator computes grade estimates
type Estimator struct {
	gradeRepo      repository.GradeRepository
	ruleRepo       repository.GradingRuleRepository
	assignmentRepo repository.AssignmentRepository
}

// NewEstimator creates a new grade estimator
func NewEstimator(
	gradeRepo repository.GradeRepository,
	ruleRepo repository.GradingRuleRepository,
	assignmentRepo repository.AssignmentRepository,
) *Estimator {
	return &Estimator{
		gradeRepo:      gradeRepo,
		ruleRepo:       ruleRepo,
		assignmentRepo: assignmentRepo,
	}
}

// EstimateFinalGrade computes weighted grade estimate for a student in a course
func (e *Estimator) EstimateFinalGrade(ctx context.Context, studentID, courseID, tenantID string) (*GradeEstimate, error) {
	// Get grading rules for course (or tenant default)
	rules, err := e.ruleRepo.GetForCourseOrTenant(ctx, tenantID, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get grading rules: %w", err)
	}

	// Get all grades for student in course
	grades, err := e.gradeRepo.ListByStudent(ctx, studentID, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student grades: %w", err)
	}

	// Get all assignments for the course to know total counts
	assignments, _, err := e.assignmentRepo.ListByCourse(ctx, courseID, 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}

	// Build weight map from rules (assignment_type -> rule)
	weightMap := make(map[string]*models.GradingRule)
	var gradeScale []models.GradeScale
	for _, rule := range rules {
		if rule.AssignmentType != "" {
			weightMap[rule.AssignmentType] = rule
		}
		if len(rule.GradeScale) > 0 {
			gradeScale = rule.GradeScale
		}
	}

	if len(gradeScale) == 0 {
		gradeScale = models.DefaultGradeScale
	}

	// Group grades by assignment type
	gradesByType := make(map[string][]float64)
	for _, grade := range grades {
		// Find the assignment to get its type
		for _, a := range assignments {
			if a.ID == grade.AssignmentID {
				aType := a.AssignmentType
				if aType == "" {
					aType = "other"
				}
				if grade.MaxScore > 0 {
					gradesByType[aType] = append(gradesByType[aType], grade.AdjustedScore/grade.MaxScore*100)
				} else if grade.Percentage > 0 {
					gradesByType[aType] = append(gradesByType[aType], grade.Percentage)
				}
				break
			}
		}
	}

	// Count assignments by type
	assignmentsByType := make(map[string]int)
	for _, a := range assignments {
		aType := a.AssignmentType
		if aType == "" {
			aType = "other"
		}
		assignmentsByType[aType]++
	}

	// Calculate weighted average
	weightedSum := 0.0
	totalWeight := 0.0
	totalGraded := 0
	totalAssignments := len(assignments)
	breakdown := make(map[string]TypeBreakdown)

	for aType, typeGrades := range gradesByType {
		if len(typeGrades) == 0 {
			continue
		}

		avg := average(typeGrades)
		weight := 1.0 // default equal weight

		if rule, ok := weightMap[aType]; ok {
			weight = rule.Weight
		}

		weightedSum += avg * weight
		totalWeight += weight
		totalGraded += len(typeGrades)

		breakdown[aType] = TypeBreakdown{
			AssignmentType: aType,
			Average:        avg,
			Weight:         weight,
			GradedCount:    len(typeGrades),
			TotalCount:     assignmentsByType[aType],
		}
	}

	// Also include types with no grades yet in breakdown
	for aType, count := range assignmentsByType {
		if _, ok := breakdown[aType]; !ok {
			weight := 1.0
			if rule, ok := weightMap[aType]; ok {
				weight = rule.Weight
			}
			breakdown[aType] = TypeBreakdown{
				AssignmentType: aType,
				Average:        0,
				Weight:         weight,
				GradedCount:    0,
				TotalCount:     count,
			}
		}
	}

	estimated := 0.0
	if totalWeight > 0 {
		estimated = weightedSum / totalWeight
	}

	// Confidence = proportion of assignments graded
	confidence := 0.0
	if totalAssignments > 0 {
		confidence = float64(totalGraded) / float64(totalAssignments)
	}

	return &GradeEstimate{
		Percentage:  estimated,
		LetterGrade: LetterGrade(estimated, gradeScale),
		Breakdown:   breakdown,
		Confidence:  confidence,
	}, nil
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
