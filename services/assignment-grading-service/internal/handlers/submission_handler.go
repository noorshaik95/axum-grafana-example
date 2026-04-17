package handlers

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"

	"slate/services/assignment-grading-service/internal/service"
	"slate/services/assignment-grading-service/pkg/storage"
)

// SubmissionHandler handles submission REST endpoints
type SubmissionHandler struct {
	svc     service.SubmissionService
	storage storage.FileStorage
}

// NewSubmissionHandler creates a new SubmissionHandler
func NewSubmissionHandler(svc service.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{svc: svc}
}

// NewSubmissionHandlerWithStorage creates a handler with file storage for ZIP downloads
func NewSubmissionHandlerWithStorage(svc service.SubmissionService, fs storage.FileStorage) *SubmissionHandler {
	return &SubmissionHandler{svc: svc, storage: fs}
}

type submitRequest struct {
	StudentID string   `json:"student_id"`
	FileURLs  []string `json:"file_urls"`
	TenantID  string   `json:"tenant_id"`
}

// Submit handles POST /assignments/:id/submissions
func (h *SubmissionHandler) Submit(w http.ResponseWriter, r *http.Request, assignmentID string) {
	var req submitRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.StudentID == "" {
		writeError(w, http.StatusBadRequest, "student_id is required")
		return
	}

	if len(req.FileURLs) == 0 {
		writeError(w, http.StatusBadRequest, "file_urls is required")
		return
	}

	submission, err := h.svc.SubmitAssignmentWithURLs(r.Context(), req.TenantID, assignmentID, req.StudentID, req.FileURLs)
	if err != nil {
		mapServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, submission)
}

// Get handles GET /submissions/:id
func (h *SubmissionHandler) Get(w http.ResponseWriter, r *http.Request, id string) {
	submission, err := h.svc.GetSubmission(r.Context(), id)
	if err != nil {
		mapServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, submission)
}

// ListByAssignment handles GET /assignments/:id/submissions
func (h *SubmissionHandler) ListByAssignment(w http.ResponseWriter, r *http.Request, assignmentID string) {
	tenantID := r.URL.Query().Get("tenantId")
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "pageSize", 20)

	submissions, total, err := h.svc.ListSubmissionsPaginated(r.Context(), assignmentID, tenantID, page, pageSize)
	if err != nil {
		mapServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"submissions": submissions,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
	})
}

// DownloadZip handles GET /assignments/:id/submissions/zip
func (h *SubmissionHandler) DownloadZip(w http.ResponseWriter, r *http.Request, assignmentID string) {
	if h.storage == nil {
		writeError(w, http.StatusNotImplemented, "file storage not configured for ZIP downloads")
		return
	}

	submissions, err := h.svc.ListSubmissions(r.Context(), assignmentID, "submitted_at", "DESC")
	if err != nil {
		mapServiceError(w, err)
		return
	}

	if len(submissions) == 0 {
		writeError(w, http.StatusNotFound, "no submissions found")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"submissions_%s.zip\"", assignmentID))

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, sub := range submissions {
		filePath := sub.FilePath
		if filePath == "" && len(sub.FileURLs) > 0 {
			filePath = sub.FileURLs[0]
		}
		if filePath == "" {
			continue
		}

		reader, err := h.storage.Get(filePath)
		if err != nil {
			continue
		}

		entryName := fmt.Sprintf("%s_%s", sub.StudentID, filePath)
		fw, err := zw.Create(entryName)
		if err != nil {
			reader.Close()
			continue
		}

		io.Copy(fw, reader)
		reader.Close()
	}
}
