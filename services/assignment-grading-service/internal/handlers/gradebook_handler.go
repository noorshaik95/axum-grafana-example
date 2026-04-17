package handlers

import (
	"net/http"

	"slate/services/assignment-grading-service/internal/service"
)

// GradebookHandler handles gradebook REST endpoints
type GradebookHandler struct {
	svc service.GradebookService
}

// NewGradebookHandler creates a new GradebookHandler
func NewGradebookHandler(svc service.GradebookService) *GradebookHandler {
	return &GradebookHandler{svc: svc}
}

// GetCourseGrades handles GET /courses/:id/grades
func (h *GradebookHandler) GetCourseGrades(w http.ResponseWriter, r *http.Request, courseID string) {
	gradebook, err := h.svc.GetCourseGradebook(r.Context(), courseID)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gradebook)
}

// GetDistribution handles GET /courses/:id/grades/distribution
func (h *GradebookHandler) GetDistribution(w http.ResponseWriter, r *http.Request, courseID string) {
	distribution, err := h.svc.GetGradeDistribution(r.Context(), courseID)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"course_id":    courseID,
		"distribution": distribution,
	})
}

// GetStudentGrades handles GET /students/:id/grades
func (h *GradebookHandler) GetStudentGrades(w http.ResponseWriter, r *http.Request, studentID string) {
	tenantID := r.URL.Query().Get("tenantId")

	if tenantID != "" {
		grades, err := h.svc.GetStudentGradesByTenant(r.Context(), tenantID, studentID)
		if err != nil {
			mapServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"student_id": studentID,
			"grades":     grades,
		})
		return
	}

	courseID := r.URL.Query().Get("courseId")
	if courseID == "" {
		writeError(w, http.StatusBadRequest, "tenantId or courseId query parameter is required")
		return
	}

	gradebook, err := h.svc.GetStudentGradebook(r.Context(), studentID, courseID)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gradebook)
}

// GetGradeEstimate handles GET /students/:id/grade-estimate
func (h *GradebookHandler) GetGradeEstimate(w http.ResponseWriter, r *http.Request, studentID string) {
	courseID := r.URL.Query().Get("courseId")
	tenantID := r.URL.Query().Get("tenantId")

	if courseID == "" || tenantID == "" {
		writeError(w, http.StatusBadRequest, "courseId and tenantId query parameters are required")
		return
	}

	estimate, err := h.svc.EstimateFinalGrade(r.Context(), studentID, courseID, tenantID)
	if err != nil {
		mapServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, estimate)
}
