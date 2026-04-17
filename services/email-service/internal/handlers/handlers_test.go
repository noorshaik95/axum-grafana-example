package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestGetUserID(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-User-ID", "user-123")

	got := getUserID(req)
	if got != "user-123" {
		t.Errorf("getUserID() = %q, want %q", got, "user-123")
	}
}

func TestGetUserID_Empty(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := getUserID(req)
	if got != "" {
		t.Errorf("getUserID() = %q, want empty", got)
	}
}

func TestGetTenantID(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-456")

	got := getTenantID(req)
	if got != "tenant-456" {
		t.Errorf("getTenantID() = %q, want %q", got, "tenant-456")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}
	writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "bad request")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "bad request" {
		t.Errorf("error = %q, want %q", resp["error"], "bad request")
	}
}

func TestSendMessage_NoAuth(t *testing.T) {
	h := &MessageHandler{}
	req := httptest.NewRequest("POST", "/messages", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()

	h.SendMessage(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestSendMessage_InvalidBody(t *testing.T) {
	h := &MessageHandler{}
	req := httptest.NewRequest("POST", "/messages", bytes.NewBufferString(`not json`))
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set("X-Tenant-ID", "tenant-1")
	w := httptest.NewRecorder()

	h.SendMessage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSendMessage_NoRecipients(t *testing.T) {
	h := &MessageHandler{}
	body := `{"body":"hello","recipient_ids":[]}`
	req := httptest.NewRequest("POST", "/messages", bytes.NewBufferString(body))
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set("X-Tenant-ID", "tenant-1")
	w := httptest.NewRecorder()

	h.SendMessage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSendMessage_NoBody(t *testing.T) {
	h := &MessageHandler{}
	body := `{"body":"","recipient_ids":["user-2"]}`
	req := httptest.NewRequest("POST", "/messages", bytes.NewBufferString(body))
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set("X-Tenant-ID", "tenant-1")
	w := httptest.NewRecorder()

	h.SendMessage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetInbox_NoAuth(t *testing.T) {
	h := &MessageHandler{}
	req := httptest.NewRequest("GET", "/messages/inbox", nil)
	w := httptest.NewRecorder()

	h.GetInbox(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGetMessage_NoAuth(t *testing.T) {
	h := &MessageHandler{}
	req := httptest.NewRequest("GET", "/messages/test-id", nil)

	w := httptest.NewRecorder()
	h.GetMessage(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestMarkRead_NoAuth(t *testing.T) {
	h := &MessageHandler{}
	req := httptest.NewRequest("PATCH", "/messages/test-id/read", nil)
	w := httptest.NewRecorder()

	h.MarkRead(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestDeleteMessage_NoAuth(t *testing.T) {
	h := &MessageHandler{}
	req := httptest.NewRequest("DELETE", "/messages/test-id", nil)
	w := httptest.NewRecorder()

	h.DeleteMessage(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGetUnreadCount_NoAuth(t *testing.T) {
	h := &MessageHandler{}
	req := httptest.NewRequest("GET", "/messages/unread-count", nil)
	w := httptest.NewRecorder()

	h.GetUnreadCount(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// Test route wiring using chi router
func TestRoutes(t *testing.T) {
	r := chi.NewRouter()
	h := &MessageHandler{}

	r.Route("/messages", func(r chi.Router) {
		r.Post("/", h.SendMessage)
		r.Post("/bulk", h.BulkSend)
		r.Get("/inbox", h.GetInbox)
		r.Get("/sent", h.GetSent)
		r.Get("/unread-count", h.GetUnreadCount)
		r.Get("/threads/{threadId}", h.GetThread)
		r.Get("/{id}", h.GetMessage)
		r.Post("/{id}/reply", h.ReplyMessage)
		r.Patch("/{id}/read", h.MarkRead)
		r.Patch("/{id}/unread", h.MarkUnread)
		r.Delete("/{id}", h.DeleteMessage)
	})

	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/messages"},
		{"POST", "/messages/bulk"},
		{"GET", "/messages/inbox"},
		{"GET", "/messages/sent"},
		{"GET", "/messages/unread-count"},
		{"GET", "/messages/threads/thread-123"},
		{"GET", "/messages/msg-123"},
		{"POST", "/messages/msg-123/reply"},
		{"PATCH", "/messages/msg-123/read"},
		{"PATCH", "/messages/msg-123/unread"},
		{"DELETE", "/messages/msg-123"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			var body *bytes.Buffer
			if route.method == "POST" {
				body = bytes.NewBufferString(`{}`)
			} else {
				body = &bytes.Buffer{}
			}
			req := httptest.NewRequest(route.method, route.path, body)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			// Should not get 404/405 (route not found)
			if w.Code == http.StatusNotFound || w.Code == http.StatusMethodNotAllowed {
				t.Errorf("route %s %s returned %d", route.method, route.path, w.Code)
			}
		})
	}
}
