package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sort"
	"sync"
	"time"

	"dops/internal/adapters"
	"dops/internal/domain"
	"dops/internal/executor"
	"dops/internal/history"
	"dops/internal/theme"
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
	mux.HandleFunc("GET /api/themes", a.handleListThemes)
	mux.HandleFunc("PUT /api/theme", a.handleSetTheme)
	mux.HandleFunc("GET /api/history", a.handleListHistory)
	mux.HandleFunc("GET /api/history/{id}", a.handleGetHistory)
	mux.HandleFunc("GET /api/history/{id}/log", a.handleGetHistoryLog)
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
	SavedValues map[string]string  `json:"saved_values,omitempty"`
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

	rb, cat, err := a.findRunbook(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	// Resolve saved values for pre-filling the form.
	resolver := vars.NewDefaultResolver()
	saved := resolver.Resolve(a.deps.Config, cat.Name, rb.Name, rb.Parameters)

	// Mask secret values — the frontend should know a saved value exists
	// but must not see the actual value.
	for _, p := range rb.Parameters {
		if p.Secret {
			if _, ok := saved[p.Name]; ok {
				saved[p.Name] = "••••••••"
			}
		}
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
		SavedValues: saved,
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
	catPath := adapters.ExpandHome(cat.RunbookRoot())
	scriptPath := filepath.Join(catPath, rb.Name, rb.Script)

	// Start execution with history recording.
	exec := a.executions.start(scriptPath, env, a.deps.Runner)

	if a.deps.History != nil {
		rec := domain.NewExecutionRecord(rb.ID, rb.Name, cat.Name, domain.ExecWeb)
		rec.Parameters = make(map[string]string, len(resolved))
		for k, v := range resolved {
			rec.Parameters[k] = v
		}
		var secretNames []string
		for _, p := range rb.Parameters {
			if p.Secret {
				secretNames = append(secretNames, p.Name)
			}
		}
		rec.MaskSecrets(secretNames)

		histStore := a.deps.History
		exec.onComplete = func(lines []string, lastLine string, err error) {
			exitCode := 0
			if err != nil {
				exitCode = 1
			}
			rec.Complete(exitCode, len(lines), lastLine)
			_ = histStore.ArchiveLog(rec, lines)
			_ = histStore.Record(rec)
		}
	}

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

func (a *api) handleListThemes(w http.ResponseWriter, _ *http.Request) {
	active := ""
	if a.deps.Theme != nil {
		active = a.deps.Theme.Name
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"active": active,
		"themes": theme.BundledNames(),
	})
}

type setThemeRequest struct {
	Name string `json:"name"`
}

func (a *api) handleSetTheme(w http.ResponseWriter, r *http.Request) {
	var req setThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	tf, err := a.deps.ThemeLoader.Load(req.Name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("theme %q not found", req.Name)})
		return
	}

	resolved, err := theme.Resolve(tf, a.deps.IsDark)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve theme"})
		return
	}

	// Update in-memory state.
	a.deps.Theme = resolved
	a.deps.Config.Theme = req.Name

	// Persist to config file (skip in demo mode).
	if a.deps.ConfigStore != nil && !a.deps.Demo {
		if err := a.deps.ConfigStore.Save(a.deps.Config); err != nil {
			log.Printf("failed to save theme config: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, themeResponse{
		Name:   resolved.Name,
		Colors: resolved.Colors,
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

// --- History Endpoints ---

func (a *api) handleListHistory(w http.ResponseWriter, r *http.Request) {
	if a.deps.History == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	opts := history.ListOpts{Limit: 50}
	if rb := r.URL.Query().Get("runbook"); rb != "" {
		opts.RunbookID = rb
	}
	if st := r.URL.Query().Get("status"); st != "" {
		opts.Status = domain.ExecStatus(st)
	}

	records, err := a.deps.History.List(opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if records == nil {
		records = []*domain.ExecutionRecord{}
	}
	writeJSON(w, http.StatusOK, records)
}

func (a *api) handleGetHistoryLog(w http.ResponseWriter, r *http.Request) {
	if a.deps.History == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "history not available"})
		return
	}

	id := r.PathValue("id")
	rec, err := a.deps.History.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	lines, available := history.ReadLog(rec.LogPath)
	if !available {
		writeJSON(w, http.StatusOK, map[string]any{"lines": []string{}, "available": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines, "available": true})
}

func (a *api) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	if a.deps.History == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "history not available"})
		return
	}

	id := r.PathValue("id")
	rec, err := a.deps.History.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

// --- Execution Store ---

// maxCompleted is the number of finished executions to keep.
// When exceeded, the oldest completed executions are evicted.
const maxCompleted = 100

type execution struct {
	id          string
	lines       []string
	done        bool
	completedAt time.Time
	exitErr     error
	cancel      func()
	notify      chan struct{}
	onComplete  func(lines []string, lastLine string, err error) // called when execution finishes
	mu          sync.Mutex
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
	s.evict()
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
		exec.completedAt = time.Now()
		exec.exitErr = err
		lineCount := len(exec.lines)
		lastLine := ""
		for i := lineCount - 1; i >= 0; i-- {
			if strings.TrimSpace(exec.lines[i]) != "" {
				lastLine = strings.TrimSpace(exec.lines[i])
				break
			}
		}
		exec.mu.Unlock()

		if exec.onComplete != nil {
			linesCopy := make([]string, len(exec.lines))
			copy(linesCopy, exec.lines)
			exec.onComplete(linesCopy, lastLine, err)
		}

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

// evict removes the oldest completed executions when the count exceeds
// maxCompleted. Must be called with s.mu held.
func (s *executionStore) evict() {
	var completed []*execution
	for _, e := range s.execs {
		e.mu.Lock()
		done := e.done
		e.mu.Unlock()
		if done {
			completed = append(completed, e)
		}
	}

	if len(completed) <= maxCompleted {
		return
	}

	sort.Slice(completed, func(i, j int) bool {
		return completed[i].completedAt.Before(completed[j].completedAt)
	})

	toRemove := len(completed) - maxCompleted
	for _, e := range completed[:toRemove] {
		delete(s.execs, e.id)
	}
}
