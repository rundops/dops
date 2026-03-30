package tui

import (
	"dops/internal/catalog"
	"dops/internal/domain"
	"dops/internal/testutil"
	"dops/internal/tui/output"
	"dops/internal/tui/palette"
	"dops/internal/tui/sidebar"
	"dops/internal/tui/wizard"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func testCatalogs() []catalog.CatalogWithRunbooks {
	return []catalog.CatalogWithRunbooks{
		{
			Catalog: domain.Catalog{Name: "default"},
			Runbooks: []domain.Runbook{
				{ID: "default.hello-world", Name: "hello-world", RiskLevel: domain.RiskLow},
				{ID: "default.rotate-tls", Name: "rotate-tls", RiskLevel: domain.RiskMedium},
			},
		},
	}
}

func TestApp_RunbookSelectedMsg(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	rb := domain.Runbook{ID: "default.rotate-tls", Name: "rotate-tls"}
	cat := domain.Catalog{Name: "default"}
	result, _ := m.Update(sidebar.RunbookSelectedMsg{Runbook: rb, Catalog: cat})
	app := result.(App)

	if app.Selected() == nil {
		t.Fatal("selected should be set after RunbookSelectedMsg")
	}
	if app.Selected().ID != "default.rotate-tls" {
		t.Errorf("selected = %q, want default.rotate-tls", app.Selected().ID)
	}
}

func TestApp_QuitOnQ(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Fatal("q should produce a quit command")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestApp_ViewNotEmpty(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	// Send WindowSizeMsg so layout has dimensions
	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := result.(App)
	view := app.View()
	if view.Content == "" {
		t.Error("View should produce non-empty content")
	}
}

func TestApp_WindowResize(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := result.(App)

	if app.Width() != 120 || app.Height() != 40 {
		t.Errorf("size = %dx%d, want 120x40", app.Width(), app.Height())
	}
}

func testCatalogsWithParams() []catalog.CatalogWithRunbooks {
	return []catalog.CatalogWithRunbooks{
		{
			Catalog: domain.Catalog{Name: "default"},
			Runbooks: []domain.Runbook{
				{
					ID:   "default.hello-world",
					Name: "hello-world",
					Parameters: []domain.Parameter{
						{Name: "greeting", Type: domain.ParamString, Required: true, Scope: "global"},
					},
				},
			},
		},
	}
}

func TestApp_ExecuteOpensWizard(t *testing.T) {
	m := NewApp(testCatalogsWithParams(), testutil.TestStyles())
	m.Init()

	// Sidebar sends RunbookExecuteMsg when Enter is pressed on a runbook
	rb := domain.Runbook{
		ID:   "default.hello-world",
		Name: "hello-world",
		Parameters: []domain.Parameter{
			{Name: "greeting", Type: domain.ParamString, Required: true, Scope: "global"},
		},
	}
	cat := domain.Catalog{Name: "default"}
	result, _ := m.Update(sidebar.RunbookExecuteMsg{Runbook: rb, Catalog: cat})
	app := result.(App)

	if app.ViewState() != stateWizard {
		t.Errorf("state = %d, want stateWizard (%d)", app.ViewState(), stateWizard)
	}
	if !app.HasWizard() {
		t.Error("wizard should be created")
	}
}

func TestApp_WizardCancel(t *testing.T) {
	m := NewApp(testCatalogsWithParams(), testutil.TestStyles())
	m.Init()

	// Set up and open wizard
	rb := domain.Runbook{
		ID:         "default.hello-world",
		Name:       "hello-world",
		Parameters: []domain.Parameter{{Name: "greeting", Type: domain.ParamString, Required: true, Scope: "global"}},
	}
	m.selected = &rb
	cat := domain.Catalog{Name: "default"}
	m.selCat = &cat
	m.state = stateWizard
	wiz := wizard.New(wizard.WizardConfig{
		Runbook:  rb,
		Catalog:  cat,
		Resolved: map[string]string{},
	})
	m.wizard = &wiz

	// Send cancel message
	result, _ := m.Update(wizard.CancelMsg{})
	app := result.(App)

	if app.ViewState() != stateNormal {
		t.Errorf("state after cancel = %d, want stateNormal", app.ViewState())
	}
	if app.HasWizard() {
		t.Error("wizard should be nil after cancel")
	}
}

func TestApp_WizardSubmit(t *testing.T) {
	m := NewApp(testCatalogsWithParams(), testutil.TestStyles())
	m.Init()

	m.state = stateWizard

	rb := domain.Runbook{ID: "default.hello-world", Name: "hello-world"}
	cat := domain.Catalog{Name: "default"}
	params := map[string]string{"greeting": "world"}

	result, _ := m.Update(wizard.SubmitMsg{Runbook: rb, Catalog: cat, Params: params})
	app := result.(App)

	if app.ViewState() != stateNormal {
		t.Errorf("state after submit = %d, want stateNormal", app.ViewState())
	}
	if app.HasWizard() {
		t.Error("wizard should be nil after submit")
	}
}

func TestApp_PaletteCancel(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	// Open palette
	p := palette.New(80)
	m.pal = &p
	m.state = statePalette

	result, _ := m.Update(palette.PaletteCancelMsg{})
	app := result.(App)

	if app.ViewState() != stateNormal {
		t.Errorf("state after palette cancel = %d, want stateNormal", app.ViewState())
	}
	if app.HasPalette() {
		t.Error("palette should be nil after cancel")
	}
}

func TestApp_PaletteSelect(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	p := palette.New(80)
	m.pal = &p
	m.state = statePalette

	cmd := palette.Command{Name: "theme: set"}
	result, _ := m.Update(palette.PaletteSelectMsg{Command: cmd})
	app := result.(App)

	if app.ViewState() != stateNormal {
		t.Errorf("state after palette select = %d, want stateNormal", app.ViewState())
	}
	if app.HasPalette() {
		t.Error("palette should be nil after select")
	}
}

func TestApp_MouseClickTranslation(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	// Send WindowSizeMsg so layout has dimensions
	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := result.(App)

	// Absolute coords for hello-world (visible index 1):
	// Y = layoutMarginTop(3) + borderTop(1) + itemIndex(1) = 5
	// X = layoutMarginLeft(3) + borderLeft(1) + padLeft(1) + some offset = 7
	result, cmd := app.Update(tea.MouseClickMsg{X: 7, Y: 5, Button: tea.MouseLeft})
	_ = result

	if cmd == nil {
		t.Fatal("click on runbook should produce a command")
	}

	msg := cmd()
	sel, ok := msg.(sidebar.RunbookSelectedMsg)
	if !ok {
		t.Fatalf("expected RunbookSelectedMsg, got %T", msg)
	}
	if sel.Runbook.ID != "default.hello-world" {
		t.Errorf("selected = %q, want default.hello-world", sel.Runbook.ID)
	}
}

func TestApp_ClickOutputFooterCopiesLogPath(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	rb := domain.Runbook{ID: "default.hello-world", Name: "hello-world", Version: "1.0.0"}
	cat := domain.Catalog{Name: "default", Path: "/tmp/default"}
	m.selected = &rb
	m.selCat = &cat
	m.output.SetCommand("dops run default.hello-world")
	m.output, _ = m.output.Update(output.ExecutionDoneMsg{LogPath: "/tmp/test.log"})

	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := result.(App)
	view := app.View().Content
	lines := strings.Split(view, "\n")

	clickX, clickY := -1, -1
	for y, line := range lines {
		clean := ansi.Strip(line)
		if idx := strings.Index(clean, "Saved to /tmp/test.log"); idx >= 0 {
			clickX = lipgloss.Width(clean[:idx]) + lipgloss.Width("Saved to ")/2
			clickY = y
			break
		}
	}

	if clickY == -1 {
		t.Fatal("footer text not found in rendered app view")
	}

	_, cmd := app.Update(tea.MouseClickMsg{X: clickX, Y: clickY, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatalf("click on rendered footer text at (%d,%d) should produce a copy command", clickX, clickY)
	}
}
