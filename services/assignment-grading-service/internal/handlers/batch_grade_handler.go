package handlers

import (
	"net/http"

	"slate/services/assignment-grading-service/internal/service"
)

// BatchGradeHandler serves POST /grades/batch (W9.5).
type BatchGradeHandler struct {
	svc   service.BatchGradingService
	queue service.GradingQueueService
}

// NewBatchGradeHandler constructs the batch grade handler.
func NewBatchGradeHandler(svc service.BatchGradingService, queue service.GradingQueueService) *BatchGradeHandler {
	return &BatchGradeHandler{svc: svc, queue: queue}
}

// Apply handles POST /grades/batch.
// Body: { pattern_id, assignment_id, submission_ids?, rubric_scores[], feedback_template, graded_by }
// If submission_ids is empty, the handler resolves them from the grading queue.
func (h *BatchGradeHandler) Apply(w http.ResponseWriter, r *http.Request) {
	var req service.BatchGradeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.SubmissionIDs) == 0 && h.queue != nil {
		groups, err := h.queue.GetGradingQueue(r.Context(), req.AssignmentID, req.InstructorID)
		if err != nil {
			mapServiceError(w, err)
			return
		}
		for _, g := range groups {
			if g.PatternID == req.PatternID {
				for _, sub := range g.Submissions {
					req.SubmissionIDs = append(req.SubmissionIDs, sub.SubmissionID)
				}
				break
			}
		}
	}
	result, err := h.svc.Apply(r.Context(), req)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

