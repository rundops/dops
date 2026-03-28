package web

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"dops/internal/domain"
	"dops/internal/executor"
	"dops/internal/theme"
	"dops/internal/vault"

	catpkg "dops/internal/catalog"
)

// ServerDeps holds all dependencies for the web server.
type ServerDeps struct {
	Config   *domain.Config
	Catalogs []catpkg.CatalogWithRunbooks
	Loader   *catpkg.DiskCatalogLoader
	Runner   executor.Runner
	Vault    *vault.Vault
	Theme    *theme.ResolvedTheme
	Port     int
}

// Server manages the HTTP server for the web UI.
type Server struct {
	deps   ServerDeps
	server *http.Server
}

// NewServer creates a configured web server.
func NewServer(deps ServerDeps) *Server {
	mux := http.NewServeMux()

	api := newAPI(deps)
	api.registerRoutes(mux)

	// SPA catch-all (must be last).
	mux.Handle("/", SPAHandler())

	return &Server{
		deps: deps,
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", deps.Port),
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Start begins listening. Returns immediately; call Shutdown to stop.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", s.deps.Port, err)
	}

	log.Printf("Web UI available at http://localhost:%d", s.deps.Port)

	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()

	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
