package sidebar

import (
	"dops/internal/catalog"
	"dops/internal/domain"
	"dops/internal/testutil"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
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
		{
			Catalog: domain.Catalog{Name: "local"},
			Runbooks: []domain.Runbook{
				{ID: "local.drain-node", Name: "drain-node", RiskLevel: domain.RiskHigh},
			},
		},
	}
}

func pressKey(m Model, key string) (Model, tea.Cmd) {
	var msg tea.KeyPressMsg
	switch key {
	case "down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp}
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "left":
		msg = tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		msg = tea.KeyPressMsg{Code: tea.KeyRight}
	case "/":
		msg = tea.KeyPressMsg{Code: '/', Text: "/"}
	default:
		if len(key) == 1 {
			msg = tea.KeyPressMsg{Code: rune(key[0]), Text: key}
		}
	}
	return m.Update(msg)
}

func TestSidebar_InitialSelection(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	cmd := m.Init()

	// Cursor starts at 0 (first item = default/ header)
	// But Init should select first runbook
	if cmd == nil {
		t.Fatal("Init should return a command")
	}

	msg := cmd()
	sel, ok := msg.(RunbookSelectedMsg)
	if !ok {
		t.Fatalf("expected RunbookSelectedMsg, got %T", msg)
	}
	if sel.Runbook.ID != "default.hello-world" {
		t.Errorf("initial selection = %q, want default.hello-world", sel.Runbook.ID)
	}
}

func TestSidebar_NavigateDown(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	// Visible: default/ (0), hello-world (1), rotate-tls (2), local/ (3), drain-node (4)
	// Cursor starts on hello-world (1)
	if sel := m.Selected(); sel == nil || sel.ID != "default.hello-world" {
		t.Errorf("initial: want hello-world, got %v", sel)
	}

	m, _ = pressKey(m, "down") // → rotate-tls
	if sel := m.Selected(); sel == nil || sel.ID != "default.rotate-tls" {
		t.Errorf("after 1 down: want rotate-tls, got %v", sel)
	}

	m, _ = pressKey(m, "down") // → local/ header
	if sel := m.Selected(); sel != nil {
		t.Error("on catalog header, Selected should be nil")
	}

	m, _ = pressKey(m, "down") // → drain-node
	if sel := m.Selected(); sel == nil || sel.ID != "local.drain-node" {
		t.Errorf("after 3 down: want drain-node, got %v", sel)
	}
}

func TestSidebar_NavigateUp(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	// Go to bottom
	for i := 0; i < 4; i++ {
		m, _ = pressKey(m, "down")
	}

	m, _ = pressKey(m, "up") // → local/ header
	m, _ = pressKey(m, "up") // → rotate-tls
	if sel := m.Selected(); sel == nil || sel.ID != "default.rotate-tls" {
		t.Errorf("want rotate-tls, got %v", sel)
	}

	m, _ = pressKey(m, "up") // → hello-world
	if sel := m.Selected(); sel == nil || sel.ID != "default.hello-world" {
		t.Errorf("want hello-world, got %v", sel)
	}
}

func TestSidebar_CollapseExpand(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	// Move cursor up to default/ header, then collapse
	m, _ = pressKey(m, "up")
	m, _ = pressKey(m, "enter")

	// default/ should be collapsed — its runbooks hidden
	vis := m.Visible()
	for _, idx := range vis {
		if !m.EntryIsHeader(idx) {
			// Check if it's a default catalog runbook by looking at visible entries
			// Since we can't access entry fields directly, verify via collapse state
		}
	}

	// Verify collapsed state
	if !m.IsCollapsed("default") {
		t.Error("default should be collapsed after enter on header")
	}

	// Press Enter again to expand
	m, _ = pressKey(m, "enter")

	if m.IsCollapsed("default") {
		t.Error("default should be expanded after second enter")
	}
}

func TestSidebar_EnterOnRunbook_EmitsExecute(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	// Cursor starts on hello-world
	_, cmd := pressKey(m, "enter")

	if cmd == nil {
		t.Fatal("enter on runbook should emit a command")
	}

	msg := cmd()
	exec, ok := msg.(RunbookExecuteMsg)
	if !ok {
		t.Fatalf("expected RunbookExecuteMsg, got %T", msg)
	}
	if exec.Runbook.ID != "default.hello-world" {
		t.Errorf("execute runbook = %q", exec.Runbook.ID)
	}
}

