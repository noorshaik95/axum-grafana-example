package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// FeatureFlag represents a feature flag configuration.
type FeatureFlag struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	Scope       string    `json:"scope"` // global | tenant | role
	ScopeID     string    `json:"scopeId,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
	UpdatedBy   string    `json:"updatedBy"`
}

// FlagsHandler handles feature flag REST endpoints.
type FlagsHandler struct{}

// NewFlagsHandler creates a FlagsHandler.
func NewFlagsHandler() *FlagsHandler {
	return &FlagsHandler{}
}

// RegisterFlagsRoutes registers feature flag routes on the mux.
func (h *FlagsHandler) RegisterFlagsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/flags", h.ListFlags)
	mux.HandleFunc("PATCH /admin/flags/{id}", h.UpdateFlag)
}

// ListFlags handles GET /admin/flags
func (h *FlagsHandler) ListFlags(w http.ResponseWriter, r *http.Request) {
	flags := []FeatureFlag{
		{
			ID:          "flag-001",
			Name:        "ai_study_plan",
			Description: "Enable AI-generated study plan for students",
			Enabled:     true,
			Scope:       "global",
			UpdatedAt:   time.Now().Add(-24 * time.Hour),
			UpdatedBy:   "admin-001",
		},
		{
			ID:          "flag-002",
			Name:        "live_collaboration",
			Description: "Enable real-time document collaboration in assignments",
			Enabled:     false,
			Scope:       "global",
			UpdatedAt:   time.Now().Add(-48 * time.Hour),
			UpdatedBy:   "admin-001",
		},
		{
			ID:          "flag-003",
			Name:        "video_conferencing",
			Description: "Enable built-in video conferencing (Jitsi)",
			Enabled:     true,
			Scope:       "global",
			UpdatedAt:   time.Now().Add(-72 * time.Hour),
			UpdatedBy:   "admin-001",
		},
		{
			ID:          "flag-004",
			Name:        "discussion_threads",
			Description: "Enable course discussion threads",
			Enabled:     true,
			Scope:       "global",
			UpdatedAt:   time.Now().Add(-96 * time.Hour),
			UpdatedBy:   "admin-001",
		},
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"flags": flags,
		"total": len(flags),
	})
}

// UpdateFlag handles PATCH /admin/flags/:id
func (h *FlagsHandler) UpdateFlag(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	adminID := r.Header.Get("X-User-ID")
	if adminID == "" {
		writeError(w, http.StatusUnauthorized, "missing user ID")
		return
	}

	var req struct {
		Enabled *bool  `json:"enabled"`
		Scope   string `json:"scope"`
		ScopeID string `json:"scopeId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Enabled == nil && req.Scope == "" {
		writeError(w, http.StatusBadRequest, "at least one of enabled or scope is required")
		return
	}

	flag := FeatureFlag{
		ID:        id,
		UpdatedAt: time.Now(),
		UpdatedBy: adminID,
	}
	if req.Enabled != nil {
		flag.Enabled = *req.Enabled
	}
	if req.Scope != "" {
		flag.Scope = req.Scope
		flag.ScopeID = req.ScopeID
	}

	writeJSON(w, http.StatusOK, flag)
}
