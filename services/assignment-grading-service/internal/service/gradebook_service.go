package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"

	"slate/services/assignment-grading-service/internal/grading"
	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
)

type gradebookService struct {
	assignmentRepo repository.AssignmentRepository
	submissionRepo repository.SubmissionRepository
	gradeRepo      repository.GradeRepository
	ruleRepo       repository.GradingRuleRepository
	estimator      *grading.Estimator
}

// NewGradebookService creates a new gradebook service
func NewGradebookService(
	assignmentRepo repository.AssignmentRepository,
	submissionRepo repository.SubmissionRepository,
	gradeRepo repository.GradeRepository,
) GradebookService {
	return &gradebookService{
		assignmentRepo: assignmentRepo,
		submissionRepo: submissionRepo,
		gradeRepo:      gradeRepo,
	}
}

// NewGradebookServiceFull creates a gradebook service with grading rules support
func NewGradebookServiceFull(
	assignmentRepo repository.AssignmentRepository,
	submissionRepo repository.SubmissionRepository,
	gradeRepo repository.GradeRepository,
	ruleRepo repository.GradingRuleRepository,
) GradebookService {
	return &gradebookService{
		assignmentRepo: assignmentRepo,
		submissionRepo: submissionRepo,
		gradeRepo:      gradeRepo,
		ruleRepo:       ruleRepo,
		estimator:      grading.NewEstimator(gradeRepo, ruleRepo, assignmentRepo),
	}
}

// GetStudentGradebook gets the gradebook for a single student
func (s *gradebookService) GetStudentGradebook(ctx context.Context, studentID, courseID string) (*StudentGradebook, error) {
	assignments, _, err := s.assignmentRepo.ListByCourse(ctx, courseID, 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignments: %w", err)
	}

	grades, err := s.gradeRepo.ListByStudent(ctx, studentID, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to list grades: %w", err)
	}

	gradeMap := make(map[string]*models.Grade)
	for _, grade := range grades {
		gradeMap[grade.AssignmentID] = grade
	}

	submissions, err := s.submissionRepo.ListByStudent(ctx, studentID, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to list submissions: %w", err)
	}

	submissionMap := make(map[string]*models.Submission)
	for _, submission := range submissions {
		submissionMap[submission.AssignmentID] = submission
	}

	entries := make([]GradebookEntry, 0, len(assignments))
	var totalPoints, earnedPoints float64

	for _, assignment := range assignments {
		entry := GradebookEntry{
			AssignmentID:    assignment.ID,
			AssignmentTitle: assignment.Title,
			MaxPoints:       assignment.MaxPoints,
			DueDate:         assignment.DueDate,
		}

		if submission, ok := submissionMap[assignment.ID]; ok {
			entry.SubmittedAt = &submission.SubmittedAt
			entry.IsLate = submission.IsLate
		}

		if grade, ok := gradeMap[assignment.ID]; ok {
			entry.Score = grade.Score
			entry.AdjustedScore = grade.AdjustedScore
			entry.Status = grade.Status
			earnedPoints += grade.AdjustedScore
		} else {
			entry.Status = "not_graded"
		}

		totalPoints += assignment.MaxPoints
		entries = append(entries, entry)
	}

	percentage := 0.0
	if totalPoints > 0 {
		percentage = (earnedPoints / totalPoints) * 100
	}

	letterGrade := calculateLetterGrade(percentage)

	return &StudentGradebook{
		StudentID:    studentID,
		CourseID:     courseID,
		Entries:      entries,
		TotalPoints:  totalPoints,
		EarnedPoints: earnedPoints,
		Percentage:   percentage,
		LetterGrade:  letterGrade,
	}, nil
}

