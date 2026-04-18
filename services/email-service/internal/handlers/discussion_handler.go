package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// DiscussionThread represents a course Q&A thread.
type DiscussionThread struct {
	ID           string            `json:"id"`
	CourseID     string            `json:"courseId"`
	AssignmentID string            `json:"assignmentId,omitempty"`
	AuthorID     string            `json:"authorId"`
	Title        string            `json:"title"`
	Body         string            `json:"body"`
	ReplyCount   int               `json:"replyCount"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	Replies      []DiscussionReply `json:"replies,omitempty"`
}

// DiscussionReply is a single reply within a thread.
type DiscussionReply struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"threadId"`
	AuthorID  string    `json:"authorId"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

// DiscussionHandler handles discussion thread HTTP endpoints.
type DiscussionHandler struct{}

// NewDiscussionHandler constructs a DiscussionHandler.
func NewDiscussionHandler() *DiscussionHandler {
	return &DiscussionHandler{}
}

// RegisterDiscussionRoutes attaches discussion routes to a Chi router.
func (h *DiscussionHandler) RegisterDiscussionRoutes(r chi.Router) {
	r.Route("/discussions", func(r chi.Router) {
		r.Get("/", h.ListThreads)
		r.Post("/", h.CreateThread)
		r.Get("/{id}", h.GetThread)
		r.Post("/{id}/replies", h.AddReply)
	})
}

// ListThreads handles GET /discussions?courseId=&assignmentId=
func (h *DiscussionHandler) ListThreads(w http.ResponseWriter, r *http.Request) {
	courseID := r.URL.Query().Get("courseId")
	assignmentID := r.URL.Query().Get("assignmentId")

	// Stub: return mock threads scoped to the query params
	threads := []DiscussionThread{
		{
			ID:           "thread-001",
			CourseID:     courseID,
			AssignmentID: assignmentID,
			AuthorID:     "student-001",
			Title:        "Question about assignment 1",
			Body:         "Can someone help me understand the requirements?",
			ReplyCount:   2,
			CreatedAt:    time.Now().Add(-24 * time.Hour),
			UpdatedAt:    time.Now().Add(-2 * time.Hour),
		},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"threads": threads, "total": len(threads)})
}

// GetThread handles GET /discussions/:id
func (h *DiscussionHandler) GetThread(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	thread := DiscussionThread{
		ID:       id,
		AuthorID: "student-001",
		Title:    "Discussion thread",
		Body:     "Thread body",
		Replies: []DiscussionReply{
			{
				ID:        "reply-001",
				ThreadID:  id,
				AuthorID:  "instructor-001",
				Body:      "Great question! Here is the answer...",
				CreatedAt: time.Now().Add(-1 * time.Hour),
			},
		},
		ReplyCount: 1,
		CreatedAt:  time.Now().Add(-24 * time.Hour),
		UpdatedAt:  time.Now().Add(-1 * time.Hour),
	}
	writeJSON(w, http.StatusOK, thread)
}

// CreateThread handles POST /discussions
func (h *DiscussionHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user ID")
		return
	}

	var req struct {
		CourseID     string `json:"courseId"`
		AssignmentID string `json:"assignmentId"`
		Title        string `json:"title"`
		Body         string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CourseID == "" || req.Title == "" || req.Body == "" {
		writeError(w, http.StatusBadRequest, "courseId, title, and body are required")
		return
	}

	thread := DiscussionThread{
		ID:           "thread-" + time.Now().Format("20060102150405"),
		CourseID:     req.CourseID,
		AssignmentID: req.AssignmentID,
		AuthorID:     userID,
		Title:        req.Title,
		Body:         req.Body,
		ReplyCount:   0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	writeJSON(w, http.StatusCreated, thread)
}

// AddReply handles POST /discussions/:id/replies
func (h *DiscussionHandler) AddReply(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	threadID := chi.URLParam(r, "id")

	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user ID")
		return
	}

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	reply := DiscussionReply{
		ID:        "reply-" + time.Now().Format("20060102150405"),
		ThreadID:  threadID,
		AuthorID:  userID,
		Body:      req.Body,
		CreatedAt: time.Now(),
	}
	writeJSON(w, http.StatusCreated, reply)
}
