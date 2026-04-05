package middleware

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"github.com/raykavin/helix-acs/internal/logger"
)

// Recovery returns middleware that catches panics, logs them with a full stack
// trace, and returns an HTTP 500 JSON error response so the server keeps running.
func Recovery(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()

					log.WithField("panic", rec).
						WithField("stack", stack).
						WithField("method", r.Method).
						WithField("path", r.URL.Path).
						Error("Recovered from panic")

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
