package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func mapServiceError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"):
		writeError(w, http.StatusNotFound, msg)
	case strings.Contains(msg, "validation failed"),
		strings.Contains(msg, "is required"),
		strings.Contains(msg, "must be"),
		strings.Contains(msg, "invalid"):
		writeError(w, http.StatusBadRequest, msg)
	case strings.Contains(msg, "already published"),
		strings.Contains(msg, "cannot"):
		writeError(w, http.StatusConflict, msg)
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
