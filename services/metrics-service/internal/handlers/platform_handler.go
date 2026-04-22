package handlers

import (
	"net/http"

	"slate/services/metrics-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// PlatformHandler serves the W12.4 extended platform metrics endpoint.
type PlatformHandler struct {
	svc *service.PlatformService
}

// NewPlatformHandler constructs a PlatformHandler.
func NewPlatformHandler(svc *service.PlatformService) *PlatformHandler {
	return &PlatformHandler{svc: svc}
}

// RegisterPlatformRoutes attaches the /metrics/platform/extended route.
func (h *PlatformHandler) RegisterPlatformRoutes(r chi.Router) {
	r.Get("/metrics/platform/extended", h.GetExtended)
}

// GetExtended returns the extended platform stats.
func (h *PlatformHandler) GetExtended(w http.ResponseWriter, r *http.Request) {
	m, err := h.svc.Get(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to compute extended platform metrics")
		writeError(w, http.StatusInternalServerError, "failed to compute platform metrics")
		return
	}
	writeJSON(w, http.StatusOK, m)
}
