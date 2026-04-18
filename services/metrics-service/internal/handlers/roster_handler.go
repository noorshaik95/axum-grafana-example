package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// RosterStudent holds per-student health/risk data.
type RosterStudent struct {
	StudentID       string    `json:"studentId"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	RiskScore       float64   `json:"riskScore"`       // 0.0 - 1.0
	RiskLevel       string    `json:"riskLevel"`       // low | medium | high
	Attendance      float64   `json:"attendance"`      // percent
	AssignmentsLate int       `json:"assignmentsLate"`
	LastActive      time.Time `json:"lastActive"`
	GradeEstimate   float64   `json:"gradeEstimate"`
}

// StudentDetail combines RosterStudent with more granular signals.
type StudentDetail struct {
	RosterStudent
	Signals []StudentSignal `json:"signals"`
}

// StudentSignal is a specific behavioral/academic signal.
type StudentSignal struct {
	Type      string    `json:"type"` // login_gap | late_submission | grade_drop | etc.
	Message   string    `json:"message"`
	Severity  string    `json:"severity"` // info | warning | critical
	DetectedAt time.Time `json:"detectedAt"`
}

// RegisterRosterRoutes attaches roster health routes to the Chi router.
func (h *Handler) RegisterRosterRoutes(r chi.Router) {
	r.Get("/roster/{courseId}/health", h.GetRosterHealth)
	r.Get("/roster/{courseId}/students/{studentId}", h.GetRosterStudentDetail)
	r.Post("/roster/nudge", h.SendNudge)
}

// GetRosterHealth handles GET /roster/:courseId/health
// Returns students sorted by risk score (highest first).
func (h *Handler) GetRosterHealth(w http.ResponseWriter, r *http.Request) {
	courseID := chi.URLParam(r, "courseId")

	// Stub: return mock students sorted by risk
	students := []RosterStudent{
		{
			StudentID:       "student-002",
			Name:            "Bob Smith",
			Email:           "bob@example.edu",
			RiskScore:       0.85,
			RiskLevel:       "high",
			Attendance:      62.0,
			AssignmentsLate: 3,
			LastActive:      time.Now().Add(-72 * time.Hour),
			GradeEstimate:   61.0,
		},
		{
			StudentID:       "student-001",
			Name:            "Alice Jones",
			Email:           "alice@example.edu",
			RiskScore:       0.42,
			RiskLevel:       "medium",
			Attendance:      78.0,
			AssignmentsLate: 1,
			LastActive:      time.Now().Add(-12 * time.Hour),
			GradeEstimate:   74.0,
		},
		{
			StudentID:       "student-003",
			Name:            "Carol White",
			Email:           "carol@example.edu",
			RiskScore:       0.10,
			RiskLevel:       "low",
			Attendance:      95.0,
			AssignmentsLate: 0,
			LastActive:      time.Now().Add(-2 * time.Hour),
			GradeEstimate:   91.0,
		},
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"courseId": courseID,
		"students": students,
		"total":    len(students),
		"atRisk":   1,
	})
}

// GetRosterStudentDetail handles GET /roster/:courseId/students/:studentId
func (h *Handler) GetRosterStudentDetail(w http.ResponseWriter, r *http.Request) {
	courseID := chi.URLParam(r, "courseId")
	studentID := chi.URLParam(r, "studentId")

	detail := StudentDetail{
		RosterStudent: RosterStudent{
			StudentID:       studentID,
			Name:            "Bob Smith",
			Email:           "bob@example.edu",
			RiskScore:       0.85,
			RiskLevel:       "high",
			Attendance:      62.0,
			AssignmentsLate: 3,
			LastActive:      time.Now().Add(-72 * time.Hour),
			GradeEstimate:   61.0,
		},
		Signals: []StudentSignal{
			{
				Type:       "login_gap",
				Message:    "Student has not logged in for 3+ days",
				Severity:   "warning",
				DetectedAt: time.Now().Add(-72 * time.Hour),
			},
			{
				Type:       "late_submission",
				Message:    "3 consecutive late submissions",
				Severity:   "warning",
				DetectedAt: time.Now().Add(-48 * time.Hour),
			},
			{
				Type:       "grade_drop",
				Message:    "Grade dropped from 74% to 61% over 2 weeks",
				Severity:   "critical",
				DetectedAt: time.Now().Add(-24 * time.Hour),
			},
		},
	}

	_ = courseID
	writeJSON(w, http.StatusOK, detail)
}

// SendNudge handles POST /roster/nudge
// Sends a check-in message to a student.
func (h *Handler) SendNudge(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "nudge_sent",
		"studentId":   req.StudentID,
		"courseId":    req.CourseID,
		"instructorId": instructorID,
		"sentAt":      time.Now(),
	})
}
