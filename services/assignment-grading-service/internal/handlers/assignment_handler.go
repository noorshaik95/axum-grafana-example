package handlers

import (
	"net/http"
	"time"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/service"
)

// AssignmentHandler handles assignment REST endpoints
type AssignmentHandler struct {
	svc service.AssignmentService
}

// NewAssignmentHandler creates a new AssignmentHandler
func NewAssignmentHandler(svc service.AssignmentService) *AssignmentHandler {
	return &AssignmentHandler{svc: svc}
}

type createAssignmentRequest struct {
	TenantID         string                   `json:"tenant_id"`
	CourseID         string                   `json:"course_id"`
	InstructorID     string                   `json:"instructor_id"`
	Title            string                   `json:"title"`
	Description      string                   `json:"description"`
	MaxPoints        float64                  `json:"max_points"`
	Rubric           []models.RubricCriterion `json:"rubric"`
	AssignmentType   string                   `json:"assignment_type"`
	DueDate          time.Time                `json:"due_date"`
	MaxFileSizeMB    int                      `json:"max_file_size_mb"`
	AllowedFileTypes []string                 `json:"allowed_file_types"`
	LatePolicy       models.LatePolicy        `json:"late_policy"`
}

// Create handles POST /assignments
func (h *AssignmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createAssignmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	assignment := &models.Assignment{
		TenantID:         req.TenantID,
		CourseID:         req.CourseID,
		InstructorID:     req.InstructorID,
		Title:            req.Title,
		Description:      req.Description,
		MaxPoints:        req.MaxPoints,
		Rubric:           req.Rubric,
		AssignmentType:   req.AssignmentType,
		DueDate:          req.DueDate,
		MaxFileSizeMB:    req.MaxFileSizeMB,
		AllowedFileTypes: req.AllowedFileTypes,
		LatePolicy:       req.LatePolicy,
	}

	result, err := h.svc.CreateAssignmentFull(r.Context(), assignment)
	if err != nil {
		mapServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// Get handles GET /assignments/:id
func (h *AssignmentHandler) Get(w http.ResponseWriter, r *http.Request, id string) {
	assignment, err := h.svc.GetAssignment(r.Context(), id)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, assignment)
}

// Update handles PUT /assignments/:id
func (h *AssignmentHandler) Update(w http.ResponseWriter, r *http.Request, id string) {
	var req createAssignmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	assignment := &models.Assignment{
		ID:               id,
		TenantID:         req.TenantID,
		CourseID:         req.CourseID,
		InstructorID:     req.InstructorID,
		Title:            req.Title,
		Description:      req.Description,
		MaxPoints:        req.MaxPoints,
		Rubric:           req.Rubric,
		AssignmentType:   req.AssignmentType,
		DueDate:          req.DueDate,
		MaxFileSizeMB:    req.MaxFileSizeMB,
		AllowedFileTypes: req.AllowedFileTypes,
		LatePolicy:       req.LatePolicy,
	}

	result, err := h.svc.UpdateAssignmentFull(r.Context(), assignment)
	if err != nil {
		mapServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Delete handles DELETE /assignments/:id (soft delete)
func (h *AssignmentHandler) Delete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.SoftDeleteAssignment(r.Context(), id); err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// List handles GET /assignments
func (h *AssignmentHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenantId")
	courseID := r.URL.Query().Get("courseId")
	instructorID := r.URL.Query().Get("instructorId")
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "pageSize", 20)

	assignments, total, err := h.svc.ListAssignmentsFiltered(r.Context(), tenantID, courseID, instructorID, page, pageSize)
	if err != nil {
		mapServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"assignments": assignments,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
	})
}
