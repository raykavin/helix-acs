package cwmp

import (
	"net/http"

	"github.com/raykavin/helix-acs/internal/auth"
	"github.com/raykavin/helix-acs/internal/logger"
)

// Server is a thin HTTP wrapper around the CWMP session Handler. It wires
// together Digest authentication, body-size limiting, and request routing so
// that Handler can stay focused on protocol logic.
type Server struct {
	handler    *Handler
	digestAuth *auth.DigestAuth
	log        logger.Logger
}

// NewServer creates a Server from its three dependencies.
func NewServer(handler *Handler, digestAuth *auth.DigestAuth, log logger.Logger) *Server {
	return &Server{
		handler:    handler,
		digestAuth: digestAuth,
		log:        log,
	}
}

// Router returns the http.Handler for the CWMP endpoint. It mounts the session
// Handler at POST /acs, wrapped with Digest auth and a 1 MB body cap.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Wrap the core handler: enforce Digest auth then delegate.
	cwmpHandler := s.digestAuth.Middleware(http.HandlerFunc(s.handler.ServeHTTP))
	cwmpHandler = limitBody(cwmpHandler, 1<<20) // 1 MB

	mux.Handle("/acs", cwmpHandler)
	mux.Handle("/acs/", cwmpHandler)

	s.log.Debug("CWMP: Router mounted at /acs")
	return mux
}

// limitBody wraps next so that request bodies larger than maxBytes are
// rejected with 413 before the handler reads anything.
func limitBody(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}
