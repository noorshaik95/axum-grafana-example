package handlers

import (
	"net/http"
	"time"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/service"
)

// AttachmentHandler serves per-assignment file endpoints (W9.6).
type AttachmentHandler struct {
	svc service.AttachmentService
}

// NewAttachmentHandler constructs an attachment HTTP handler.
func NewAttachmentHandler(svc service.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{svc: svc}
}

type attachmentCreateRequest struct {
	FilePath    string `json:"file_path"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Kind        string `json:"kind"`
}

// Create handles POST /assignments/:id/attachments.
func (h *AttachmentHandler) Create(w http.ResponseWriter, r *http.Request, assignmentID string) {
	var req attachmentCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	a := &models.AssignmentAttachment{
		AssignmentID: assignmentID,
		FilePath:     req.FilePath,
		FileName:     req.FileName,
		ContentType:  req.ContentType,
		SizeBytes:    req.SizeBytes,
		Kind:         req.Kind,
	}
	result, err := h.svc.Create(r.Context(), a)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// List handles GET /assignments/:id/attachments — returns signed URLs.
func (h *AttachmentHandler) List(w http.ResponseWriter, r *http.Request, assignmentID string) {
	attachments, err := h.svc.ListWithSignedURLs(r.Context(), assignmentID, 15*time.Minute)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"attachments": attachments})
}

// Delete handles DELETE /attachments/:id.
func (h *AttachmentHandler) Delete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.Delete(r.Context(), id); err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
