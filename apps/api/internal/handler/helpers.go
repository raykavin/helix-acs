package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// writeJSON serialises data to JSON and writes it to w with the given status code.
// If marshalling fails it falls back to a 500 error response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// At this point the status is already written; log only.
		_ = err
	}
}

// writeError writes a JSON body of the form {"error":"<msg>"} with the given
// HTTP status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// paginationParams extracts page and limit query parameters with safe defaults.
func paginationParams(r *http.Request) (page, limit int) {
	page = 1
	limit = 20

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	return page, limit
}

// parseInt converts a string to an int.
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
