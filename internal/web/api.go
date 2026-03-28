package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"dops/internal/domain"
	"dops/internal/executor"
	"dops/internal/vars"
)

type api struct {
	deps       ServerDeps
	executions *executionStore
}

func newAPI(deps ServerDeps) *api {
	return &api{
		deps:       deps,
		executions: newExecutionStore(),
	}
}

func (a *api) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/catalogs", a.handleListCatalogs)
	mux.HandleFunc("GET /api/runbooks/{id}", a.handleGetRunbook)
	mux.HandleFunc("POST /api/runbooks/{id}/execute", a.handleExecuteRunbook)
	mux.HandleFunc("GET /api/executions/{id}/stream", a.handleStreamExecution)
	mux.HandleFunc("POST /api/executions/{id}/cancel", a.handleCancelExecution)
	mux.HandleFunc("GET /api/theme", a.handleGetTheme)
}

// --- Catalog & Runbook Endpoints ---

type catalogResponse struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name,omitempty"`
	Path        string            `json:"path"`
	Active      bool              `json:"active"`
	Runbooks    []runbookSummary  `json:"runbooks"`
}

type runbookSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RiskLevel   string `json:"risk_level"`
	ParamCount  int    `json:"param_count"`
}

type runbookDetail struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Aliases     []string           `json:"aliases,omitempty"`
	Description string             `json:"description"`
	Version     string             `json:"version"`
	RiskLevel   string             `json:"risk_level"`
	Script      string             `json:"script"`
	Parameters  []domain.Parameter `json:"parameters"`
}

func (a *api) handleListCatalogs(w http.ResponseWriter, _ *http.Request) {
	var result []catalogResponse
	for _, cwr := range a.deps.Catalogs {
		cat := catalogResponse{
			Name:        cwr.Catalog.Name,
			DisplayName: cwr.Catalog.DisplayName,
			Path:        cwr.Catalog.Path,
			Active:      cwr.Catalog.Active,
		}
		for _, rb := range cwr.Runbooks {
			cat.Runbooks = append(cat.Runbooks, runbookSummary{
				ID:          rb.ID,
				Name:        rb.Name,
				Description: rb.Description,
				RiskLevel:   string(rb.RiskLevel),
				ParamCount:  len(rb.Parameters),
			})
		}
		result = append(result, cat)
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *api) handleGetRunbook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	rb, _, err := a.findRunbook(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, runbookDetail{
		ID:          rb.ID,
		Name:        rb.Name,
		Aliases:     rb.Aliases,
		Description: rb.Description,
		Version:     rb.Version,
		RiskLevel:   string(rb.RiskLevel),
		Script:      rb.Script,
		Parameters:  rb.Parameters,
	})
}

// --- Execution Endpoints ---

type executeRequest struct {
	Params map[string]string `json:"params"`
}

type executeResponse struct {
	ExecutionID string `json:"execution_id"`
}

func (a *api) handleExecuteRunbook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	rb, cat, err := a.findRunbook(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	var req executeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Resolve saved vars and merge with request params.
	resolver := vars.NewDefaultResolver()
	resolved := resolver.Resolve(a.deps.Config, cat.Name, rb.Name, rb.Parameters)
	for k, v := range req.Params {
		resolved[k] = v
	}

	// Build env.
	env := make(map[string]string, len(resolved))
	for k, v := range resolved {
		env[strings.ToUpper(k)] = v
	}

	// Resolve script path.
	catPath := cat.RunbookRoot()
	scriptPath := catPath + "/" + rb.Name + "/" + rb.Script

	// Start execution.
	exec := a.executions.start(scriptPath, env, a.deps.Runner)

	writeJSON(w, http.StatusAccepted, executeResponse{ExecutionID: exec.id})
}

func (a *api) handleStreamExecution(w http.ResponseWriter, r *http.Request) {
	execID := r.PathValue("id")
	exec, ok := a.executions.get(execID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "execution not found"})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher.Flush()

	// Stream existing + new lines.
	lastIdx := 0
	for {
		exec.mu.Lock()
		lines := exec.lines[lastIdx:]
		done := exec.done
		exitErr := exec.exitErr
		exec.mu.Unlock()

		for _, line := range lines {
			fmt.Fprintf(w, "data: %s\n\n", line)
			lastIdx++
		}
		if len(lines) > 0 {
			flusher.Flush()
		}

		if done {
			status := "success"
			if exitErr != nil {
				status = "error: " + exitErr.Error()
			}
			fmt.Fprintf(w, "event: done\ndata: %s\n\n", status)
			flusher.Flush()
			return
		}

		// Wait for new data or client disconnect.
		select {
		case <-r.Context().Done():
			return
		case <-exec.notify:
			// New data available.
		}
	}
}

func (a *api) handleCancelExecution(w http.ResponseWriter, r *http.Request) {
	execID := r.PathValue("id")
	exec, ok := a.executions.get(execID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "execution not found"})
		return
	}

	exec.cancel()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// --- Theme Endpoint ---

type themeResponse struct {
	Name   string            `json:"name"`
	Colors map[string]string `json:"colors"`
}

func (a *api) handleGetTheme(w http.ResponseWriter, _ *http.Request) {
	if a.deps.Theme == nil {
		writeJSON(w, http.StatusOK, themeResponse{Name: "default", Colors: map[string]string{}})
		return
	}
	writeJSON(w, http.StatusOK, themeResponse{
		Name:   a.deps.Theme.Name,
		Colors: a.deps.Theme.Colors,
	})
}

// --- Helpers ---

func (a *api) findRunbook(id string) (*domain.Runbook, *domain.Catalog, error) {
	if domain.ValidateRunbookID(id) == nil {
		rb, cat, err := a.deps.Loader.FindByID(id)
		if err == nil {
			return rb, cat, nil
		}
	}
	rb, cat, err := a.deps.Loader.FindByAlias(id)
	if err != nil {
		return nil, nil, fmt.Errorf("runbook %q not found", id)
	}
	return rb, cat, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

// --- Execution Store ---

type execution struct {
	id      string
	lines   []string
	done    bool
	exitErr error
	cancel  func()
	notify  chan struct{}
	mu      sync.Mutex
}

type executionStore struct {
	mu    sync.Mutex
	execs map[string]*execution
	seq   int
}

func newExecutionStore() *executionStore {
	return &executionStore{execs: make(map[string]*execution)}
}

func (s *executionStore) start(scriptPath string, env map[string]string, runner executor.Runner) *execution {
	s.mu.Lock()
	s.seq++
	id := fmt.Sprintf("exec-%d", s.seq)
	s.mu.Unlock()

	ctx, cancel := contextWithCancel()
	exec := &execution{
		id:     id,
		cancel: cancel,
		notify: make(chan struct{}, 1),
	}

	s.mu.Lock()
	s.execs[id] = exec
	s.mu.Unlock()

	lines, errs := runner.Run(ctx, scriptPath, env)

	go func() {
		for line := range lines {
			exec.mu.Lock()
			exec.lines = append(exec.lines, line.Text)
			exec.mu.Unlock()

			// Signal waiters non-blocking.
			select {
			case exec.notify <- struct{}{}:
			default:
			}
		}

		err := <-errs
		exec.mu.Lock()
		exec.done = true
		exec.exitErr = err
		exec.mu.Unlock()

		// Final signal.
		select {
		case exec.notify <- struct{}{}:
		default:
		}
	}()

	return exec
}

func (s *executionStore) get(id string) (*execution, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	exec, ok := s.execs[id]
	return exec, ok
}
