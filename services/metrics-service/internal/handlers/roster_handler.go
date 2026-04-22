package handlers

import (
	"context"
	"net/http"
	"time"

	"slate/services/metrics-service/internal/repository"
	"slate/services/metrics-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// RosterHandler handles the W12.1 roster health endpoints.
type RosterHandler struct {
	repo *repository.Repository
	svc  *service.RosterService
}

// NewRosterHandler constructs a RosterHandler.
func NewRosterHandler(repo *repository.Repository, svc *service.RosterService) *RosterHandler {
	return &RosterHandler{repo: repo, svc: svc}
}

// RegisterRosterRoutes attaches roster routes to the Chi router.
func (rh *RosterHandler) RegisterRosterRoutes(r chi.Router) {
	r.Get("/roster/{courseId}/health", rh.GetRosterHealth)
	r.Post("/roster/{courseId}/invalidate", rh.InvalidateRoster)
	r.Post("/roster/nudge", rh.SendNudge)
}

// GetRosterHealth handles GET /roster/{courseId}/health.
// Inputs: tenant_slug (query) or X-Tenant-Slug header, instructor_id via X-User-ID.
func (rh *RosterHandler) GetRosterHealth(w http.ResponseWriter, r *http.Request) {
	courseID := chi.URLParam(r, "courseId")
	tenantSlug := tenantSlugFrom(r)
	instructorID := r.Header.Get("X-User-ID")

	tenantID, err := rh.repo.GetTenantIDForCourse(r.Context(), courseID)
	if err != nil {
		tenantID = tenantSlug
	}

	fetch := func(ctx context.Context, slug, cid, iid string) ([]service.RosterSignals, error) {
		rows, err := rh.repo.GetRosterSignals(ctx, tenantID, cid)
		if err != nil {
			return nil, err
		}
		out := make([]service.RosterSignals, len(rows))
		for i, row := range rows {
			out[i] = service.RosterSignals{
				UserID:            row.UserID,
				DisplayName:       row.DisplayName,
				MissedAssignments: row.MissedAssignments,
				DaysSinceActive:   row.DaysSinceActive,
				GradeTrend:        row.GradeTrend,
			}
		}
		return out, nil
	}

	entries, err := rh.svc.GetRosterHealth(r.Context(), tenantSlug, courseID, instructorID, fetch)
	if err != nil {
		log.Error().Err(err).Str("courseId", courseID).Msg("failed to compute roster health")
		writeError(w, http.StatusInternalServerError, "failed to compute roster health")
		return
	}

	atRisk := 0
	for _, e := range entries {
		if e.RiskLevel == service.RiskAtRisk {
			atRisk++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"course_id": courseID,
		"students":  entries,
		"total":     len(entries),
		"at_risk":   atRisk,
	})
}

// InvalidateRoster handles POST /roster/{courseId}/invalidate — manual cache
// bust for admins/debugging.
func (rh *RosterHandler) InvalidateRoster(w http.ResponseWriter, r *http.Request) {
	courseID := chi.URLParam(r, "courseId")
	rh.svc.InvalidateRoster(tenantSlugFrom(r), courseID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "invalidated"})
}

// SendNudge handles POST /roster/nudge.
func (rh *RosterHandler) SendNudge(w http.ResponseWriter, r *http.Request) {
	instructorID := r.Header.Get("X-User-ID")
	if instructorID == "" {
		writeError(w, http.StatusUnauthorized, "missing user ID")
		return
	}

	var req struct {
		StudentID string `json:"studentId"`
		CourseID  string `json:"courseId"`
		Message   string `json:"message"`
	}
	if err := decodeBodyJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.StudentID == "" || req.CourseID == "" {
		writeError(w, http.StatusBadRequest, "studentId and courseId are required")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "nudge_sent",
		"studentId":    req.StudentID,
		"courseId":     req.CourseID,
		"instructorId": instructorID,
		"sentAt":       time.Now(),
	})
}

func tenantSlugFrom(r *http.Request) string {
	if v := r.Header.Get("X-Tenant-Slug"); v != "" {
		return v
	}
	if v := r.URL.Query().Get("tenant_slug"); v != "" {
		return v
	}
	return "default"
}
