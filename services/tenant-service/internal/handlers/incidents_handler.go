package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// Incident represents a platform or tenant-level incident.
type Incident struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`   // open | watch | resolved
	Priority    string          `json:"priority"` // p1 | p2 | p3 | p4
	TenantID    string          `json:"tenantId,omitempty"`
	CreatedBy   string          `json:"createdBy"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	Timeline    []TimelineEntry `json:"timeline,omitempty"`
}

// TimelineEntry is a single update in the incident timeline.
type TimelineEntry struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"authorId"`
	Message   string    `json:"message"`
	Type      string    `json:"type"` // update | escalation | resolution
	CreatedAt time.Time `json:"createdAt"`
}

// IncidentHandler handles admin incident REST endpoints.
type IncidentHandler struct{}

// NewIncidentHandler creates an IncidentHandler.
func NewIncidentHandler() *IncidentHandler {
	return &IncidentHandler{}
}

// RegisterIncidentRoutes registers incident routes on the mux.
func (h *IncidentHandler) RegisterIncidentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/incidents", h.ListIncidents)
	mux.HandleFunc("POST /admin/incidents", h.CreateIncident)
	mux.HandleFunc("GET /admin/incidents/{id}", h.GetIncident)
	mux.HandleFunc("POST /admin/incidents/{id}/update", h.AddTimelineEntry)
	mux.HandleFunc("POST /admin/incidents/{id}/notify", h.NotifyTenant)
	mux.HandleFunc("PATCH /admin/incidents/{id}/status", h.UpdateStatus)
}

// ListIncidents handles GET /admin/incidents?status=
func (h *IncidentHandler) ListIncidents(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")

	incidents := []Incident{
		{
			ID:        "inc-001",
			Title:     "Database connection spike",
			Status:    "open",
			Priority:  "p2",
			CreatedBy: "admin-001",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt: time.Now().Add(-30 * time.Minute),
		},
		{
			ID:        "inc-002",
			Title:     "Slow asset delivery for tenant acme",
			Status:    "watch",
			Priority:  "p3",
			TenantID:  "tenant-acme",
			CreatedBy: "admin-001",
			CreatedAt: time.Now().Add(-6 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		},
	}

	// Apply filter
	if statusFilter != "" {
		filtered := incidents[:0]
		for _, inc := range incidents {
			if inc.Status == statusFilter {
				filtered = append(filtered, inc)
			}
		}
		incidents = filtered
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"incidents": incidents,
		"total":     len(incidents),
	})
}

// GetIncident handles GET /admin/incidents/:id
func (h *IncidentHandler) GetIncident(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	incident := Incident{
		ID:          id,
		Title:       "Database connection spike",
		Description: "Elevated connection count detected on primary DB cluster.",
		Status:      "open",
		Priority:    "p2",
		CreatedBy:   "admin-001",
		CreatedAt:   time.Now().Add(-2 * time.Hour),
		UpdatedAt:   time.Now().Add(-30 * time.Minute),
		Timeline: []TimelineEntry{
			{
				ID:        "entry-001",
				AuthorID:  "admin-001",
				Message:   "Incident created. Investigating DB metrics.",
				Type:      "update",
				CreatedAt: time.Now().Add(-2 * time.Hour),
			},
			{
				ID:        "entry-002",
				AuthorID:  "admin-001",
				Message:   "Root cause identified: missing index on events table.",
				Type:      "update",
				CreatedAt: time.Now().Add(-1 * time.Hour),
			},
		},
	}
	writeJSON(w, http.StatusOK, incident)
}

// CreateIncident handles POST /admin/incidents
func (h *IncidentHandler) CreateIncident(w http.ResponseWriter, r *http.Request) {
	adminID := r.Header.Get("X-User-ID")
	if adminID == "" {
		writeError(w, http.StatusUnauthorized, "missing user ID")
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
		TenantID    string `json:"tenantId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.Priority == "" {
		req.Priority = "p3"
	}

	incident := Incident{
		ID:          "inc-" + time.Now().Format("20060102150405"),
		Title:       req.Title,
		Description: req.Description,
		Status:      "open",
		Priority:    req.Priority,
		TenantID:    req.TenantID,
		CreatedBy:   adminID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Timeline:    []TimelineEntry{},
	}
	writeJSON(w, http.StatusCreated, incident)
}

// AddTimelineEntry handles POST /admin/incidents/:id/update
func (h *IncidentHandler) AddTimelineEntry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	adminID := r.Header.Get("X-User-ID")

	var req struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}
	if req.Type == "" {
		req.Type = "update"
	}

	entry := TimelineEntry{
		ID:        "entry-" + time.Now().Format("20060102150405"),
		AuthorID:  adminID,
		Message:   req.Message,
		Type:      req.Type,
		CreatedAt: time.Now(),
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"incidentId": id,
		"entry":      entry,
	})
}

// NotifyTenant handles POST /admin/incidents/:id/notify
func (h *IncidentHandler) NotifyTenant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"incidentId": id,
		"status":     "notification_queued",
		"sentAt":     time.Now(),
	})
}

// UpdateStatus handles PATCH /admin/incidents/:id/status
func (h *IncidentHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Status   string `json:"status"`
		Priority string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status == "" && req.Priority == "" {
		writeError(w, http.StatusBadRequest, "status or priority is required")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"incidentId": id,
		"status":     req.Status,
		"priority":   req.Priority,
		"updatedAt":  time.Now(),
	})
}
