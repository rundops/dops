package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
