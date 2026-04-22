package handlers

import (
	"net/http"

	"slate/services/metrics-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// ExportHandler owns the W12.3 export endpoints.
type ExportHandler struct {
	svc *service.ExportService
}

// NewExportHandler constructs an ExportHandler.
func NewExportHandler(svc *service.ExportService) *ExportHandler {
	return &ExportHandler{svc: svc}
}

// RegisterExportRoutes attaches export routes.
func (h *ExportHandler) RegisterExportRoutes(r chi.Router) {
	r.Post("/api/metrics/export", h.CreateExport)
	r.Get("/api/metrics/export/{jobId}", h.GetExport)
}

// CreateExport accepts POST /api/metrics/export.
func (h *ExportHandler) CreateExport(w http.ResponseWriter, r *http.Request) {
	var req service.ExportRequest
	if err := decodeBodyJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	job, err := h.svc.Enqueue(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	log.Info().Str("jobId", job.ID).Str("type", req.Type).Msg("export enqueued")
	writeJSON(w, http.StatusAccepted, job)
}

// GetExport returns the current state of an export job.
func (h *ExportHandler) GetExport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "jobId")
	job := h.svc.Get(id)
	if job == nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}
