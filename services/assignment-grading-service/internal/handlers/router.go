package handlers

import (
	"net/http"
	"strings"

	"slate/services/assignment-grading-service/internal/service"
)

// Router sets up all REST API routes
type Router struct {
	assignmentHandler   *AssignmentHandler
	submissionHandler   *SubmissionHandler
	gradingHandler      *GradingHandler
	gradebookHandler    *GradebookHandler
	gradingRuleHandler  *GradingRuleHandler
	gradingQueueHandler *GradingQueueHandler
}

// NewRouter creates a new REST API router
func NewRouter(
	assignmentSvc service.AssignmentService,
	submissionSvc service.SubmissionService,
	gradingSvc service.GradingService,
	gradebookSvc service.GradebookService,
	gradingRuleSvc service.GradingRuleService,
) *Router {
	return &Router{
		assignmentHandler:   NewAssignmentHandler(assignmentSvc),
		submissionHandler:   NewSubmissionHandler(submissionSvc),
		gradingHandler:      NewGradingHandler(gradingSvc),
		gradebookHandler:    NewGradebookHandler(gradebookSvc),
		gradingRuleHandler:  NewGradingRuleHandler(gradingRuleSvc),
		gradingQueueHandler: NewGradingQueueHandler(),
	}
}

// Handler returns the http.Handler for all REST endpoints
func (r *Router) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"assignment-grading-service"}`))
	})

	// Grading queue (instructor queue-style grading + batch grading)
	r.gradingQueueHandler.RegisterRoutes(mux)

	// Assignments
	mux.HandleFunc("/assignments", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodPost:
			r.assignmentHandler.Create(w, req)
		case http.MethodGet:
			r.assignmentHandler.List(w, req)
		default:
			methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/assignments/", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/assignments/")
		parts := strings.Split(path, "/")

		if len(parts) == 0 || parts[0] == "" {
			notFound(w)
			return
		}

		assignmentID := parts[0]

		// /assignments/:id/submissions
		if len(parts) >= 2 && parts[1] == "submissions" {
			if len(parts) == 3 && parts[2] == "zip" {
				// /assignments/:id/submissions/zip
				if req.Method == http.MethodGet {
					r.submissionHandler.DownloadZip(w, req, assignmentID)
				} else {
					methodNotAllowed(w)
				}
				return
			}
			switch req.Method {
			case http.MethodPost:
				r.submissionHandler.Submit(w, req, assignmentID)
			case http.MethodGet:
				r.submissionHandler.ListByAssignment(w, req, assignmentID)
			default:
				methodNotAllowed(w)
			}
			return
		}

		// /assignments/:id/auto-grade
		if len(parts) == 2 && parts[1] == "auto-grade" {
			if req.Method == http.MethodPost {
				r.gradingHandler.AutoGrade(w, req, assignmentID)
			} else {
				methodNotAllowed(w)
			}
			return
		}

		// /assignments/:id
		if len(parts) == 1 {
			switch req.Method {
			case http.MethodGet:
				r.assignmentHandler.Get(w, req, assignmentID)
			case http.MethodPut:
				r.assignmentHandler.Update(w, req, assignmentID)
			case http.MethodDelete:
				r.assignmentHandler.Delete(w, req, assignmentID)
			default:
				methodNotAllowed(w)
			}
			return
		}

		notFound(w)
	})

	// Submissions
	mux.HandleFunc("/submissions/", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/submissions/")
		parts := strings.Split(path, "/")

		if len(parts) == 0 || parts[0] == "" {
			notFound(w)
			return
		}

		submissionID := parts[0]

		// /submissions/:id/grade
		if len(parts) == 2 && parts[1] == "grade" {
			if req.Method == http.MethodPost {
				r.gradingHandler.GradeSubmission(w, req, submissionID)
			} else {
				methodNotAllowed(w)
			}
			return
		}

		// /submissions/:id
		if len(parts) == 1 {
			if req.Method == http.MethodGet {
				r.submissionHandler.Get(w, req, submissionID)
			} else {
				methodNotAllowed(w)
			}
			return
		}

		notFound(w)
	})

	// Courses - grades
	mux.HandleFunc("/courses/", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/courses/")
		parts := strings.Split(path, "/")

		if len(parts) < 2 || parts[0] == "" {
			notFound(w)
			return
		}

		courseID := parts[0]

		// /courses/:id/grades/distribution
		if len(parts) == 3 && parts[1] == "grades" && parts[2] == "distribution" {
			if req.Method == http.MethodGet {
				r.gradebookHandler.GetDistribution(w, req, courseID)
			} else {
				methodNotAllowed(w)
			}
			return
		}

		// /courses/:id/grades
		if len(parts) == 2 && parts[1] == "grades" {
			if req.Method == http.MethodGet {
				r.gradebookHandler.GetCourseGrades(w, req, courseID)
			} else {
				methodNotAllowed(w)
			}
			return
		}

		notFound(w)
	})

	// Students - grades
	mux.HandleFunc("/students/", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/students/")
		parts := strings.Split(path, "/")

		if len(parts) < 2 || parts[0] == "" {
			notFound(w)
			return
		}

		studentID := parts[0]

		// /students/:id/grade-estimate
		if len(parts) == 2 && parts[1] == "grade-estimate" {
			if req.Method == http.MethodGet {
				r.gradebookHandler.GetGradeEstimate(w, req, studentID)
			} else {
				methodNotAllowed(w)
			}
			return
		}

		// /students/:id/grades
		if len(parts) == 2 && parts[1] == "grades" {
			if req.Method == http.MethodGet {
				r.gradebookHandler.GetStudentGrades(w, req, studentID)
			} else {
				methodNotAllowed(w)
			}
			return
		}

		notFound(w)
	})

	// Grading rules
	mux.HandleFunc("/grading-rules", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			r.gradingRuleHandler.List(w, req)
		case http.MethodPost:
			r.gradingRuleHandler.Create(w, req)
		default:
			methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/grading-rules/", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/grading-rules/")
		parts := strings.Split(path, "/")

		if len(parts) == 0 || parts[0] == "" {
			notFound(w)
			return
		}

		ruleID := parts[0]

		switch req.Method {
		case http.MethodPut:
			r.gradingRuleHandler.Update(w, req, ruleID)
		case http.MethodDelete:
			r.gradingRuleHandler.Delete(w, req, ruleID)
		default:
			methodNotAllowed(w)
		}
	})

	return mux
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
}

func notFound(w http.ResponseWriter) {
	http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
}
