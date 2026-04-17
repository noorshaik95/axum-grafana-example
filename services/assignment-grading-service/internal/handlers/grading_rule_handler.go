package handlers

import (
	"net/http"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/service"
)

// GradingRuleHandler handles grading rule REST endpoints
type GradingRuleHandler struct {
	svc service.GradingRuleService
}

// NewGradingRuleHandler creates a new GradingRuleHandler
func NewGradingRuleHandler(svc service.GradingRuleService) *GradingRuleHandler {
	return &GradingRuleHandler{svc: svc}
}

type gradingRuleRequest struct {
	TenantID          string              `json:"tenant_id"`
	CourseID          string              `json:"course_id"`
	AssignmentType    string              `json:"assignment_type"`
	Weight            float64             `json:"weight"`
	LatePenaltyPerDay float64             `json:"late_penalty_per_day"`
	MaxLatePenalty    float64             `json:"max_late_penalty"`
	GradeScale        []models.GradeScale `json:"grade_scale"`
}

// Create handles POST /grading-rules
func (h *GradingRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req gradingRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rule := &models.GradingRule{
		TenantID:          req.TenantID,
		CourseID:          req.CourseID,
		AssignmentType:    req.AssignmentType,
		Weight:            req.Weight,
		LatePenaltyPerDay: req.LatePenaltyPerDay,
		MaxLatePenalty:    req.MaxLatePenalty,
		GradeScale:        req.GradeScale,
	}

	result, err := h.svc.CreateRule(r.Context(), rule)
	if err != nil {
		mapServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// List handles GET /grading-rules
func (h *GradingRuleHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenantId")
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenantId query parameter is required")
		return
	}

	rules, err := h.svc.ListRules(r.Context(), tenantID)
	if err != nil {
		mapServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rules": rules,
	})
}

// Update handles PUT /grading-rules/:id
func (h *GradingRuleHandler) Update(w http.ResponseWriter, r *http.Request, id string) {
	var req gradingRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rule := &models.GradingRule{
		ID:                id,
		TenantID:          req.TenantID,
		CourseID:          req.CourseID,
		AssignmentType:    req.AssignmentType,
		Weight:            req.Weight,
		LatePenaltyPerDay: req.LatePenaltyPerDay,
		MaxLatePenalty:    req.MaxLatePenalty,
		GradeScale:        req.GradeScale,
	}

	result, err := h.svc.UpdateRule(r.Context(), rule)
	if err != nil {
		mapServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Delete handles DELETE /grading-rules/:id
func (h *GradingRuleHandler) Delete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.DeleteRule(r.Context(), id); err != nil {
		mapServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
