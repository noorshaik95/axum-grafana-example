package grpc

import (
	"context"
	"fmt"

	pb "slate/services/assignment-grading-service/api/proto"
	"slate/services/assignment-grading-service/internal/service"
)

// GradingServiceServer implements the GradingService gRPC interface
type GradingServiceServer struct {
	pb.UnimplementedGradingServiceServer
	service      service.GradingService
	queueService service.GradingQueueService
}

// NewGradingServiceServer creates a new GradingServiceServer
func NewGradingServiceServer(svc service.GradingService) *GradingServiceServer {
	return &GradingServiceServer{
		service: svc,
	}
}

// NewGradingServiceServerFull wires grading + pattern-queue services so that
// queue / batch-publish RPCs (W9.4) have their dependencies satisfied.
func NewGradingServiceServerFull(svc service.GradingService, queueSvc service.GradingQueueService) *GradingServiceServer {
	return &GradingServiceServer{
		service:      svc,
		queueService: queueSvc,
	}
}

// CreateGrade creates a draft grade with score and feedback
func (s *GradingServiceServer) CreateGrade(ctx context.Context, req *pb.CreateGradeRequest) (*pb.CreateGradeResponse, error) {
	log.WithContext(ctx).
		Str("submission_id", req.SubmissionId).
		Float64("score", req.Score).
		Str("graded_by", req.GradedBy).
		Msg("CreateGrade called")

	// Call service layer (handles adjusted score calculation with late penalties)
	grade, err := s.service.CreateGrade(
		ctx,
		req.SubmissionId,
		req.Score,
		req.Feedback,
		req.GradedBy,
	)

	if err != nil {
		log.ErrorWithContext(ctx).
			Err(err).
			Str("submission_id", req.SubmissionId).
			Msg("Failed to create grade")
		return nil, mapError(err)
	}

	log.WithContext(ctx).
		Str("grade_id", grade.ID).
		Str("submission_id", req.SubmissionId).
		Float64("score", grade.Score).
		Float64("adjusted_score", grade.AdjustedScore).
		Msg("Grade created successfully")

	return &pb.CreateGradeResponse{
		Grade: gradeToProto(grade),
	}, nil
}

// UpdateGrade updates a draft grade's score and feedback
func (s *GradingServiceServer) UpdateGrade(ctx context.Context, req *pb.UpdateGradeRequest) (*pb.UpdateGradeResponse, error) {
	log.WithContext(ctx).
		Str("grade_id", req.Id).
		Float64("score", req.Score).
		Msg("UpdateGrade called")

	// Call service layer (checks draft status and recalculates adjusted score)
	grade, err := s.service.UpdateGrade(
		ctx,
		req.Id,
		req.Score,
		req.Feedback,
	)

	if err != nil {
		log.ErrorWithContext(ctx).
			Err(err).
			Str("grade_id", req.Id).
			Msg("Failed to update grade")
		return nil, mapError(err)
	}

	log.WithContext(ctx).
		Str("grade_id", grade.ID).
		Float64("score", grade.Score).
		Float64("adjusted_score", grade.AdjustedScore).
		Msg("Grade updated successfully")

	return &pb.UpdateGradeResponse{
		Grade: gradeToProto(grade),
	}, nil
}

// PublishGrade publishes a grade to make it visible to students
func (s *GradingServiceServer) PublishGrade(ctx context.Context, req *pb.PublishGradeRequest) (*pb.PublishGradeResponse, error) {
	log.WithContext(ctx).
		Str("grade_id", req.Id).
		Msg("PublishGrade called")

	// Call service layer (updates status and published_at timestamp)
	grade, err := s.service.PublishGrade(ctx, req.Id)
	if err != nil {
		log.ErrorWithContext(ctx).
			Err(err).
			Str("grade_id", req.Id).
			Msg("Failed to publish grade")
		return nil, mapError(err)
	}

	log.WithContext(ctx).
		Str("grade_id", grade.ID).
		Str("status", grade.Status).
		Msg("Grade published successfully")

	return &pb.PublishGradeResponse{
		Grade: gradeToProto(grade),
	}, nil
}

