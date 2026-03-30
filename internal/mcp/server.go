package mcp

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	catpkg "dops/internal/catalog"
	"dops/internal/domain"
	"dops/internal/executor"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerConfig holds dependencies for the MCP server.
type ServerConfig struct {
	Version  string
	DopsHome string
	Catalogs []catpkg.CatalogWithRunbooks
	Runner   executor.Runner
	Config   *domain.Config
	MaxRisk  domain.RiskLevel
}

// Server wraps the MCP SDK server with dops catalog integration.
type Server struct {
	srv      *mcpsdk.Server
	catalogs []catpkg.CatalogWithRunbooks
	runner   executor.Runner
	cfg      *domain.Config
	dopsHome string
}

// NewServer creates a new MCP server with tools and resources from the catalog.
func NewServer(sc ServerConfig) *Server {
	srv := mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: "dops", Version: sc.Version},
		nil,
	)

	s := &Server{
		srv:      srv,
		catalogs: sc.Catalogs,
		runner:   sc.Runner,
		cfg:      sc.Config,
		dopsHome: sc.DopsHome,
	}

	s.registerTools(sc.MaxRisk)
	s.registerResources()
	s.registerSchemaResources()
	s.registerPrompts()

	return s
}

func (s *Server) registerTools(maxRisk domain.RiskLevel) {
	for _, c := range s.catalogs {
		cat := c.Catalog
		for _, rb := range c.Runbooks {
			rb := rb // capture
			cat := cat

			// Skip runbooks above the allowed risk level.
			if maxRisk != "" && rb.RiskLevel.Exceeds(maxRisk) {
				continue
			}

			resolved := make(map[string]string) // TODO: pass resolved vars
			schema := RunbookToInputSchema(rb, resolved)

			s.srv.AddTool(
				&mcpsdk.Tool{
					Name:        rb.ID,
					Description: RunbookToDescription(rb),
					InputSchema: schema,
				},
				s.makeToolHandler(rb, cat),
			)
		}
	}
}

func (s *Server) makeToolHandler(rb domain.Runbook, cat domain.Catalog) func(context.Context, *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		// Parse arguments.
		var args map[string]any
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return mkerr("invalid arguments: " + err.Error()), nil
		}

		result, err := HandleToolCall(ctx, rb, cat, s.cfg, s.runner, args, nil /* TODO: wire progress notifications */)
		if err != nil {
			return mkerr(err.Error()), nil
		}

		// Format as text.
		var sb strings.Builder
		sb.WriteString(result.Output)
		sb.WriteString(fmt.Sprintf("\n\n---\nExit code: %d | Duration: %s | Lines: %d",
			result.ExitCode, result.Duration, result.OutputLines))
		if result.LogPath != "" {
			sb.WriteString(fmt.Sprintf(" | Log: %s", result.LogPath))
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: sb.String()}},
			IsError: result.ExitCode != 0,
		}, nil
	}
}

func (s *Server) registerResources() {
	// Static resource: catalog listing.
	s.srv.AddResource(
		&mcpsdk.Resource{
			URI:         "dops://catalog",
			Name:        "Runbook Catalog",
			Description: "List of all available runbooks",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
			text, err := CatalogListJSON(s.catalogs)
			if err != nil {
				return nil, err
			}
			return &mcpsdk.ReadResourceResult{
				Contents: []*mcpsdk.ResourceContents{
					{
						URI:      "dops://catalog",
						MIMEType: "application/json",
						Text:     text,
					},
				},
			}, nil
		},
	)

	// Template resource: individual runbook detail.
	s.srv.AddResourceTemplate(
		&mcpsdk.ResourceTemplate{
			URITemplate: "dops://catalog/{id}",
			Name:        "Runbook Detail",
			Description: "Detailed information about a specific runbook",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
			uri := req.Params.URI
			id := strings.TrimPrefix(uri, "dops://catalog/")

			for _, c := range s.catalogs {
				for _, rb := range c.Runbooks {
					if rb.ID == id {
						text, err := RunbookDetailJSON(rb, c.Catalog)
						if err != nil {
							return nil, err
						}
						return &mcpsdk.ReadResourceResult{
							Contents: []*mcpsdk.ResourceContents{
								{
									URI:      uri,
									MIMEType: "application/json",
									Text:     text,
								},
							},
						}, nil
					}
				}
			}
			return nil, fmt.Errorf("runbook not found: %s", id)
		},
	)
}

func (s *Server) registerSchemaResources() {
	s.srv.AddResource(
		&mcpsdk.Resource{
			URI:         "dops://schema/runbook",
			Name:        "Runbook YAML Schema",
			Description: "Complete schema reference for runbook.yaml files",
			MIMEType:    "text/markdown",
		},
		func(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
			return &mcpsdk.ReadResourceResult{
				Contents: []*mcpsdk.ResourceContents{
					{URI: "dops://schema/runbook", MIMEType: "text/markdown", Text: runbookSchema},
				},
			}, nil
		},
	)

	s.srv.AddResource(
		&mcpsdk.Resource{
			URI:         "dops://schema/shell-style",
			Name:        "Shell Script Style Guide",
			Description: "POSIX-compatible shell style guide for dops runbook scripts",
			MIMEType:    "text/markdown",
		},
		func(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
			return &mcpsdk.ReadResourceResult{
				Contents: []*mcpsdk.ResourceContents{
					{URI: "dops://schema/shell-style", MIMEType: "text/markdown", Text: shellStyleGuide},
				},
			}, nil
		},
	)
}

// ServeStdio starts the MCP server on stdin/stdout.
func (s *Server) ServeStdio(ctx context.Context) error {
	_, err := s.srv.Connect(ctx, &mcpsdk.StdioTransport{}, nil)
	if err != nil {
		return fmt.Errorf("connect mcp stdio: %w", err)
	}
	<-ctx.Done()
	return nil
}

// ServeHTTP starts the MCP server on an HTTP port with gzip support.
func (s *Server) ServeHTTP(ctx context.Context, addr string) error {
	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return s.srv
	}, nil)
	wrapped := gzipMiddleware(handler)
	server := &http.Server{Addr: addr, Handler: wrapped, ReadHeaderTimeout: 60 * time.Second}

	go func() {
		<-ctx.Done()
		_ = server.Close() // best-effort shutdown on context cancellation
	}()

	return server.ListenAndServe()
}

// gzipMiddleware wraps an HTTP handler with gzip compression.
// Skips gzip for SSE/streaming responses to avoid buffering delays.
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip gzip if client doesn't accept it.
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip gzip for SSE requests — gzip buffering breaks event streaming.
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzw := &gzipResponseWriter{ResponseWriter: w, Writer: gz}
		next.ServeHTTP(gzw, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	io.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func mkerr(msg string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: msg}},
		IsError: true,
	}
}
