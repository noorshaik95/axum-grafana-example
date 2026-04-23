package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	aipb "slate/services/ai-service/api/proto"
	"slate/services/ai-service/internal/budget"
	"slate/services/ai-service/internal/service"
)

// Server wires the gRPC surface defined in proto/ai.proto to the concrete
// service handlers.
type Server struct {
	aipb.UnimplementedAiServiceServer
	Welcomer        *service.Welcomer
	Grades          *service.GradeProjector
	StudyPlan       *service.StudyPlanner
	Palette         *service.CmdPalette
	Feedback        *service.DraftFeedback
}

func (s *Server) GetWelcomeMessage(ctx context.Context, req *aipb.WelcomeMessageRequest) (*aipb.WelcomeMessageResponse, error) {
	if s.Welcomer == nil {
		return nil, status.Error(codes.Unimplemented, "welcome not configured")
	}
	resp, err := s.Welcomer.Welcome(ctx, req)
	return resp, mapError(err)
}

func (s *Server) GetGradeProjection(ctx context.Context, req *aipb.GradeProjectionRequest) (*aipb.GradeProjectionResponse, error) {
	if s.Grades == nil {
		return nil, status.Error(codes.Unimplemented, "grades not configured")
	}
	resp, err := s.Grades.Project(ctx, req)
	return resp, mapError(err)
}

func (s *Server) GenerateStudyPlan(ctx context.Context, req *aipb.StudyPlanRequest) (*aipb.StudyPlanResponse, error) {
	if s.StudyPlan == nil {
		return nil, status.Error(codes.Unimplemented, "study plan not configured")
	}
	resp, err := s.StudyPlan.Generate(ctx, req)
	return resp, mapError(err)
}

func (s *Server) GetCommandPaletteResults(ctx context.Context, req *aipb.CmdPaletteRequest) (*aipb.CmdPaletteResponse, error) {
	if s.Palette == nil {
		return nil, status.Error(codes.Unimplemented, "palette not configured")
	}
	resp, err := s.Palette.Resolve(ctx, req)
	return resp, mapError(err)
}

func (s *Server) GetDraftFeedback(ctx context.Context, req *aipb.DraftFeedbackRequest) (*aipb.DraftFeedbackResponse, error) {
	if s.Feedback == nil {
		return nil, status.Error(codes.Unimplemented, "feedback not configured")
	}
	resp, err := s.Feedback.Evaluate(ctx, req)
	if errors.Is(err, service.ErrFlagDisabled) {
		return nil, status.Error(codes.FailedPrecondition, "flag_disabled")
	}
	return resp, mapError(err)
}

// mapError converts package-level sentinels into gRPC status codes.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, budget.ErrBudgetExceeded) {
		return status.Error(codes.ResourceExhausted, err.Error())
	}
	return status.Error(codes.Internal, err.Error())
}
