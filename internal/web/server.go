package web

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"dops/internal/config"
	"dops/internal/domain"
	"dops/internal/executor"
	histpkg "dops/internal/history"
	"dops/internal/theme"

	catpkg "dops/internal/catalog"
)

// CatalogLoader is the subset of catalog loading behaviour the web server needs.
type CatalogLoader interface {
	FindByID(id string) (*domain.Runbook, *domain.Catalog, error)
	FindByAlias(alias string) (*domain.Runbook, *domain.Catalog, error)
}

// ServerDeps holds all dependencies for the web server.
type ServerDeps struct {
	Config      *domain.Config
	ConfigStore config.ConfigStore
	Catalogs    []catpkg.CatalogWithRunbooks
	Loader      CatalogLoader
	Runner      executor.Runner
	Vault       domain.VaultStore
	History     histpkg.ExecutionStore
	Theme       *theme.ResolvedTheme
	ThemeLoader theme.ThemeLoader
	IsDark      bool
	Port        int
	Demo        bool
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
