package handlers

import (
	"net/http"

	"slate/services/assignment-grading-service/internal/service"
)

// GradingHandler handles grading REST endpoints
type GradingHandler struct {
	svc service.GradingService
}

// NewGradingHandler creates a new GradingHandler
func NewGradingHandler(svc service.GradingService) *GradingHandler {
	return &GradingHandler{svc: svc}
}

type gradeRequest struct {
	Score        float64                `json:"score"`
	Feedback     string                 `json:"feedback"`
	GradedBy     string                 `json:"graded_by"`
	RubricScores map[string]interface{} `json:"rubric_scores"`
	CourseID     string                 `json:"course_id"`
	TenantID     string                 `json:"tenant_id"`
}

// GradeSubmission handles POST /submissions/:id/grade
func (h *GradingHandler) GradeSubmission(w http.ResponseWriter, r *http.Request, submissionID string) {
	var req gradeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.GradedBy == "" {
		writeError(w, http.StatusBadRequest, "graded_by is required")
		return
	}

	grade, err := h.svc.CreateGradeFull(
		r.Context(), submissionID, req.Score, req.Feedback, req.GradedBy,
		req.RubricScores, req.CourseID, req.TenantID,
	)
	if err != nil {
		mapServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, grade)
}

// AutoGrade handles POST /assignments/:id/auto-grade
func (h *GradingHandler) AutoGrade(w http.ResponseWriter, r *http.Request, assignmentID string) {
	if err := h.svc.AutoGrade(r.Context(), assignmentID); err != nil {
		mapServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"assignment_id": assignmentID,
		"message":       "auto-grading completed",
	})
}
