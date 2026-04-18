package wiring

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	l "github.com/raykavin/helix-acs/internal/logger"
)

// NewHTTPServer returns an http.Server with conservative production timeouts.
func NewHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// StartServer runs ListenAndServe in a goroutine and forwards non-closed errors to errCh.
func StartServer(srv *http.Server, name string, log l.Logger, errCh chan<- error) {
	log.WithField("addr", srv.Addr).Infof("%s server listening", name)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errCh <- fmt.Errorf("%s server: %w", name, err)
	}
}

// ShutdownServers attempts a graceful shutdown of all provided servers within a
// 30-second window. Shutdown errors are logged but not returned.
func ShutdownServers(log l.Logger, servers ...*http.Server) error {
	log.Info("shutting down servers (30s timeout)")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, srv := range servers {
		if err := srv.Shutdown(ctx); err != nil {
			log.WithError(err).Errorf("error shutting down server on %s", srv.Addr)
		}
	}
	return nil
}

// ServeHTTP starts an HTTP server on addr and blocks until ctx is cancelled or a
// server error occurs. The server is shut down gracefully on exit.
func ServeHTTP(ctx context.Context, addr, name string, handler http.Handler, log l.Logger) error {
	srv := NewHTTPServer(addr, handler)
	serverErr := make(chan error, 1)

	go StartServer(srv, name, log, serverErr)

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-serverErr:
		log.WithError(err).Error("server error")
	}

	return ShutdownServers(log, srv)
}