func TestSidebar_LeftCollapses(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	// Cursor starts on hello-world — left jumps to header, left again collapses
	m, _ = pressKey(m, "left") // → default/ header
	m, _ = pressKey(m, "left") // collapse

	if !m.IsCollapsed("default") {
		t.Error("left on header should collapse catalog")
	}

	// Runbooks should be hidden
	vis := m.VisibleRunbooks()
	for _, rb := range vis {
		if rb.ID == "default.hello-world" || rb.ID == "default.rotate-tls" {
			t.Error("default runbooks should be hidden")
		}
	}
}

func TestSidebar_RightExpands(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	// Move to header, collapse
	m, _ = pressKey(m, "left") // → header
	m, _ = pressKey(m, "left") // collapse
	if !m.IsCollapsed("default") {
		t.Fatal("should be collapsed")
	}

	// Right arrow expands
	m, _ = pressKey(m, "right")
	if m.IsCollapsed("default") {
		t.Error("right on collapsed header should expand")
	}
}

func TestSidebar_LeftOnRunbook_JumpsToParent(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	// Cursor starts on hello-world
	if sel := m.Selected(); sel == nil || sel.ID != "default.hello-world" {
		t.Fatal("should start on hello-world")
	}

	// Left arrow jumps to parent header
	m, _ = pressKey(m, "left")
	if m.Selected() != nil {
		t.Error("should be on header (nil selection)")
	}
	if m.Cursor() != 0 {
		t.Errorf("cursor = %d, want 0 (default/ header)", m.Cursor())
	}
}

func TestSidebar_MouseClickRunbook(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	// Visible: default/ (0), hello-world (1), rotate-tls (2), local/ (3), drain-node (4)
	// Content-relative: Y=0 = item 0, Y=2 = item 2 (rotate-tls)
	m, cmd := m.Update(tea.MouseClickMsg{X: 5, Y: 2, Button: tea.MouseLeft})

	if m.Cursor() != 2 {
		t.Errorf("cursor = %d, want 2", m.Cursor())
	}
	sel := m.Selected()
	if sel == nil || sel.ID != "default.rotate-tls" {
		t.Errorf("selected = %v, want rotate-tls", sel)
	}
	if cmd == nil {
		t.Error("click on runbook should emit selection command")
	}
}

func TestSidebar_MouseClickHeader(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	// Content-relative: Y=0 = item 0 (default/ header)
	m, _ = m.Update(tea.MouseClickMsg{X: 5, Y: 0, Button: tea.MouseLeft})

	if !m.IsCollapsed("default") {
		t.Error("click on header should collapse catalog")
	}

	// Click again should expand
	m, _ = m.Update(tea.MouseClickMsg{X: 5, Y: 0, Button: tea.MouseLeft})

	if m.IsCollapsed("default") {
		t.Error("second click should expand catalog")
	}
}

