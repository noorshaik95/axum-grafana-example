package grpc

import (
	"context"
	pb "slate/services/assignment-grading-service/api/proto"
	"slate/services/assignment-grading-service/internal/service"

	"slate/libs/common-go/tracing"

	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
)

// SubmissionServiceServer implements the SubmissionService gRPC interface
type SubmissionServiceServer struct {
	pb.UnimplementedSubmissionServiceServer
	service service.SubmissionService
}

// NewSubmissionServiceServer creates a new SubmissionServiceServer
func NewSubmissionServiceServer(svc service.SubmissionService) *SubmissionServiceServer {
	return &SubmissionServiceServer{
		service: svc,
	}
}

// SubmitAssignment accepts a student submission with file content
func (s *SubmissionServiceServer) SubmitAssignment(ctx context.Context, req *pb.SubmitAssignmentRequest) (*pb.SubmitAssignmentResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "submit_assignment_handler",
		attribute.String("assignment_id", req.AssignmentId),
		attribute.String("student_id", req.StudentId))
	defer span.End()

	log.WithContext(ctx).
		Str("assignment_id", req.AssignmentId).
		Str("student_id", req.StudentId).
		Str("file_name", req.FileName).
		Msg("SubmitAssignment called")

	// Call service layer with file content
	submission, err := s.service.SubmitAssignment(
		ctx,
		req.AssignmentId,
		req.StudentId,
		req.FileContent,
		req.FileName,
		req.ContentType,
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "submit assignment failed")
		log.ErrorWithContext(ctx).
			Err(err).
			Str("assignment_id", req.AssignmentId).
			Str("student_id", req.StudentId).
			Msg("Failed to submit assignment")
		return nil, mapError(err)
	}

	log.WithContext(ctx).
		Str("submission_id", submission.ID).
		Str("assignment_id", req.AssignmentId).
		Str("student_id", req.StudentId).
		Bool("is_late", submission.IsLate).
		Msg("Assignment submitted successfully")

	span.SetStatus(otelcodes.Ok, "")
	return &pb.SubmitAssignmentResponse{
		Submission: submissionToProto(submission),
	}, nil
}

// GetSubmission retrieves a submission by ID
func (s *SubmissionServiceServer) GetSubmission(ctx context.Context, req *pb.GetSubmissionRequest) (*pb.GetSubmissionResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "get_submission_handler",
		attribute.String("submission_id", req.Id))
	defer span.End()

	log.WithContext(ctx).
		Str("submission_id", req.Id).
		Msg("GetSubmission called")

	submission, err := s.service.GetSubmission(ctx, req.Id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "get submission failed")
		log.ErrorWithContext(ctx).
			Err(err).
			Str("submission_id", req.Id).
			Msg("Failed to get submission")
		return nil, mapError(err)
	}

	log.WithContext(ctx).
		Str("submission_id", submission.ID).
		Msg("Submission retrieved successfully")

	span.SetStatus(otelcodes.Ok, "")
	return &pb.GetSubmissionResponse{
		Submission: submissionToProto(submission),
	}, nil
}

// ListSubmissions lists all submissions for an assignment with sorting
func (s *SubmissionServiceServer) ListSubmissions(ctx context.Context, req *pb.ListSubmissionsRequest) (*pb.ListSubmissionsResponse, error) {
	ctx, span := tracing.StartSpan(ctx, "list_submissions_handler",
		attribute.String("assignment_id", req.AssignmentId))
	defer span.End()

	log.WithContext(ctx).
		Str("assignment_id", req.AssignmentId).
		Str("sort_by", req.SortBy).
		Str("order", req.Order).
		Msg("ListSubmissions called")

	submissions, err := s.service.ListSubmissions(
		ctx,
		req.AssignmentId,
		req.SortBy,
		req.Order,
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, "list submissions failed")
		log.ErrorWithContext(ctx).
			Err(err).
			Str("assignment_id", req.AssignmentId).
			Msg("Failed to list submissions")
		return nil, mapError(err)
	}

	// Convert submissions to proto
	protoSubmissions := make([]*pb.Submission, 0, len(submissions))
	for _, submission := range submissions {
		protoSubmissions = append(protoSubmissions, submissionToProto(submission))
	}

	log.WithContext(ctx).
		Str("assignment_id", req.AssignmentId).
		Int("count", len(submissions)).
		Msg("Submissions listed successfully")

	span.SetStatus(otelcodes.Ok, "")
	return &pb.ListSubmissionsResponse{
		Submissions: protoSubmissions,
	}, nil
}

// ListStudentSubmissions lists all submissions by a student for a course
func (s *SubmissionServiceServer) ListStudentSubmissions(ctx context.Context, req *pb.ListStudentSubmissionsRequest) (*pb.ListStudentSubmissionsResponse, error) {
	log.WithContext(ctx).
		Str("student_id", req.StudentId).
		Str("course_id", req.CourseId).
		Msg("ListStudentSubmissions called")

	submissions, err := s.service.ListStudentSubmissions(
		ctx,
		req.StudentId,
		req.CourseId,
	)

	if err != nil {
		log.ErrorWithContext(ctx).
			Err(err).
			Str("student_id", req.StudentId).
			Str("course_id", req.CourseId).
			Msg("Failed to list student submissions")
		return nil, mapError(err)
	}

	// Convert submissions to proto
	protoSubmissions := make([]*pb.Submission, 0, len(submissions))
	for _, submission := range submissions {
		protoSubmissions = append(protoSubmissions, submissionToProto(submission))
	}

	log.WithContext(ctx).
		Str("student_id", req.StudentId).
		Str("course_id", req.CourseId).
		Int("count", len(submissions)).
		Msg("Student submissions listed successfully")

	return &pb.ListStudentSubmissionsResponse{
		Submissions: protoSubmissions,
	}, nil
}

// SaveDraft stores or replaces a student's draft submission (W9.2).
func (s *SubmissionServiceServer) SaveDraft(ctx context.Context, req *pb.SaveDraftRequest) (*pb.SaveDraftResponse, error) {
	log.WithContext(ctx).
		Str("assignment_id", req.AssignmentId).
		Str("student_id", req.StudentId).
		Int("file_urls", len(req.FileUrls)).
		Msg("SaveDraft called")

	draft, err := s.service.UpsertDraft(ctx, req.TenantId, req.AssignmentId, req.StudentId, req.FileUrls)
	if err != nil {
		log.ErrorWithContext(ctx).Err(err).Str("assignment_id", req.AssignmentId).Msg("Failed to save draft")
		return nil, mapError(err)
	}

	return &pb.SaveDraftResponse{
		Submission: submissionToProto(draft),
	}, nil
}

// GetDraft fetches the current draft submission for (assignment, student).
func (s *SubmissionServiceServer) GetDraft(ctx context.Context, req *pb.GetDraftRequest) (*pb.GetDraftResponse, error) {
	log.WithContext(ctx).
		Str("assignment_id", req.AssignmentId).
		Str("student_id", req.StudentId).
		Msg("GetDraft called")

	draft, err := s.service.GetDraft(ctx, req.AssignmentId, req.StudentId)
	if err != nil {
		log.ErrorWithContext(ctx).Err(err).Str("assignment_id", req.AssignmentId).Msg("Failed to get draft")
		return nil, mapError(err)
	}

	return &pb.GetDraftResponse{
		Submission: submissionToProto(draft),
	}, nil
}