// GetCourseGradebook gets the gradebook for an entire course
func (s *gradebookService) GetCourseGradebook(ctx context.Context, courseID string) (*CourseGradebook, error) {
	assignments, _, err := s.assignmentRepo.ListByCourse(ctx, courseID, 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignments: %w", err)
	}

	grades, err := s.gradeRepo.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to list grades: %w", err)
	}

	studentGrades := make(map[string][]*models.Grade)
	for _, grade := range grades {
		studentGrades[grade.StudentID] = append(studentGrades[grade.StudentID], grade)
	}

	var totalPoints float64
	for _, assignment := range assignments {
		totalPoints += assignment.MaxPoints
	}

	students := make([]StudentSummary, 0, len(studentGrades))
	for studentID, studentGradeList := range studentGrades {
		var earnedPoints float64
		gradeMap := make(map[string]*models.Grade)
		for _, grade := range studentGradeList {
			gradeMap[grade.AssignmentID] = grade
			earnedPoints += grade.AdjustedScore
		}

		var entries []GradebookEntry
		for _, assignment := range assignments {
			entry := GradebookEntry{
				AssignmentID:    assignment.ID,
				AssignmentTitle: assignment.Title,
				MaxPoints:       assignment.MaxPoints,
				DueDate:         assignment.DueDate,
			}

			if grade, ok := gradeMap[assignment.ID]; ok {
				entry.Score = grade.Score
				entry.AdjustedScore = grade.AdjustedScore
				entry.Status = grade.Status
			} else {
				entry.Status = "not_graded"
			}

			entries = append(entries, entry)
		}

		percentage := 0.0
		if totalPoints > 0 {
			percentage = (earnedPoints / totalPoints) * 100
		}

		students = append(students, StudentSummary{
			StudentID:    studentID,
			TotalPoints:  totalPoints,
			EarnedPoints: earnedPoints,
			Percentage:   percentage,
			LetterGrade:  calculateLetterGrade(percentage),
			Entries:      entries,
		})
	}

	return &CourseGradebook{
		CourseID: courseID,
		Students: students,
	}, nil
}

// GetGradeStatistics gets statistics for an assignment
func (s *gradebookService) GetGradeStatistics(ctx context.Context, assignmentID string) (*repository.GradeStatistics, error) {
	stats, err := s.gradeRepo.GetStatistics(ctx, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}
	return stats, nil
}

// ExportGrades exports grades to CSV format
func (s *gradebookService) ExportGrades(ctx context.Context, courseID, format string) ([]byte, error) {
	if format != "csv" {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	gradebook, err := s.GetCourseGradebook(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get gradebook: %w", err)
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	header := []string{"Student ID", "Total Points", "Earned Points", "Percentage", "Letter Grade"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, student := range gradebook.Students {
		row := []string{
			student.StudentID,
			fmt.Sprintf("%.2f", student.TotalPoints),
			fmt.Sprintf("%.2f", student.EarnedPoints),
			fmt.Sprintf("%.2f", student.Percentage),
			student.LetterGrade,
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// GetStudentGradesByTenant returns all grades for a student in a tenant
func (s *gradebookService) GetStudentGradesByTenant(ctx context.Context, tenantID, studentID string) ([]*models.Grade, error) {
	grades, err := s.gradeRepo.ListByStudentTenant(ctx, tenantID, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student grades: %w", err)
	}
	return grades, nil
}

// GetGradeDistribution returns the grade distribution for a course
func (s *gradebookService) GetGradeDistribution(ctx context.Context, courseID string) ([]grading.GradeDistribution, error) {
	grades, err := s.gradeRepo.ListByCourse(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get course grades: %w", err)
	}

	return grading.ComputeDistribution(grades, nil), nil
}

// EstimateFinalGrade computes a weighted grade estimate for a student
func (s *gradebookService) EstimateFinalGrade(ctx context.Context, studentID, courseID, tenantID string) (*grading.GradeEstimate, error) {
	if s.estimator == nil {
		return nil, fmt.Errorf("grade estimation not available: grading rules not configured")
	}
	return s.estimator.EstimateFinalGrade(ctx, studentID, courseID, tenantID)
}

// calculateLetterGrade converts a percentage to a letter grade
func calculateLetterGrade(percentage float64) string {
	return grading.LetterGrade(percentage, nil)
}