func TestSidebar_DoubleClickExecutes(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	// Single click on Y=1 (hello-world) — selects
	m, cmd := m.Update(tea.MouseClickMsg{X: 5, Y: 1, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("single click should emit selection")
	}
	if _, ok := cmd().(RunbookExecuteMsg); ok {
		t.Error("single click should NOT execute")
	}

	// Second click on same Y immediately — double-click executes
	m, cmd = m.Update(tea.MouseClickMsg{X: 5, Y: 1, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("double click should emit a command")
	}
	msg := cmd()
	exec, ok := msg.(RunbookExecuteMsg)
	if !ok {
		t.Fatalf("double click should emit RunbookExecuteMsg, got %T", msg)
	}
	if exec.Runbook.ID != "default.hello-world" {
		t.Errorf("executed = %q, want default.hello-world", exec.Runbook.ID)
	}
}

func TestSidebar_MouseHover(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	// Hover over Y=1 (item 1 = hello-world)
	m, _ = m.Update(tea.MouseMotionMsg{X: 5, Y: 1})

	if m.HoverIdx() != 1 {
		t.Errorf("hoverIdx = %d, want 1", m.HoverIdx())
	}

	// Hover outside bounds
	m, _ = m.Update(tea.MouseMotionMsg{X: 5, Y: 100})
	if m.HoverIdx() != -1 {
		t.Errorf("hoverIdx = %d, want -1 (out of bounds)", m.HoverIdx())
	}
}

func TestSidebar_KeyboardClearsHover(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	// Set hover
	m, _ = m.Update(tea.MouseMotionMsg{X: 5, Y: 1})
	if m.HoverIdx() != 1 {
		t.Fatal("hover should be set")
	}

	// Keyboard input clears hover
	m, _ = pressKey(m, "down")
	if m.HoverIdx() != -1 {
		t.Errorf("hoverIdx = %d, want -1 after keyboard", m.HoverIdx())
	}
}

func TestSidebar_EmptyCatalogs(t *testing.T) {
	m := New(nil, 20, testutil.TestStyles())
	m.Init()

	if m.Selected() != nil {
		t.Error("expected nil selection with no catalogs")
	}

	m, _ = pressKey(m, "down")
	m, _ = pressKey(m, "up")
}

func TestSidebar_ViewNotEmpty(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	view := m.View()
	if len(view) == 0 {
		t.Error("View returned empty string")
	}
}

func TestSidebar_ViewShowsCollapseIndicator(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())
	m.Init()

	view := m.View()
	if !strings.Contains(view, "▼") {
		t.Error("expanded catalog should show ▼")
	}

	m, _ = pressKey(m, "up")    // → default/ header
	m, _ = pressKey(m, "enter") // collapse default/

	view = m.View()
	if !strings.Contains(view, "▶") {
		t.Error("collapsed catalog should show ▶")
	}
}

func TestRenderEntry_HoverVsUnselected(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())

	e := entry{
		runbook: domain.Runbook{Name: "test-runbook"},
		catalog: domain.Catalog{Name: "default"},
	}
	s := entryStyles{
		cursor:         lipgloss.NewStyle().Bold(true),
		hover:          lipgloss.NewStyle().Underline(true),
		header:         lipgloss.NewStyle(),
		headerSelected: lipgloss.NewStyle(),
		selected:       lipgloss.NewStyle().Bold(true),
		unselected:     lipgloss.NewStyle(),
	}

	hovered := m.renderEntry(e, s, false, true, false, false)
	normal := m.renderEntry(e, s, false, false, false, false)

	if hovered == normal {
		t.Error("hovered entry should be styled differently from unselected entry")
	}
}

func TestRenderEntry_SelectedTakesPriority(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())

	e := entry{
		runbook: domain.Runbook{Name: "test-runbook"},
		catalog: domain.Catalog{Name: "default"},
	}
	s := entryStyles{
		cursor:         lipgloss.NewStyle().Bold(true),
		hover:          lipgloss.NewStyle().Underline(true),
		header:         lipgloss.NewStyle(),
		headerSelected: lipgloss.NewStyle(),
		selected:       lipgloss.NewStyle().Bold(true),
		unselected:     lipgloss.NewStyle(),
	}

	// isCursor=true, isHovered=true — selected should win.
	selectedHovered := m.renderEntry(e, s, true, true, false, false)
	selectedOnly := m.renderEntry(e, s, true, false, false, false)

	if selectedHovered != selectedOnly {
		t.Errorf("selected should take priority over hover\nselected+hover: %q\nselected only:  %q", selectedHovered, selectedOnly)
	}
}

func TestBuildLines_HoverAppliesCorrectly(t *testing.T) {
	m := New(testCatalogs(), 20, testutil.TestStyles())

	// Cursor on hello-world (vis index 1), hover on rotate-tls (vis index 2).
	m.hoverIdx = 2

	vis := m.visible()
	lines := m.buildLines(vis)
	if len(lines) < 5 {
		t.Fatalf("expected at least 5 lines, got %d", len(lines))
	}

	// Build another version without hover.
	m2 := New(testCatalogs(), 20, testutil.TestStyles())
	m2.hoverIdx = -1

	vis2 := m2.visible()
	lines2 := m2.buildLines(vis2)

	// The hovered entry (rotate-tls, lines[2]) should be different from the
	// non-hovered version (lines2[2]).
	if lines[2] == lines2[2] {
		t.Error("hovered rotate-tls should be styled differently from non-hovered rotate-tls")
	}

	// The non-hovered entries should be identical.
	if lines[4] != lines2[4] {
		t.Error("non-hovered drain-node should be identical in both versions")
	}
}
