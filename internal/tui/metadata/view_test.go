package metadata

import (
	"dops/internal/domain"
	"dops/internal/testutil"
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	rb := &domain.Runbook{
		ID:          "default.hello-world",
		Name:        "hello-world",
		Description: "Prints a hello world message",
		Version:     "1.0.0",
		RiskLevel:   domain.RiskLow,
	}

	cat := &domain.Catalog{Name: "default", Path: "~/.dops/catalogs/default"}
	out := Render(rb, cat, 40, false, testutil.TestStyles())

	if !strings.Contains(out, "hello-world") {
		t.Error("output should contain runbook name")
	}
	if !strings.Contains(out, "1.0.0") {
		t.Error("output should contain version")
	}
	if !strings.Contains(out, "low") {
		t.Error("output should contain risk level")
	}
	if !strings.Contains(out, "Prints a hello world message") {
		t.Error("output should contain description")
	}
	if !strings.Contains(out, "catalogs/default") {
		t.Error("output should contain local path")
	}
}

func TestRender_PathTruncation(t *testing.T) {
	rb := &domain.Runbook{
		Name:      "hello-world",
		Version:   "1.0.0",
		RiskLevel: domain.RiskLow,
	}
	cat := &domain.Catalog{Name: "default", Path: "~/.dops/catalogs/default"}

	// Wide enough to show full path.
	wide := Render(rb, cat, 60, false, testutil.TestStyles())
	if !strings.Contains(wide, "runbook.yaml") {
		t.Error("wide render should show full path")
	}

	// Narrow should truncate with ellipsis.
	narrow := Render(rb, cat, 30, false, testutil.TestStyles())
	if strings.Contains(narrow, "runbook.yaml") {
		t.Error("narrow render should truncate path")
	}
	if !strings.Contains(narrow, "…") {
		t.Error("narrow render should show ellipsis")
	}
}

func TestRender_GitCatalog(t *testing.T) {
	rb := &domain.Runbook{
		Name:      "drain-node",
		Version:   "2.1.0",
		RiskLevel: domain.RiskHigh,
	}
	cat := &domain.Catalog{Name: "public", URL: "https://github.com/org/public-catalog"}
	out := Render(rb, cat, 50, false, testutil.TestStyles())

	if !strings.Contains(out, "public-catalog") {
		t.Error("output should contain catalog URL")
	}
}

func TestRender_CopiedFlash(t *testing.T) {
	rb := &domain.Runbook{
		Name:      "hello-world",
		Version:   "1.0.0",
		RiskLevel: domain.RiskLow,
	}
	cat := &domain.Catalog{Name: "default", Path: "~/.dops/catalogs/default"}
	out := Render(rb, cat, 60, true, testutil.TestStyles())

	// Path should still be visible (flashed green, not replaced).
	if !strings.Contains(out, "runbook.yaml") {
		t.Error("output should still show path when flash is true")
	}
}

func TestRender_Nil(t *testing.T) {
	out := Render(nil, nil, 40, false, testutil.TestStyles())
	if len(out) == 0 {
		t.Error("nil runbook should still produce output")
	}
}

func TestLocation(t *testing.T) {
	rb := &domain.Runbook{Name: "hello-world"}

	t.Run("local catalog", func(t *testing.T) {
		cat := &domain.Catalog{Path: "~/.dops/catalogs/default"}
		loc := Location(rb, cat)
		if loc != "~/.dops/catalogs/default/hello-world/runbook.yaml" {
			t.Errorf("got %q", loc)
		}
	})

	t.Run("git catalog", func(t *testing.T) {
		cat := &domain.Catalog{URL: "https://github.com/org/repo"}
		loc := Location(rb, cat)
		if loc != "https://github.com/org/repo" {
			t.Errorf("got %q", loc)
		}
	})

	t.Run("nil inputs", func(t *testing.T) {
		if Location(nil, nil) != "" {
			t.Error("nil inputs should return empty")
		}
	})
}
