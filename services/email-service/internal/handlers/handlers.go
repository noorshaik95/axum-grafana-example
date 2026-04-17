package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"slate/services/email-service/internal/kafka"
	"slate/services/email-service/internal/models"
	"slate/services/email-service/internal/repository"

	"github.com/go-chi/chi/v5"
)

type MessageHandler struct {
	repo     *repository.MessageRepository
	producer *kafka.Producer
}

func NewMessageHandler(repo *repository.MessageRepository, producer *kafka.Producer) *MessageHandler {
	return &MessageHandler{repo: repo, producer: producer}
}

// getUserID extracts user ID from JWT claims or X-User-ID header.
func getUserID(r *http.Request) string {
	if uid := r.Header.Get("X-User-ID"); uid != "" {
		return uid
	}
	return ""
}

// getTenantID extracts tenant ID from header.
func getTenantID(r *http.Request) string {
	if tid := r.Header.Get("X-Tenant-ID"); tid != "" {
		return tid
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// SendMessage handles POST /messages
func (h *MessageHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	tenantID := getTenantID(r)
	if userID == "" || tenantID == "" {
		writeError(w, http.StatusUnauthorized, "missing user or tenant ID")
		return
	}

	var req models.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.RecipientIDs) == 0 {
		writeError(w, http.StatusBadRequest, "at least one recipient required")
		return
	}
	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	// Use tenant from header, not from body
	msg, err := h.repo.SendMessage(r.Context(), tenantID, userID, req.Subject, req.Body, req.RecipientIDs)
	if err != nil {
		log.Printf("ERROR send message: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to send message")
		return
	}

	h.producer.PublishMessageSent(r.Context(), msg.ID, userID, req.RecipientIDs, req.Subject, tenantID)

	writeJSON(w, http.StatusCreated, msg)
}

// BulkSend handles POST /messages/bulk
func (h *MessageHandler) BulkSend(w http.ResponseWriter, r *http.Request) {
	// Same as SendMessage — the model already supports multiple recipients
	h.SendMessage(w, r)
}

// GetInbox handles GET /messages/inbox
func (h *MessageHandler) GetInbox(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	tenantID := getTenantID(r)
	if userID == "" || tenantID == "" {
		writeError(w, http.StatusUnauthorized, "missing user or tenant ID")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	result, err := h.repo.GetInbox(r.Context(), tenantID, userID, cursor, limit)
	if err != nil {
		log.Printf("ERROR get inbox: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get inbox")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetSent handles GET /messages/sent
func (h *MessageHandler) GetSent(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	tenantID := getTenantID(r)
	if userID == "" || tenantID == "" {
		writeError(w, http.StatusUnauthorized, "missing user or tenant ID")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	result, err := h.repo.GetSent(r.Context(), tenantID, userID, cursor, limit)
	if err != nil {
		log.Printf("ERROR get sent: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get sent messages")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetMessage handles GET /messages/:id
func (h *MessageHandler) GetMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	msgID := chi.URLParam(r, "id")

	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user ID")
		return
	}

	canAccess, err := h.repo.UserCanAccessMessage(r.Context(), msgID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "access check failed")
		return
	}
	if !canAccess {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	msg, err := h.repo.GetMessage(r.Context(), msgID)
	if err != nil {
		log.Printf("ERROR get message: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get message")
		return
	}
	if msg == nil {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}

	writeJSON(w, http.StatusOK, msg)
}

// ReplyMessage handles POST /messages/:id/reply
func (h *MessageHandler) ReplyMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	parentID := chi.URLParam(r, "id")

	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user ID")
		return
	}

	canAccess, err := h.repo.UserCanAccessMessage(r.Context(), parentID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "access check failed")
		return
	}
	if !canAccess {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	var req models.ReplyMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	msg, err := h.repo.ReplyToMessage(r.Context(), parentID, userID, req.Body)
	if err != nil {
		log.Printf("ERROR reply: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to reply")
		return
	}

	writeJSON(w, http.StatusCreated, msg)
}

// MarkRead handles PATCH /messages/:id/read
func (h *MessageHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	msgID := chi.URLParam(r, "id")

	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user ID")
		return
	}

	if err := h.repo.MarkRead(r.Context(), msgID, userID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	h.producer.PublishMessageRead(r.Context(), msgID, userID, getTenantID(r))

	writeJSON(w, http.StatusOK, map[string]string{"status": "read"})
}

// MarkUnread handles PATCH /messages/:id/unread
func (h *MessageHandler) MarkUnread(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	msgID := chi.URLParam(r, "id")

	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user ID")
		return
	}

	if err := h.repo.MarkUnread(r.Context(), msgID, userID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "unread"})
}

// DeleteMessage handles DELETE /messages/:id
func (h *MessageHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	msgID := chi.URLParam(r, "id")

	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user ID")
		return
	}

	if err := h.repo.SoftDelete(r.Context(), msgID, userID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GetUnreadCount handles GET /messages/unread-count
func (h *MessageHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	tenantID := getTenantID(r)

	if userID == "" || tenantID == "" {
		writeError(w, http.StatusUnauthorized, "missing user or tenant ID")
		return
	}

	count, err := h.repo.GetUnreadCount(r.Context(), tenantID, userID)
	if err != nil {
		log.Printf("ERROR unread count: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get unread count")
		return
	}

	writeJSON(w, http.StatusOK, models.UnreadCountResponse{Count: count})
}

// GetThread handles GET /messages/threads/:threadId
func (h *MessageHandler) GetThread(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	threadID := chi.URLParam(r, "threadId")

	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user ID")
		return
	}

	messages, err := h.repo.GetThread(r.Context(), threadID)
	if err != nil {
		log.Printf("ERROR get thread: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get thread")
		return
	}

	writeJSON(w, http.StatusOK, messages)
}
