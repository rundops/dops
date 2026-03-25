package wizard

import (
	"dops/internal/domain"
	"strings"
	"testing"
)

func testRunbook() domain.Runbook {
	return domain.Runbook{
		ID:   "default.hello-world",
		Name: "hello-world",
		Parameters: []domain.Parameter{
			{Name: "region", Type: domain.ParamString, Required: true, Scope: "global"},
			{Name: "namespace", Type: domain.ParamString, Required: true, Scope: "catalog"},
			{Name: "dry_run", Type: domain.ParamBoolean, Required: false, Scope: "runbook", Default: false},
			{Name: "env", Type: domain.ParamSelect, Required: true, Scope: "global", Options: []string{"dev", "staging", "prod"}},
		},
	}
}

func TestShouldSkip_AllResolved(t *testing.T) {
	rb := testRunbook()
	resolved := map[string]string{
		"region":    "us-east-1",
		"namespace": "platform",
		"dry_run":   "false",
		"env":       "prod",
	}

	if !ShouldSkip(rb.Parameters, resolved) {
		t.Error("should skip when all required params are resolved")
	}
}

func TestShouldSkip_MissingRequired(t *testing.T) {
	rb := testRunbook()
	resolved := map[string]string{
		"region": "us-east-1",
		// namespace is missing and required
		"env": "prod",
	}

	if ShouldSkip(rb.Parameters, resolved) {
		t.Error("should not skip when required param is missing")
	}
}

func TestShouldSkip_OptionalMissing(t *testing.T) {
	rb := testRunbook()
	resolved := map[string]string{
		"region":    "us-east-1",
		"namespace": "platform",
		"env":       "prod",
		// dry_run is missing but optional
	}

	if !ShouldSkip(rb.Parameters, resolved) {
		t.Error("should skip when only optional params are missing")
	}
}

func TestMissingParams_AllResolved(t *testing.T) {
	rb := testRunbook()
	resolved := map[string]string{
		"region": "us-east-1", "namespace": "platform", "dry_run": "false", "env": "prod",
	}

	missing := missingParams(rb.Parameters, resolved)
	if len(missing) != 0 {
		t.Errorf("expected 0 missing, got %d", len(missing))
	}
}

func TestMissingParams_SomeMissing(t *testing.T) {
	rb := testRunbook()
	resolved := map[string]string{
		"region": "us-east-1",
	}

	missing := missingParams(rb.Parameters, resolved)

	names := make(map[string]bool)
	for _, p := range missing {
		names[p.Name] = true
	}

	if !names["namespace"] {
		t.Error("namespace should be in missing list")
	}
	if !names["env"] {
		t.Error("env should be in missing list")
	}
}

func TestBuildCommand_Format(t *testing.T) {
	rb := testRunbook()
	params := map[string]string{
		"region":    "us-east-1",
		"namespace": "platform",
		"env":       "prod",
	}

	cmd := BuildCommand(rb, params)
	if cmd == "" {
		t.Fatal("command should not be empty")
	}

	if !strings.Contains(cmd, "default.hello-world") {
		t.Error("command should contain runbook ID")
	}
}

func TestNewModel_WithmissingParams(t *testing.T) {
	rb := testRunbook()
	cat := domain.Catalog{Name: "default"}
	resolved := map[string]string{
		"region": "us-east-1",
	}

	m := New(rb, cat, resolved)

	if m.runbook.ID != "default.hello-world" {
		t.Errorf("runbook = %q", m.runbook.ID)
	}

	// Should have missing params to collect.
	if len(m.params) == 0 {
		t.Error("should have missing params")
	}
}

func TestNewModel_CommandHeader(t *testing.T) {
	rb := testRunbook()
	cat := domain.Catalog{Name: "default"}
	resolved := map[string]string{"region": "us-east-1"}

	m := New(rb, cat, resolved)
	view := m.View()

	if !strings.Contains(view, "dops run") {
		t.Error("view should show command header")
	}
}

func TestNewModel_FooterHints(t *testing.T) {
	rb := testRunbook()
	cat := domain.Catalog{Name: "default"}
	resolved := map[string]string{"region": "us-east-1"}

	m := New(rb, cat, resolved)
	hints := m.FooterHints()

	if hints == "" {
		t.Error("footer hints should not be empty")
	}
}

