package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// BroadcastMessage represents an admin broadcast.
type BroadcastMessage struct {
	ID         string    `json:"id"`
	AuthorID   string    `json:"authorId"`
	Subject    string    `json:"subject"`
	Body       string    `json:"body"`
	TargetType string    `json:"targetType"` // all | tenant | role
	TargetID   string    `json:"targetId,omitempty"`
	SentAt     time.Time `json:"sentAt"`
	RecipCount int       `json:"recipientCount"`
}

// BroadcastHandler handles admin broadcast HTTP endpoints.
type BroadcastHandler struct{}

// NewBroadcastHandler constructs a BroadcastHandler.
func NewBroadcastHandler() *BroadcastHandler {
	return &BroadcastHandler{}
}

// RegisterBroadcastRoutes attaches broadcast routes to an http.ServeMux equivalent (chi router).
func (h *BroadcastHandler) RegisterBroadcastRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /admin/broadcast", h.CreateBroadcast)
	mux.HandleFunc("GET /admin/broadcast/history", h.GetHistory)
}

// CreateBroadcast handles POST /admin/broadcast
func (h *BroadcastHandler) CreateBroadcast(w http.ResponseWriter, r *http.Request) {
	adminID := r.Header.Get("X-User-ID")
	if adminID == "" {
		writeError(w, http.StatusUnauthorized, "missing user ID")
		return
	}

	var req struct {
		Subject    string `json:"subject"`
		Body       string `json:"body"`
		TargetType string `json:"targetType"`
		TargetID   string `json:"targetId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Subject == "" || req.Body == "" {
		writeError(w, http.StatusBadRequest, "subject and body are required")
		return
	}
	if req.TargetType == "" {
		req.TargetType = "all"
	}

	broadcast := BroadcastMessage{
		ID:         "broadcast-" + time.Now().Format("20060102150405"),
		AuthorID:   adminID,
		Subject:    req.Subject,
		Body:       req.Body,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		SentAt:     time.Now(),
		RecipCount: 0, // Would be computed from actual recipient list
	}
	writeJSON(w, http.StatusCreated, broadcast)
}

// GetHistory handles GET /admin/broadcast/history
func (h *BroadcastHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	// Stub: return empty history list
	broadcasts := []BroadcastMessage{}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"broadcasts": broadcasts,
		"total":      0,
	})
}