// GetGrade retrieves a grade by ID
func (s *GradingServiceServer) GetGrade(ctx context.Context, req *pb.GetGradeRequest) (*pb.GetGradeResponse, error) {
	log.WithContext(ctx).
		Str("grade_id", req.Id).
		Msg("GetGrade called")

	grade, err := s.service.GetGrade(ctx, req.Id)
	if err != nil {
		log.ErrorWithContext(ctx).
			Err(err).
			Str("grade_id", req.Id).
			Msg("Failed to get grade")
		return nil, mapError(err)
	}

	log.WithContext(ctx).
		Str("grade_id", grade.ID).
		Msg("Grade retrieved successfully")

	return &pb.GetGradeResponse{
		Grade: gradeToProto(grade),
	}, nil
}

// GetGradingQueue returns pattern-grouped submissions awaiting grading (W9.4).
func (s *GradingServiceServer) GetGradingQueue(ctx context.Context, req *pb.GetGradingQueueRequest) (*pb.GetGradingQueueResponse, error) {
	log.WithContext(ctx).
		Str("assignment_id", req.AssignmentId).
		Str("instructor_id", req.InstructorId).
		Msg("GetGradingQueue called")

	if s.queueService == nil {
		return nil, mapError(fmt.Errorf("queue service not configured"))
	}

	groups, err := s.queueService.GetGradingQueue(ctx, req.AssignmentId, req.InstructorId)
	if err != nil {
		log.ErrorWithContext(ctx).Err(err).Str("assignment_id", req.AssignmentId).Msg("Failed to get grading queue")
		return nil, mapError(err)
	}

	protoGroups := make([]*pb.PatternGroup, 0, len(groups))
	for i := range groups {
		protoGroups = append(protoGroups, patternGroupToProto(&groups[i]))
	}

	return &pb.GetGradingQueueResponse{Groups: protoGroups}, nil
}

// GetGradingQueueCount returns the total number of submissions awaiting grading
// across all pattern groups for an assignment.
func (s *GradingServiceServer) GetGradingQueueCount(ctx context.Context, req *pb.GetGradingQueueCountRequest) (*pb.GetGradingQueueCountResponse, error) {
	log.WithContext(ctx).
		Str("assignment_id", req.AssignmentId).
		Str("instructor_id", req.InstructorId).
		Msg("GetGradingQueueCount called")

	if s.queueService == nil {
		return nil, mapError(fmt.Errorf("queue service not configured"))
	}

	groups, err := s.queueService.GetGradingQueue(ctx, req.AssignmentId, req.InstructorId)
	if err != nil {
		log.ErrorWithContext(ctx).Err(err).Str("assignment_id", req.AssignmentId).Msg("Failed to count grading queue")
		return nil, mapError(err)
	}

	total := int32(0)
	for _, g := range groups {
		total += int32(g.Count)
	}
	return &pb.GetGradingQueueCountResponse{Count: total}, nil
}

// BatchPublishGrades publishes a set of draft grades in one call.
func (s *GradingServiceServer) BatchPublishGrades(ctx context.Context, req *pb.BatchPublishGradesRequest) (*pb.BatchPublishGradesResponse, error) {
	log.WithContext(ctx).
		Int("grade_ids", len(req.GradeIds)).
		Msg("BatchPublishGrades called")

	var published int32
	failed := make([]string, 0)
	for _, id := range req.GradeIds {
		if _, err := s.service.PublishGrade(ctx, id); err != nil {
			log.ErrorWithContext(ctx).Err(err).Str("grade_id", id).Msg("BatchPublishGrades: publish failed")
			failed = append(failed, id)
			continue
		}
		published++
	}
	return &pb.BatchPublishGradesResponse{
		PublishedCount: published,
		FailedIds:      failed,
	}, nil
}
