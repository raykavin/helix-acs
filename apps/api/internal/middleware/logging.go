package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/raykavin/helix-acs/packages/logger"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Logging returns middleware that logs each request using zerolog. It attaches
// a unique X-Request-ID header to every response and logs method, path, status
// code, and elapsed duration.
func Logging(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Reuse a provided request ID or generate a new one.
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.NewString()
			}
			w.Header().Set("X-Request-ID", requestID)

			rw := newResponseWriter(w)
			next.ServeHTTP(rw, r)

			dur := time.Since(start)

			msg := "HTTP Request"
			event := log.
				WithField("request_id", requestID).
				WithField("method", r.Method).
				WithField("path", r.URL.Path).
				WithField("status", rw.status).
				WithField("duration", dur).
				WithField("remote_addr", r.RemoteAddr)

			if rw.status >= 500 {
				event.Error(msg)
			} else if rw.status >= 400 {
				event.Warn(msg)
			} else {
				event.Info(msg)
			}

		})
	}
}
