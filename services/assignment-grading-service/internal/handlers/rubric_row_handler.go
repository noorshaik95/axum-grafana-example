package handlers

import (
	"net/http"
	"strings"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/service"
)

// RubricRowHandler serves rubric-row REST endpoints (W9.1).
type RubricRowHandler struct {
	svc service.RubricRowService
}

// NewRubricRowHandler constructs a rubric-row HTTP handler.
func NewRubricRowHandler(svc service.RubricRowService) *RubricRowHandler {
	return &RubricRowHandler{svc: svc}
}

type rubricRowRequest struct {
	AssignmentID string  `json:"assignment_id"`
	Title        string  `json:"title"`
	MaxPoints    float64 `json:"max_points"`
	SortOrder    int     `json:"sort_order"`
}

// Create handles POST /assignments/:id/rubric-rows.
func (h *RubricRowHandler) Create(w http.ResponseWriter, r *http.Request, assignmentID string) {
	var req rubricRowRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	row := &models.RubricRow{
		AssignmentID: assignmentID,
		Title:        strings.TrimSpace(req.Title),
		MaxPoints:    req.MaxPoints,
		SortOrder:    req.SortOrder,
	}
	result, err := h.svc.Create(r.Context(), row)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// List handles GET /assignments/:id/rubric-rows.
func (h *RubricRowHandler) List(w http.ResponseWriter, r *http.Request, assignmentID string) {
	rows, err := h.svc.List(r.Context(), assignmentID)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"rubric_rows": rows})
}

// Update handles PUT /rubric-rows/:id. Body must include assignment_id.
func (h *RubricRowHandler) Update(w http.ResponseWriter, r *http.Request, id string) {
	var req rubricRowRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	row := &models.RubricRow{
		ID:           id,
		AssignmentID: req.AssignmentID,
		Title:        strings.TrimSpace(req.Title),
		MaxPoints:    req.MaxPoints,
		SortOrder:    req.SortOrder,
	}
	result, err := h.svc.Update(r.Context(), row)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// Delete handles DELETE /rubric-rows/:id.
func (h *RubricRowHandler) Delete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.Delete(r.Context(), id); err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
