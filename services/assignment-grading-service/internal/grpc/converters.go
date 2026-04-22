package grpc

import (
	pb "slate/services/assignment-grading-service/api/proto"
	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
	"slate/services/assignment-grading-service/internal/service"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Assignment conversions

func assignmentToProto(a *models.Assignment) *pb.Assignment {
	if a == nil {
		return nil
	}

	return &pb.Assignment{
		Id:          a.ID,
		CourseId:    a.CourseID,
		Title:       a.Title,
		Description: a.Description,
		MaxPoints:   a.MaxPoints,
		DueDate:     timestamppb.New(a.DueDate),
		LatePolicy:  latePolicyToProto(&a.LatePolicy),
		CreatedAt:   timestamppb.New(a.CreatedAt),
		UpdatedAt:   timestamppb.New(a.UpdatedAt),
	}
}

func latePolicyToProto(lp *models.LatePolicy) *pb.LatePolicy {
	if lp == nil {
		return nil
	}

	return &pb.LatePolicy{
		PenaltyPercentPerDay: int32(lp.PenaltyPercentPerDay),
		MaxLateDays:          int32(lp.MaxLateDays),
	}
}

func protoToLatePolicy(lp *pb.LatePolicy) models.LatePolicy {
	if lp == nil {
		return models.LatePolicy{}
	}

	return models.LatePolicy{
		PenaltyPercentPerDay: int(lp.PenaltyPercentPerDay),
		MaxLateDays:          int(lp.MaxLateDays),
	}
}

// Submission conversions

func submissionToProto(s *models.Submission) *pb.Submission {
	if s == nil {
		return nil
	}

	return &pb.Submission{
		Id:           s.ID,
		AssignmentId: s.AssignmentID,
		StudentId:    s.StudentID,
		FilePath:     s.FilePath,
		SubmittedAt:  timestamppb.New(s.SubmittedAt),
		Status:       s.Status,
		IsLate:       s.IsLate,
		DaysLate:     int32(s.DaysLate),
		CreatedAt:    timestamppb.New(s.CreatedAt),
		UpdatedAt:    timestamppb.New(s.UpdatedAt),
	}
}

// Grade conversions

func gradeToProto(g *models.Grade) *pb.Grade {
	if g == nil {
		return nil
	}

	grade := &pb.Grade{
		Id:            g.ID,
		SubmissionId:  g.SubmissionID,
		StudentId:     g.StudentID,
		AssignmentId:  g.AssignmentID,
		Score:         g.Score,
		AdjustedScore: g.AdjustedScore,
		Feedback:      g.Feedback,
		Status:        g.Status,
		GradedBy:      g.GradedBy,
		CreatedAt:     timestamppb.New(g.CreatedAt),
		UpdatedAt:     timestamppb.New(g.UpdatedAt),
	}

	if g.GradedAt != nil {
		grade.GradedAt = timestamppb.New(*g.GradedAt)
	}

	if g.PublishedAt != nil {
		grade.PublishedAt = timestamppb.New(*g.PublishedAt)
	}

	return grade
}

// Gradebook conversions

func gradebookEntryToProto(e *service.GradebookEntry) *pb.GradebookEntry {
	if e == nil {
		return nil
	}

	entry := &pb.GradebookEntry{
		AssignmentId:    e.AssignmentID,
		AssignmentTitle: e.AssignmentTitle,
		MaxPoints:       e.MaxPoints,
		Score:           e.Score,
		AdjustedScore:   e.AdjustedScore,
		Status:          e.Status,
		DueDate:         timestamppb.New(e.DueDate),
		IsLate:          e.IsLate,
	}

	if e.SubmittedAt != nil {
		entry.SubmittedAt = timestamppb.New(*e.SubmittedAt)
	}

	return entry
}

func studentGradebookToProto(sg *service.StudentGradebook) *pb.GetStudentGradebookResponse {
	if sg == nil {
		return nil
	}

	entries := make([]*pb.GradebookEntry, 0, len(sg.Entries))
	for i := range sg.Entries {
		entries = append(entries, gradebookEntryToProto(&sg.Entries[i]))
	}

	return &pb.GetStudentGradebookResponse{
		StudentId:    sg.StudentID,
		CourseId:     sg.CourseID,
		Entries:      entries,
		TotalPoints:  sg.TotalPoints,
		EarnedPoints: sg.EarnedPoints,
		Percentage:   sg.Percentage,
		LetterGrade:  sg.LetterGrade,
	}
}

func studentSummaryToProto(ss *service.StudentSummary) *pb.StudentGradeSummary {
	if ss == nil {
		return nil
	}

	entries := make([]*pb.GradebookEntry, 0, len(ss.Entries))
	for i := range ss.Entries {
		entries = append(entries, gradebookEntryToProto(&ss.Entries[i]))
	}

	return &pb.StudentGradeSummary{
		StudentId:    ss.StudentID,
		TotalPoints:  ss.TotalPoints,
		EarnedPoints: ss.EarnedPoints,
		Percentage:   ss.Percentage,
		LetterGrade:  ss.LetterGrade,
		Entries:      entries,
	}
}

func courseGradebookToProto(cg *service.CourseGradebook) *pb.GetCourseGradebookResponse {
	if cg == nil {
		return nil
	}

	students := make([]*pb.StudentGradeSummary, 0, len(cg.Students))
	for i := range cg.Students {
		students = append(students, studentSummaryToProto(&cg.Students[i]))
	}

	return &pb.GetCourseGradebookResponse{
		CourseId: cg.CourseID,
		Students: students,
	}
}

// Grading queue conversions (W9.4)

func patternGroupToProto(g *service.PatternGroup) *pb.PatternGroup {
	if g == nil {
		return nil
	}
	subs := make([]*pb.PatternSubmission, 0, len(g.Submissions))
	for _, ps := range g.Submissions {
		subs = append(subs, &pb.PatternSubmission{
			SubmissionId: ps.SubmissionID,
			StudentId:    ps.StudentID,
			Status:       ps.Status,
		})
	}
	return &pb.PatternGroup{
		PatternId:           g.PatternID,
		Description:         g.Description,
		Count:               int32(g.Count),
		AutoScoreSuggestion: g.AutoScoreSuggestion,
		Submissions:         subs,
	}
}

// GradeStatistics conversions

func gradeStatisticsToProto(gs *repository.GradeStatistics, assignmentID string) *pb.GradeStatistics {
	if gs == nil {
		return nil
	}

	return &pb.GradeStatistics{
		AssignmentId:     assignmentID,
		TotalSubmissions: int32(gs.TotalSubmissions),
		GradedCount:      int32(gs.GradedCount),
		Mean:             gs.Mean,
		Median:           gs.Median,
		StdDeviation:     gs.StdDeviation,
		MinScore:         gs.MinScore,
		MaxScore:         gs.MaxScore,
	}
}
