package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dops/internal/domain"
	"dops/internal/theme"

	catpkg "dops/internal/catalog"
)

func testDeps() ServerDeps {
	return ServerDeps{
		Config: &domain.Config{
			Theme:    "github",
			Defaults: domain.Defaults{MaxRiskLevel: domain.RiskHigh},
		},
		Catalogs: []catpkg.CatalogWithRunbooks{
			{
				Catalog: domain.Catalog{Name: "default", DisplayName: "My Ops", Active: true},
				Runbooks: []domain.Runbook{
					{
						ID:          "default.hello-world",
						Name:        "hello-world",
						Aliases:     []string{"hw"},
						Description: "Says hello",
						Version:     "1.0.0",
						RiskLevel:   domain.RiskLow,
						Script:      "./script.sh",
						Parameters: []domain.Parameter{
							{Name: "greeting", Type: domain.ParamString, Required: true, Description: "The greeting"},
						},
					},
				},
			},
		},
		Theme: &theme.ResolvedTheme{
			Name:   "github",
			Colors: map[string]string{"primary": "#58a6ff", "background": "#0d1117"},
		},
		Port: 0,
	}
}

func setupTestAPI(t *testing.T) *http.ServeMux {
	t.Helper()
	deps := testDeps()
	a := newAPI(deps)
	mux := http.NewServeMux()
	a.registerRoutes(mux)
	return mux
}

func TestListCatalogs(t *testing.T) {
	mux := setupTestAPI(t)
	req := httptest.NewRequest("GET", "/api/catalogs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var catalogs []catalogResponse
	if err := json.Unmarshal(w.Body.Bytes(), &catalogs); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(catalogs) != 1 {
		t.Fatalf("expected 1 catalog, got %d", len(catalogs))
	}
	if catalogs[0].Name != "default" {
		t.Errorf("catalog name = %q", catalogs[0].Name)
	}
	if catalogs[0].DisplayName != "My Ops" {
		t.Errorf("display name = %q", catalogs[0].DisplayName)
	}
	if len(catalogs[0].Runbooks) != 1 {
		t.Fatalf("expected 1 runbook, got %d", len(catalogs[0].Runbooks))
	}
	if catalogs[0].Runbooks[0].ID != "default.hello-world" {
		t.Errorf("runbook id = %q", catalogs[0].Runbooks[0].ID)
	}
}

func TestGetTheme(t *testing.T) {
	mux := setupTestAPI(t)
	req := httptest.NewRequest("GET", "/api/theme", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var theme themeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &theme); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if theme.Name != "github" {
		t.Errorf("theme name = %q", theme.Name)
	}
	if theme.Colors["primary"] != "#58a6ff" {
		t.Errorf("primary = %q", theme.Colors["primary"])
	}
}

func TestExecutionStoreEviction(t *testing.T) {
	store := newExecutionStore()

	// Seed the store with maxCompleted+10 completed executions.
	for i := 0; i < maxCompleted+10; i++ {
		id := fmt.Sprintf("exec-%d", i+1)
		store.execs[id] = &execution{
			id:          id,
			done:        true,
			completedAt: time.Now().Add(time.Duration(i) * time.Second),
			notify:      make(chan struct{}, 1),
		}
	}
	store.seq = maxCompleted + 10

	if len(store.execs) != maxCompleted+10 {
		t.Fatalf("setup: expected %d execs, got %d", maxCompleted+10, len(store.execs))
	}

	// Trigger eviction by calling evict under lock.
	store.mu.Lock()
	store.evict()
	store.mu.Unlock()

	if len(store.execs) != maxCompleted {
		t.Fatalf("after eviction: expected %d execs, got %d", maxCompleted, len(store.execs))
	}

	// The 10 oldest (exec-1 through exec-10) should have been removed.
	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("exec-%d", i)
		if _, ok := store.execs[id]; ok {
			t.Errorf("expected %s to be evicted", id)
		}
	}

	// The newest should still be present.
	newest := fmt.Sprintf("exec-%d", maxCompleted+10)
	if _, ok := store.execs[newest]; !ok {
		t.Errorf("expected %s to still be present", newest)
	}
}

func TestExecutionStoreEvictionKeepsRunning(t *testing.T) {
	store := newExecutionStore()

	// Add maxCompleted+5 completed and 3 still-running executions.
	for i := 0; i < maxCompleted+5; i++ {
		id := fmt.Sprintf("exec-done-%d", i)
		store.execs[id] = &execution{
			id:          id,
			done:        true,
			completedAt: time.Now().Add(time.Duration(i) * time.Second),
			notify:      make(chan struct{}, 1),
		}
	}
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("exec-running-%d", i)
		store.execs[id] = &execution{
			id:     id,
			done:   false,
			notify: make(chan struct{}, 1),
		}
	}

	store.mu.Lock()
	store.evict()
	store.mu.Unlock()

	// Running executions must survive eviction.
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("exec-running-%d", i)
		if _, ok := store.execs[id]; !ok {
			t.Errorf("running execution %s was incorrectly evicted", id)
		}
	}

	// Completed count should be exactly maxCompleted.
	completed := 0
	for _, e := range store.execs {
		if e.done {
			completed++
		}
	}
	if completed != maxCompleted {
		t.Errorf("completed count = %d, want %d", completed, maxCompleted)
	}
}
