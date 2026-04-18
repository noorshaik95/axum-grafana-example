package handlers

import (
	"net/http"
	"strings"
	"time"
)

// SubmissionSummary is a lightweight summary for the grading queue.
type SubmissionSummary struct {
	SubmissionID string    `json:"submissionId"`
	StudentID    string    `json:"studentId"`
	StudentName  string    `json:"studentName"`
	SubmittedAt  time.Time `json:"submittedAt"`
	Status       string    `json:"status"`
	PatternID    string    `json:"patternId,omitempty"`
	PatternLabel string    `json:"patternLabel,omitempty"`
}

// PatternGroup groups submissions by detected answer pattern.
type PatternGroup struct {
	PatternID    string              `json:"patternId"`
	PatternLabel string              `json:"patternLabel"`
	Count        int                 `json:"count"`
	Submissions  []SubmissionSummary `json:"submissions"`
}

// GradingQueueHandler handles queue-style grading endpoints.
type GradingQueueHandler struct{}

// NewGradingQueueHandler creates a GradingQueueHandler.
func NewGradingQueueHandler() *GradingQueueHandler {
	return &GradingQueueHandler{}
}

// RegisterRoutes adds grading queue routes to the mux.
func (h *GradingQueueHandler) RegisterRoutes(mux *http.ServeMux) {
	// GET /grading/queue/:assignmentId
	// GET /grading/queue/:assignmentId/:submissionId
	mux.HandleFunc("/grading/queue/", h.handleQueuePath)

	// POST /grading/submit/:submissionId
	mux.HandleFunc("/grading/submit/", h.handleSubmitPath)

	// POST /grading/batch
	mux.HandleFunc("/grading/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.BatchGrade(w, r)
	})
}

func (h *GradingQueueHandler) handleQueuePath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/grading/queue/")
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")

	switch len(parts) {
	case 1:
		// /grading/queue/:assignmentId
		h.ListQueue(w, r, parts[0])
	case 2:
		// /grading/queue/:assignmentId/:submissionId
		h.GetQueueSubmission(w, r, parts[0], parts[1])
	default:
		notFound(w)
	}
}

func (h *GradingQueueHandler) handleSubmitPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	submissionID := strings.TrimPrefix(r.URL.Path, "/grading/submit/")
	submissionID = strings.TrimSuffix(submissionID, "/")
	if submissionID == "" {
		notFound(w)
		return
	}
	h.SubmitGrade(w, r, submissionID)
}

// ListQueue handles GET /grading/queue/:assignmentId
// Returns submissions grouped by answer pattern.
func (h *GradingQueueHandler) ListQueue(w http.ResponseWriter, r *http.Request, assignmentID string) {
	// Stub: return mock pattern groups
	groups := []PatternGroup{
		{
			PatternID:    "pattern-001",
			PatternLabel: "Correct approach with minor errors",
			Count:        12,
			Submissions: []SubmissionSummary{
				{
					SubmissionID: "sub-001",
					StudentID:    "student-001",
					StudentName:  "Alice",
					SubmittedAt:  time.Now().Add(-3 * time.Hour),
					Status:       "pending",
					PatternID:    "pattern-001",
					PatternLabel: "Correct approach with minor errors",
				},
			},
		},
		{
			PatternID:    "pattern-002",
			PatternLabel: "Misunderstood the question",
			Count:        5,
			Submissions:  []SubmissionSummary{},
		},
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"assignmentId": assignmentID,
		"patterns":     groups,
		"totalPending": 17,
	})
}

// GetQueueSubmission handles GET /grading/queue/:assignmentId/:submissionId
func (h *GradingQueueHandler) GetQueueSubmission(w http.ResponseWriter, r *http.Request, assignmentID, submissionID string) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"assignmentId": assignmentID,
		"submissionId": submissionID,
		"studentId":    "student-001",
		"content":      "Student submission content here...",
		"attachments":  []interface{}{},
		"submittedAt":  time.Now().Add(-3 * time.Hour),
		"status":       "pending",
	})
}

// SubmitGrade handles POST /grading/submit/:submissionId
func (h *GradingQueueHandler) SubmitGrade(w http.ResponseWriter, r *http.Request, submissionID string) {
	var req struct {
		Score    float64 `json:"score"`
		Feedback string  `json:"feedback"`
		GradedBy string  `json:"gradedBy"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.GradedBy == "" {
		writeError(w, http.StatusBadRequest, "gradedBy is required")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"submissionId": submissionID,
		"score":        req.Score,
		"feedback":     req.Feedback,
		"gradedBy":     req.GradedBy,
		"gradedAt":     time.Now(),
		"status":       "graded",
	})
}

// BatchGrade handles POST /grading/batch
func (h *GradingQueueHandler) BatchGrade(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AssignmentID string  `json:"assignmentId"`
		PatternID    string  `json:"patternId"`
		Score        float64 `json:"score"`
		Feedback     string  `json:"feedback"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AssignmentID == "" || req.PatternID == "" {
		writeError(w, http.StatusBadRequest, "assignmentId and patternId are required")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"assignmentId":   req.AssignmentID,
		"patternId":      req.PatternID,
		"score":          req.Score,
		"gradedCount":    0, // stub
		"message":        "batch grading queued",
	})
}
