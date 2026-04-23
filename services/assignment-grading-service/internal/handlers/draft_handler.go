package handlers

import (
	"net/http"

	"slate/services/assignment-grading-service/internal/service"
)

// DraftHandler serves the draft-submission endpoints (W9.2).
type DraftHandler struct {
	svc service.SubmissionService
}

// NewDraftHandler constructs a draft-submission handler.
func NewDraftHandler(svc service.SubmissionService) *DraftHandler {
	return &DraftHandler{svc: svc}
}

type draftRequest struct {
	TenantID  string   `json:"tenant_id"`
	StudentID string   `json:"student_id"`
	FileURLs  []string `json:"file_urls"`
}

// Upsert handles PATCH /assignments/:id/submissions/draft.
func (h *DraftHandler) Upsert(w http.ResponseWriter, r *http.Request, assignmentID string) {
	var req draftRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.StudentID == "" {
		writeError(w, http.StatusBadRequest, "student_id is required")
		return
	}
	if len(req.FileURLs) == 0 {
		writeError(w, http.StatusBadRequest, "file_urls is required")
		return
	}
	draft, err := h.svc.UpsertDraft(r.Context(), req.TenantID, assignmentID, req.StudentID, req.FileURLs)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

// Get handles GET /assignments/:id/submissions/draft?student_id=...
func (h *DraftHandler) Get(w http.ResponseWriter, r *http.Request, assignmentID string) {
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		writeError(w, http.StatusBadRequest, "student_id is required")
		return
	}
	draft, err := h.svc.GetDraft(r.Context(), assignmentID, studentID)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, draft)
}
