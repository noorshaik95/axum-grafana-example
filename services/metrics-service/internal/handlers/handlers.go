package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"

	"slate/services/metrics-service/internal/analytics"
	"slate/services/metrics-service/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	repo *repository.Repository
}

// New creates a new Handler.
func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes registers all metrics routes on the given Chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/metrics/platform", h.GetPlatformMetrics)
	r.Get("/metrics/tenants/{id}", h.GetTenantMetrics)
	r.Get("/metrics/courses/{id}", h.GetCourseEngagement)
	r.Get("/metrics/students/{id}", h.GetStudentProgress)
	r.Get("/metrics/students/{id}/time-on-task", h.GetStudentTimeOnTask)
	r.Get("/metrics/grades/distribution/{courseId}", h.GetGradeDistribution)
	r.Get("/metrics/grades/comparison/{studentId}/{courseId}", h.GetStudentComparison)
}

func (h *Handler) GetPlatformMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.repo.GetPlatformMetrics(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get platform metrics")
		writeError(w, http.StatusInternalServerError, "failed to get platform metrics")
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (h *Handler) GetTenantMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "id")
	metrics, err := h.repo.GetTenantMetrics(r.Context(), tenantID)
	if err != nil {
		log.Error().Err(err).Str("tenantId", tenantID).Msg("failed to get tenant metrics")
		writeError(w, http.StatusInternalServerError, "failed to get tenant metrics")
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (h *Handler) GetCourseEngagement(w http.ResponseWriter, r *http.Request) {
	courseID := chi.URLParam(r, "id")
	metrics, err := h.repo.GetCourseEngagement(r.Context(), courseID)
	if err != nil {
		log.Error().Err(err).Str("courseId", courseID).Msg("failed to get course engagement")
		writeError(w, http.StatusInternalServerError, "failed to get course engagement")
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (h *Handler) GetStudentProgress(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "id")

	// Resolve tenant from student data
	tenantID, err := h.repo.GetTenantIDForStudent(r.Context(), studentID)
	if err != nil {
		// Try header as fallback
		tenantID = r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "tenant not found for student; provide X-Tenant-ID header")
			return
		}
	}

	progress, err := h.repo.GetStudentProgress(r.Context(), tenantID, studentID)
	if err != nil {
		log.Error().Err(err).Str("studentId", studentID).Msg("failed to get student progress")
		writeError(w, http.StatusInternalServerError, "failed to get student progress")
		return
	}
	if progress == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	writeJSON(w, http.StatusOK, progress)
}

func (h *Handler) GetStudentTimeOnTask(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "id")

	tenantID, err := h.repo.GetTenantIDForStudent(r.Context(), studentID)
	if err != nil {
		tenantID = r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "tenant not found for student; provide X-Tenant-ID header")
			return
		}
	}

	timeOnTask, err := h.repo.GetStudentTimeOnTask(r.Context(), tenantID, studentID)
	if err != nil {
		log.Error().Err(err).Str("studentId", studentID).Msg("failed to get time on task")
		writeError(w, http.StatusInternalServerError, "failed to get time on task")
		return
	}
	if timeOnTask == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}
	writeJSON(w, http.StatusOK, timeOnTask)
}

func (h *Handler) GetGradeDistribution(w http.ResponseWriter, r *http.Request) {
	courseID := chi.URLParam(r, "courseId")

	tenantID, err := h.repo.GetTenantIDForCourse(r.Context(), courseID)
	if err != nil {
		tenantID = r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "tenant not found for course; provide X-Tenant-ID header")
			return
		}
	}

	scores, err := h.repo.GetScoresForCourse(r.Context(), tenantID, courseID)
	if err != nil && err != sql.ErrNoRows {
		log.Error().Err(err).Str("courseId", courseID).Msg("failed to get scores for distribution")
		writeError(w, http.StatusInternalServerError, "failed to compute grade distribution")
		return
	}

	dist := analytics.ComputeDistribution(courseID, scores)
	writeJSON(w, http.StatusOK, dist)
}

func (h *Handler) GetStudentComparison(w http.ResponseWriter, r *http.Request) {
	studentID := chi.URLParam(r, "studentId")
	courseID := chi.URLParam(r, "courseId")

	tenantID, err := h.repo.GetTenantIDForCourse(r.Context(), courseID)
	if err != nil {
		tenantID = r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			writeError(w, http.StatusBadRequest, "tenant not found for course; provide X-Tenant-ID header")
			return
		}
	}

	scores, err := h.repo.GetScoresForCourse(r.Context(), tenantID, courseID)
	if err != nil && err != sql.ErrNoRows {
		log.Error().Err(err).Str("courseId", courseID).Msg("failed to get scores")
		writeError(w, http.StatusInternalServerError, "failed to compute comparison")
		return
	}

	dist := analytics.ComputeDistribution(courseID, scores)

	studentScore, err := h.repo.GetStudentScoreForCourse(r.Context(), tenantID, studentID, courseID)
	if err != nil && err != sql.ErrNoRows {
		log.Error().Err(err).Str("studentId", studentID).Msg("failed to get student score")
		writeError(w, http.StatusInternalServerError, "failed to get student score")
		return
	}

	sort.Float64s(scores)
	comparison := struct {
		Distribution      interface{} `json:"distribution"`
		StudentScore      float64     `json:"studentScore"`
		StudentPercentile float64     `json:"studentPercentile"`
		LetterGrade       string      `json:"letterGrade"`
	}{
		Distribution:      dist,
		StudentScore:      studentScore,
		StudentPercentile: analytics.ComputeStudentPercentile(scores, studentScore),
		LetterGrade:       analytics.LetterGrade(studentScore),
	}

	writeJSON(w, http.StatusOK, comparison)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func decodeBodyJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
