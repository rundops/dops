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

	m := New(WizardConfig{Runbook: rb, Catalog: cat, Resolved: resolved})

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

	m := New(WizardConfig{Runbook: rb, Catalog: cat, Resolved: resolved})
	view := m.View()

	if !strings.Contains(view, "dops run") {
		t.Error("view should show command header")
	}
}

func TestNewModel_FooterHints(t *testing.T) {
	rb := testRunbook()
	cat := domain.Catalog{Name: "default"}
	resolved := map[string]string{"region": "us-east-1"}

	m := New(WizardConfig{Runbook: rb, Catalog: cat, Resolved: resolved})
	hints := m.FooterHints()

	if hints == "" {
		t.Error("footer hints should not be empty")
	}
}

func TestSkipSavedFields_PartialPrefill(t *testing.T) {
	rb := testRunbook()
	cat := domain.Catalog{Name: "default"}
	// Pre-fill region and env, leave namespace and dry_run empty.
	resolved := map[string]string{
		"region": "us-east-1",
		"env":    "prod",
	}

	m := New(WizardConfig{Runbook: rb, Catalog: cat, Resolved: resolved})

	// region (index 0) should be skipped.
	if !m.skipped[0] {
		t.Error("region (index 0) should be skipped — it has a saved value")
	}

	// Current field should be namespace (index 1) — first non-prefilled.
	if m.current != 1 {
		t.Errorf("current field should be 1 (namespace), got %d", m.current)
	}

	// env (index 3) not yet skipped — it will be skipped when we advance past dry_run.
	if m.skipped[3] {
		t.Error("env (index 3) should not be skipped yet — advance hasn't reached it")
	}

	// Skipped count should be 1 (just region so far).
	if m.SkippedCount() != 1 {
		t.Errorf("expected 1 skipped field, got %d", m.SkippedCount())
	}

	// View should show the skipped summary.
	view := m.View()
	if !strings.Contains(view, "Applied 1 saved value") {
		t.Error("view should show 'Applied 1 saved value' message")
	}
}

func TestSkipSavedFields_AllPrefilled(t *testing.T) {
	rb := testRunbook()
	cat := domain.Catalog{Name: "default"}
	resolved := map[string]string{
		"region":    "us-east-1",
		"namespace": "platform",
		"dry_run":   "false",
		"env":       "prod",
	}

	m := New(WizardConfig{Runbook: rb, Catalog: cat, Resolved: resolved})

	// All fields should be skipped.
	if m.SkippedCount() != 4 {
		t.Errorf("expected 4 skipped fields, got %d", m.SkippedCount())
	}

	// current should be past all fields.
	if m.current < len(m.params) {
		t.Errorf("current should be >= len(params), got %d", m.current)
	}

	// Init should return a submit command.
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a submit command when all fields are skipped")
	}
}

func TestSkipSavedFields_PromptAll(t *testing.T) {
	rb := testRunbook()
	cat := domain.Catalog{Name: "default"}
	resolved := map[string]string{
		"region":    "us-east-1",
		"namespace": "platform",
		"dry_run":   "false",
		"env":       "prod",
	}

	m := New(WizardConfig{Runbook: rb, Catalog: cat, Resolved: resolved, PromptAll: true})

	// No fields should be skipped when promptAll is true.
	if m.SkippedCount() != 0 {
		t.Errorf("expected 0 skipped fields with promptAll, got %d", m.SkippedCount())
	}

	// Should start at first field.
	if m.current != 0 {
		t.Errorf("current should be 0, got %d", m.current)
	}
}

func TestSkipSavedFields_CtrlEHint(t *testing.T) {
	rb := testRunbook()
	cat := domain.Catalog{Name: "default"}
	resolved := map[string]string{
		"region": "us-east-1",
		"env":    "prod",
	}

	m := New(WizardConfig{Runbook: rb, Catalog: cat, Resolved: resolved})
	hints := m.FooterHints()

	if !strings.Contains(hints, "ctrl+e") {
		t.Error("footer hints should mention ctrl+e when fields are skipped")
	}
}

