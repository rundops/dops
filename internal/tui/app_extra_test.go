package tui

import (
	"dops/internal/domain"
	"dops/internal/testutil"
	"dops/internal/tui/confirm"
	"dops/internal/tui/footer"
	"dops/internal/tui/output"
	"dops/internal/tui/palette"
	"dops/internal/tui/sidebar"
	"dops/internal/tui/wizard"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// ---------------------------------------------------------------------------
// Pure helper: isMouseMsg
// ---------------------------------------------------------------------------

func TestIsMouseMsg(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.Msg
		want bool
	}{
		{"MouseClickMsg", tea.MouseClickMsg{X: 1, Y: 2, Button: tea.MouseLeft}, true},
		{"MouseMotionMsg", tea.MouseMotionMsg{X: 1, Y: 2}, true},
		{"MouseReleaseMsg", tea.MouseReleaseMsg{X: 1, Y: 2}, true},
		{"MouseWheelMsg", tea.MouseWheelMsg{X: 1, Y: 2}, true},
		{"KeyPressMsg", tea.KeyPressMsg{Code: 'a'}, false},
		{"WindowSizeMsg", tea.WindowSizeMsg{Width: 80, Height: 24}, false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMouseMsg(tt.msg); got != tt.want {
				t.Errorf("isMouseMsg(%T) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Pure helper: isMouseClick
// ---------------------------------------------------------------------------

func TestIsMouseClick(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.Msg
		want bool
	}{
		{"MouseClickMsg", tea.MouseClickMsg{X: 1, Y: 2, Button: tea.MouseLeft}, true},
		{"MouseMotionMsg", tea.MouseMotionMsg{X: 1, Y: 2}, false},
		{"KeyPressMsg", tea.KeyPressMsg{Code: 'a'}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMouseClick(tt.msg); got != tt.want {
				t.Errorf("isMouseClick(%T) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Pure helper: mouseCoords
// ---------------------------------------------------------------------------

func TestMouseCoords(t *testing.T) {
	tests := []struct {
		name   string
		msg    tea.Msg
		wantX  int
		wantY  int
		wantOk bool
	}{
		{"MouseClickMsg", tea.MouseClickMsg{X: 10, Y: 20, Button: tea.MouseLeft}, 10, 20, true},
		{"MouseReleaseMsg", tea.MouseReleaseMsg{X: 5, Y: 15}, 5, 15, true},
		{"MouseMotionMsg", tea.MouseMotionMsg{X: 3, Y: 7}, 3, 7, true},
		{"MouseWheelMsg", tea.MouseWheelMsg{X: 8, Y: 12}, 8, 12, true},
		{"KeyPressMsg", tea.KeyPressMsg{Code: 'a'}, 0, 0, false},
		{"nil", nil, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y, ok := mouseCoords(tt.msg)
			if x != tt.wantX || y != tt.wantY || ok != tt.wantOk {
				t.Errorf("mouseCoords(%T) = (%d, %d, %v), want (%d, %d, %v)",
					tt.msg, x, y, ok, tt.wantX, tt.wantY, tt.wantOk)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Pure helper: clamp
// ---------------------------------------------------------------------------

func TestClamp(t *testing.T) {
	tests := []struct {
		v, min, want int
	}{
		{5, 3, 5},
		{3, 3, 3},
		{1, 3, 3},
		{0, 1, 1},
		{-5, 0, 0},
		{100, 50, 100},
	}
	for _, tt := range tests {
		if got := clamp(tt.v, tt.min); got != tt.want {
			t.Errorf("clamp(%d, %d) = %d, want %d", tt.v, tt.min, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Pure helper: sidebarWidth
// ---------------------------------------------------------------------------

func TestSidebarWidth(t *testing.T) {
	tests := []struct {
		total int
		want  int
	}{
		{60, sidebarMinWidth},        // 60/3=20 < min → clamped to min
		{90, sidebarMinWidth},        // 90/3=30 = min
		{120, 40},                    // 120/3=40
		{180, sidebarMaxWidth},       // 180/3=60 > max → clamped to max
		{10, sidebarMinWidth},        // very small → clamped to min
	}
	for _, tt := range tests {
		if got := sidebarWidth(tt.total); got != tt.want {
			t.Errorf("sidebarWidth(%d) = %d, want %d", tt.total, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Pure helper: buildEnv
// ---------------------------------------------------------------------------

func TestBuildEnv(t *testing.T) {
	params := map[string]string{
		"greeting": "hello",
		"target":   "world",
	}
	env := buildEnv(params)

	if env["GREETING"] != "hello" {
		t.Errorf("GREETING = %q, want hello", env["GREETING"])
	}
	if env["TARGET"] != "world" {
		t.Errorf("TARGET = %q, want world", env["TARGET"])
	}
	if len(env) != 2 {
		t.Errorf("len(env) = %d, want 2", len(env))
	}

	// Empty params.
	empty := buildEnv(map[string]string{})
	if len(empty) != 0 {
		t.Errorf("buildEnv(empty) returned %d entries", len(empty))
	}
}

// ---------------------------------------------------------------------------
// Pure helper: appFooterState
// ---------------------------------------------------------------------------

func TestAppFooterState(t *testing.T) {
	tests := []struct {
		state viewState
		want  footer.State
	}{
		{stateNormal, footer.StateNormal},
		{stateWizard, footer.StateWizard},
		{statePalette, footer.StatePalette},
		{stateConfirm, footer.StateConfirm},
		{stateHelp, footer.StateHelp},
		{viewState(99), footer.StateNormal}, // unknown → default
	}
	for _, tt := range tests {
		if got := appFooterState(tt.state); got != tt.want {
			t.Errorf("appFooterState(%d) = %d, want %d", tt.state, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Pure helper: injectBorderBadge
// ---------------------------------------------------------------------------

func TestInjectBorderBadge(t *testing.T) {
	// Build a box with a known top border line.
	box := "╭──────────────────────────────────────────╮\n│ content                                  │\n╰──────────────────────────────────────────╯"
	result := injectBorderBadge(box, "OK", nil)

	// The first line should contain "OK".
	lines := strings.Split(result, "\n")
	if !strings.Contains(lines[0], "OK") {
		t.Errorf("badge not found in top line: %q", lines[0])
	}

	// Content lines should be unmodified.
	if !strings.Contains(lines[1], "content") {
		t.Errorf("content line modified: %q", lines[1])
	}
}

func TestInjectBorderBadge_TooNarrow(t *testing.T) {
	// Very short box — badge shouldn't fit.
	box := "╭──╮\n│  │\n╰──╯"
	result := injectBorderBadge(box, "This badge is very long and should not fit", nil)

	// Should return original unchanged.
	if result != box {
		t.Error("expected unchanged box when badge too wide")
	}
}

func TestInjectBorderBadge_Empty(t *testing.T) {
	result := injectBorderBadge("", "badge", nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// Pure helper: highlightLine
// ---------------------------------------------------------------------------

func TestHighlightLine_OutsideRange(t *testing.T) {
	line := "hello world"
	bounds := selectionBounds{top: 0, bottom: 10, left: 0, right: 20}

	// Line index below selection start.
	result := highlightLine(line, 0, 5, 2, 10, 4, bounds, testHighlightStyle())
	if result != line {
		t.Errorf("expected unchanged line for i < startY, got %q", result)
	}

	// Line index above selection end.
	result = highlightLine(line, 5, 5, 2, 10, 4, bounds, testHighlightStyle())
	if result != line {
		t.Errorf("expected unchanged line for i > endY, got %q", result)
	}
}

func TestHighlightLine_EmptyLine(t *testing.T) {
	bounds := selectionBounds{top: 0, bottom: 10, left: 0, right: 20}
	result := highlightLine("", 3, 0, 2, 10, 4, bounds, testHighlightStyle())
	if result != "" {
		t.Errorf("expected empty line unchanged, got %q", result)
	}
}

func TestHighlightLine_WithinRange(t *testing.T) {
	line := "hello world"
	bounds := selectionBounds{top: 0, bottom: 10, left: 0, right: 20}

	// Middle line (not start or end).
	result := highlightLine(line, 3, 0, 2, 10, 4, bounds, testHighlightStyle())
	if result == line {
		t.Error("expected line to be modified within selection range")
	}
	// Should contain the original text (possibly styled).
	if !strings.Contains(result, "hello world") {
		t.Errorf("highlighted line should contain original text, got %q", result)
	}
}

func TestHighlightLine_SingleLineSelection(t *testing.T) {
	line := "hello world"
	bounds := selectionBounds{top: 0, bottom: 10, left: 0, right: 20}

	// startY == endY == i, selecting chars 2-7.
	result := highlightLine(line, 3, 2, 3, 7, 3, bounds, testHighlightStyle())
	if result == line {
		t.Error("expected line to be modified for single-line selection")
	}
}

// ---------------------------------------------------------------------------
// Pure helper: applySelectionHighlight
// ---------------------------------------------------------------------------

func TestApplySelectionHighlight_NoSelection(t *testing.T) {
	content := "line0\nline1\nline2"
	sel := output.TextSelection{Active: true, AnchorX: 5, AnchorY: 5, FocusX: 5, FocusY: 5}
	bounds := selectionBounds{top: 0, bottom: 2, left: 0, right: 10}

	// Selection is outside content bounds — startY > endY after clamping.
	sel.AnchorY = 10
	sel.FocusY = 12
	result := applySelectionHighlight(content, sel, nil, bounds)

	// Should still return something (possibly unchanged).
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestApplySelectionHighlight_WithinBounds(t *testing.T) {
	content := "aaaaaaaaaa\nbbbbbbbbbb\ncccccccccc"
	sel := output.TextSelection{Active: true, AnchorX: 2, AnchorY: 0, FocusX: 5, FocusY: 2}
	bounds := selectionBounds{top: 0, bottom: 2, left: 0, right: 10}

	result := applySelectionHighlight(content, sel, nil, bounds)
	if result == content {
		t.Error("expected content to be modified by selection highlight")
	}
}

// ---------------------------------------------------------------------------
// Model query methods
// ---------------------------------------------------------------------------

func TestSelectedCatalog(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	if m.SelectedCatalog() != nil {
		t.Error("SelectedCatalog should be nil initially")
	}

	cat := domain.Catalog{Name: "default"}
	rb := domain.Runbook{ID: "default.hello-world", Name: "hello-world"}
	result, _ := m.Update(sidebar.RunbookSelectedMsg{Runbook: rb, Catalog: cat})
	app := result.(App)

	if app.SelectedCatalog() == nil {
		t.Fatal("SelectedCatalog should be set after RunbookSelectedMsg")
	}
	if app.SelectedCatalog().Name != "default" {
		t.Errorf("SelectedCatalog().Name = %q, want default", app.SelectedCatalog().Name)
	}
}

func TestSelectedRunbook(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	if m.Selected() != nil {
		t.Error("Selected should be nil initially")
	}

	rb := domain.Runbook{ID: "default.hello-world", Name: "hello-world"}
	cat := domain.Catalog{Name: "default"}
	result, _ := m.Update(sidebar.RunbookSelectedMsg{Runbook: rb, Catalog: cat})
	app := result.(App)

	if app.Selected() == nil || app.Selected().ID != "default.hello-world" {
		t.Errorf("Selected().ID = %v, want default.hello-world", app.Selected())
	}
}

// ---------------------------------------------------------------------------
// handleKeyPress: Tab toggles focus
// ---------------------------------------------------------------------------

func TestHandleKeyPress_Tab(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateNormal
	m.focus = focusSidebar

	result, _, handled := m.handleKeyPress(tea.KeyPressMsg{Code: tea.KeyTab})
	if !handled {
		t.Fatal("tab should be handled")
	}
	app := result.(App)
	if app.focus != focusOutput {
		t.Errorf("focus after tab = %d, want focusOutput (%d)", app.focus, focusOutput)
	}

	// Tab again to go back.
	result, _, handled = app.handleKeyPress(tea.KeyPressMsg{Code: tea.KeyTab})
	if !handled {
		t.Fatal("tab should be handled")
	}
	app = result.(App)
	if app.focus != focusSidebar {
		t.Errorf("focus after second tab = %d, want focusSidebar (%d)", app.focus, focusSidebar)
	}
}

// ---------------------------------------------------------------------------
// handleKeyPress: ? toggles help
// ---------------------------------------------------------------------------

func TestHandleKeyPress_Help(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateNormal

	result, _, handled := m.handleKeyPress(tea.KeyPressMsg{Code: '?'})
	if !handled {
		t.Fatal("? should be handled")
	}
	app := result.(App)
	if app.state != stateHelp {
		t.Errorf("state after ? = %d, want stateHelp (%d)", app.state, stateHelp)
	}

	// ? again to close help.
	result, _, handled = app.handleKeyPress(tea.KeyPressMsg{Code: '?'})
	if !handled {
		t.Fatal("? in help state should be handled")
	}
	app = result.(App)
	if app.state != stateNormal {
		t.Errorf("state after second ? = %d, want stateNormal", app.state)
	}
}

// ---------------------------------------------------------------------------
// handleKeyPress: Escape closes help
// ---------------------------------------------------------------------------

func TestHandleKeyPress_EscapeClosesHelp(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateHelp

	result, _, handled := m.handleKeyPress(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !handled {
		t.Fatal("escape in help state should be handled")
	}
	app := result.(App)
	if app.state != stateNormal {
		t.Errorf("state after escape = %d, want stateNormal", app.state)
	}
}

// ---------------------------------------------------------------------------
// handleKeyPress: unhandled key in normal state
// ---------------------------------------------------------------------------

func TestHandleKeyPress_UnhandledKey(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateNormal

	_, _, handled := m.handleKeyPress(tea.KeyPressMsg{Code: 'z'})
	if handled {
		t.Error("'z' should not be handled in normal state")
	}
}

// ---------------------------------------------------------------------------
// handleKeyPress: ctrl+c quits
// ---------------------------------------------------------------------------

func TestHandleKeyPress_CtrlC(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateNormal

	_, cmd, handled := m.handleKeyPress(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if !handled {
		t.Fatal("ctrl+c should be handled")
	}
	if cmd == nil {
		t.Fatal("ctrl+c should produce a quit command")
	}
}

// ---------------------------------------------------------------------------
// handleKeyPress: ctrl+x cancels execution
// ---------------------------------------------------------------------------

func TestHandleKeyPress_CtrlX_CancelsExec(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateNormal

	cancelled := false
	m.execRunning = true
	m.cancelExec = func() { cancelled = true }

	_, _, handled := m.handleKeyPress(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	if !handled {
		t.Fatal("ctrl+x should be handled")
	}
	if !cancelled {
		t.Error("ctrl+x should call cancelExec")
	}
}

func TestHandleKeyPress_CtrlX_NoOp_WhenNotRunning(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateNormal
	m.execRunning = false

	_, _, handled := m.handleKeyPress(tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl})
	if !handled {
		t.Fatal("ctrl+x should be handled even when not running")
	}
}

// ---------------------------------------------------------------------------
// handleAppMessage: executionDoneMsg
// ---------------------------------------------------------------------------

func TestHandleAppMessage_ExecutionDone(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.execRunning = true
	m.cancelExec = func() {}

	result, _, handled := m.handleAppMessage(executionDoneMsg{LogPath: "/tmp/test.log"})
	if !handled {
		t.Fatal("executionDoneMsg should be handled")
	}
	app := result.(App)
	if app.execRunning {
		t.Error("execRunning should be false after executionDoneMsg")
	}
	if app.cancelExec != nil {
		t.Error("cancelExec should be nil after executionDoneMsg")
	}
}

// ---------------------------------------------------------------------------
// handleAppMessage: copiedFlashMsg
// ---------------------------------------------------------------------------

func TestHandleAppMessage_CopiedFlash(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.copiedFlash = true

	result, _, handled := m.handleAppMessage(copiedFlashMsg{})
	if !handled {
		t.Fatal("copiedFlashMsg should be handled")
	}
	app := result.(App)
	if app.copiedFlash {
		t.Error("copiedFlash should be false after copiedFlashMsg")
	}
}

// ---------------------------------------------------------------------------
// handleAppMessage: updateAvailableMsg
// ---------------------------------------------------------------------------

func TestHandleAppMessage_UpdateAvailable(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	result, _, handled := m.handleAppMessage(updateAvailableMsg{Version: "1.2.3"})
	if !handled {
		t.Fatal("updateAvailableMsg should be handled")
	}
	app := result.(App)
	if app.updateAvailable != "1.2.3" {
		t.Errorf("updateAvailable = %q, want 1.2.3", app.updateAvailable)
	}
}

// ---------------------------------------------------------------------------
// handleAppMessage: confirm.CancelMsg
// ---------------------------------------------------------------------------

func TestHandleAppMessage_ConfirmCancel(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateConfirm
	c := confirm.New(confirm.Params{
		Runbook: domain.Runbook{ID: "test", Name: "test"},
		Catalog: domain.Catalog{Name: "default"},
		Width:   60,
	})
	m.conf = &c

	result, _, handled := m.handleAppMessage(confirm.CancelMsg{})
	if !handled {
		t.Fatal("confirm.CancelMsg should be handled")
	}
	app := result.(App)
	if app.state != stateNormal {
		t.Errorf("state = %d, want stateNormal", app.state)
	}
	if app.conf != nil {
		t.Error("conf should be nil after cancel")
	}
}

// ---------------------------------------------------------------------------
// handleAppMessage: output.OutputLineMsg
// ---------------------------------------------------------------------------

func TestHandleAppMessage_OutputLine(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	_, _, handled := m.handleAppMessage(output.OutputLineMsg{Text: "hello"})
	if !handled {
		t.Fatal("OutputLineMsg should be handled")
	}
}

// ---------------------------------------------------------------------------
// handleAppMessage: unhandled message
// ---------------------------------------------------------------------------

func TestHandleAppMessage_Unhandled(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	_, _, handled := m.handleAppMessage(tea.WindowSizeMsg{Width: 80, Height: 24})
	if handled {
		t.Error("WindowSizeMsg should not be handled by handleAppMessage")
	}
}

// ---------------------------------------------------------------------------
// focusTargetFromMouse
// ---------------------------------------------------------------------------

func TestFocusTargetFromMouse(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := result.(App)

	// Non-mouse message should return false.
	_, ok := app.focusTargetFromMouse(tea.KeyPressMsg{Code: 'a'})
	if ok {
		t.Error("non-mouse msg should return ok=false")
	}

	// Click in sidebar area.
	target, ok := app.focusTargetFromMouse(tea.MouseClickMsg{X: 5, Y: 10, Button: tea.MouseLeft})
	if !ok {
		t.Fatal("click in sidebar should return ok=true")
	}
	if target != focusSidebar {
		t.Errorf("click in sidebar target = %d, want focusSidebar", target)
	}

	// Click in output area (far right).
	target, ok = app.focusTargetFromMouse(tea.MouseClickMsg{X: 100, Y: 10, Button: tea.MouseLeft})
	if !ok {
		t.Fatal("click in output area should return ok=true")
	}
	if target != focusOutput {
		t.Errorf("click in output area target = %d, want focusOutput", target)
	}
}

// ---------------------------------------------------------------------------
// View: zero dimensions return empty
// ---------------------------------------------------------------------------

func TestView_ZeroDimensions(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	// No WindowSizeMsg sent, so width/height are 0.
	v := m.View()
	if v.Content != "" {
		t.Error("View with zero dimensions should return empty content")
	}
}

// ---------------------------------------------------------------------------
// View: help overlay
// ---------------------------------------------------------------------------

func TestView_HelpOverlay(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := result.(App)

	app.state = stateHelp
	v := app.View()
	if v.Content == "" {
		t.Error("help overlay should produce non-empty content")
	}
}

// ---------------------------------------------------------------------------
// Model: NewAppWithDeps
// ---------------------------------------------------------------------------

func TestNewAppWithDeps(t *testing.T) {
	deps := AppDeps{
		Styles:   testutil.TestStyles(),
		Catalogs: testCatalogs(),
		DryRun:   true,
	}
	app := NewAppWithDeps(deps)
	app.Init()

	if app.Selected() != nil {
		t.Error("new app should have no selection")
	}
	if app.state != stateNormal {
		t.Error("new app should start in stateNormal")
	}
}

// ---------------------------------------------------------------------------
// Model: SetConfig
// ---------------------------------------------------------------------------

func TestSetConfig(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	cfg := &domain.Config{}
	m.SetConfig(cfg)
	if m.deps.Config != cfg {
		t.Error("SetConfig should set deps.Config")
	}
}

// ---------------------------------------------------------------------------
// routeToComponent: wizard/palette/confirm states
// ---------------------------------------------------------------------------

func TestRouteToComponent_WizardNil(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateWizard
	m.wizard = nil

	// Should not panic.
	result, _ := m.routeToComponent(tea.KeyPressMsg{Code: 'a'})
	if result == nil {
		t.Error("routeToComponent should return non-nil model")
	}
}

func TestRouteToComponent_PaletteNil(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = statePalette
	m.pal = nil

	result, _ := m.routeToComponent(tea.KeyPressMsg{Code: 'a'})
	if result == nil {
		t.Error("routeToComponent should return non-nil model")
	}
}

func TestRouteToComponent_ConfirmNil(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateConfirm
	m.conf = nil

	result, _ := m.routeToComponent(tea.KeyPressMsg{Code: 'a'})
	if result == nil {
		t.Error("routeToComponent should return non-nil model")
	}
}

// ---------------------------------------------------------------------------
// routeToComponent: normal state, focus output, non-mouse
// ---------------------------------------------------------------------------

func TestRouteToComponent_FocusOutput_KeyPress(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateNormal
	m.focus = focusOutput

	result, _ := m.routeToComponent(tea.KeyPressMsg{Code: 'j'})
	if result == nil {
		t.Error("routeToComponent should return non-nil model")
	}
}

// ---------------------------------------------------------------------------
// computeLayout: basic sanity
// ---------------------------------------------------------------------------

func TestComputeLayout(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	l := m.computeLayout()
	if l.innerW <= 0 {
		t.Errorf("innerW = %d, want > 0", l.innerW)
	}
	if l.sidebarW < sidebarMinWidth {
		t.Errorf("sidebarW = %d, want >= %d", l.sidebarW, sidebarMinWidth)
	}
	if l.rightW <= 0 {
		t.Errorf("rightW = %d, want > 0", l.rightW)
	}
	if l.panelRows <= 0 {
		t.Errorf("panelRows = %d, want > 0", l.panelRows)
	}
	if l.outputContentH <= 0 {
		t.Errorf("outputContentH = %d, want > 0", l.outputContentH)
	}
}

// ---------------------------------------------------------------------------
// sidebarBounds
// ---------------------------------------------------------------------------

func TestSidebarBounds(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	// Point inside sidebar area.
	_, _, in := m.sidebarBounds(5, 5)
	if !in {
		t.Error("(5,5) should be in sidebar bounds")
	}

	// Point far right — outside sidebar.
	_, _, in = m.sidebarBounds(100, 5)
	if in {
		t.Error("(100,5) should be outside sidebar bounds")
	}

	// Point above margin.
	_, _, in = m.sidebarBounds(5, 0)
	if in {
		t.Error("(5,0) should be outside sidebar bounds (above margin)")
	}
}

// ---------------------------------------------------------------------------
// translateMouseForOutput
// ---------------------------------------------------------------------------

func TestTranslateMouseForOutput(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	// Click msg
	click := tea.MouseClickMsg{X: 50, Y: 10, Button: tea.MouseLeft}
	result := m.translateMouseForOutput(click)
	translated, ok := result.(tea.MouseClickMsg)
	if !ok {
		t.Fatal("should return MouseClickMsg")
	}
	if translated.X >= 50 {
		t.Errorf("X should be translated (reduced), got %d", translated.X)
	}

	// Release msg
	release := tea.MouseReleaseMsg{X: 50, Y: 10}
	rResult := m.translateMouseForOutput(release)
	if _, ok := rResult.(tea.MouseReleaseMsg); !ok {
		t.Fatal("should return MouseReleaseMsg")
	}

	// Motion msg
	motion := tea.MouseMotionMsg{X: 50, Y: 10}
	mResult := m.translateMouseForOutput(motion)
	if _, ok := mResult.(tea.MouseMotionMsg); !ok {
		t.Fatal("should return MouseMotionMsg")
	}

	// Wheel msg
	wheel := tea.MouseWheelMsg{X: 50, Y: 10}
	wResult := m.translateMouseForOutput(wheel)
	if _, ok := wResult.(tea.MouseWheelMsg); !ok {
		t.Fatal("should return MouseWheelMsg")
	}

	// Non-mouse passthrough
	key := tea.KeyPressMsg{Code: tea.KeyEnter}
	if m.translateMouseForOutput(key) != key {
		t.Error("non-mouse msg should pass through unchanged")
	}
}

// ---------------------------------------------------------------------------
// outputPaneBounds
// ---------------------------------------------------------------------------

func TestOutputPaneBounds(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	top, bottom, left, right := m.outputPaneBounds()
	if top >= bottom {
		t.Errorf("top (%d) should be < bottom (%d)", top, bottom)
	}
	if left >= right {
		t.Errorf("left (%d) should be < right (%d)", left, right)
	}
	if top < 0 || left < 0 {
		t.Error("bounds should not be negative")
	}
}

// ---------------------------------------------------------------------------
// openPalette / openWizard
// ---------------------------------------------------------------------------

func TestOpenPalette(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	result, _ := m.openPalette()
	app := result.(App)
	if app.state != statePalette {
		t.Error("state should be statePalette after openPalette")
	}
	if app.pal == nil {
		t.Error("palette should not be nil")
	}
}

func TestOpenWizard_NoSelection(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40
	m.selected = nil

	result, cmd := m.openWizard()
	app := result.(App)
	if app.wizard != nil {
		t.Error("wizard should be nil when no runbook selected")
	}
	if cmd != nil {
		t.Error("should return nil cmd when no selection")
	}
}

func TestOpenWizard_WithSelection(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	rb := domain.Runbook{
		ID:   "test.hello",
		Name: "hello",
		Parameters: []domain.Parameter{
			{Name: "region", Type: domain.ParamString, Required: true, Scope: "global"},
		},
	}
	cat := testCatalogs()[0].Catalog
	m.selected = &rb
	m.selCat = &cat

	result, _ := m.openWizard()
	app := result.(App)
	if app.wizard == nil {
		t.Error("wizard should be created when runbook is selected")
	}
	if app.state != stateWizard {
		t.Error("state should be stateWizard")
	}
}

func TestOpenWizard_NoParams_OpensConfirm(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	rb := domain.Runbook{
		ID:       "test.noparam",
		Name:     "noparam",
		RiskLevel: domain.RiskHigh,
	}
	cat := testCatalogs()[0].Catalog
	m.selected = &rb
	m.selCat = &cat

	result, _ := m.openWizard()
	app := result.(App)
	// No params + high risk → should open confirm dialog
	if app.conf == nil {
		t.Error("should open confirm for high-risk runbook with no params")
	}
}

// ---------------------------------------------------------------------------
// viewConfirmOverlay / viewWizardOverlay / viewPaletteOverlay
// ---------------------------------------------------------------------------

func TestViewConfirmOverlay(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	rb := domain.Runbook{ID: "test.confirm", Name: "confirm", RiskLevel: domain.RiskHigh}
	cat := testCatalogs()[0].Catalog
	params := map[string]string{}
	c := confirm.New(confirm.Params{Runbook: rb, Catalog: cat, Resolved: params, Styles: m.deps.Styles})
	m.conf = &c
	m.state = stateConfirm

	view := m.viewConfirmOverlay()
	if view.Content == "" {
		t.Error("confirm overlay should produce non-empty content")
	}
}

func TestViewPaletteOverlay(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	result, _ := m.openPalette()
	app := result.(App)

	view := app.viewPaletteOverlay()
	if view.Content == "" {
		t.Error("palette overlay should produce non-empty content")
	}
}

func TestViewWizardOverlay(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	rb := domain.Runbook{
		ID:   "test.wiz",
		Name: "wiz",
		Parameters: []domain.Parameter{
			{Name: "x", Type: domain.ParamString, Required: true, Scope: "global"},
		},
	}
	cat := testCatalogs()[0].Catalog
	m.selected = &rb
	m.selCat = &cat

	result, _ := m.openWizard()
	app := result.(App)

	view := app.viewWizardOverlay()
	if view.Content == "" {
		t.Error("wizard overlay should produce non-empty content")
	}
}

// ---------------------------------------------------------------------------
// Output accessor
// ---------------------------------------------------------------------------

func TestOutput(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	o := m.Output()
	// Just ensure it returns without panic and is the zero model
	_ = o
}

// ---------------------------------------------------------------------------
// openConfirm with different risk levels
// ---------------------------------------------------------------------------

func TestOpenConfirm_LowRisk_SkipsConfirm(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	rb := domain.Runbook{
		ID:        "test.low",
		Name:      "low",
		RiskLevel: domain.RiskLow,
	}
	cat := testCatalogs()[0].Catalog
	params := map[string]string{}

	result, _ := m.openConfirm(rb, cat, params)
	app := result.(App)
	if app.conf != nil {
		t.Error("low risk should skip confirm dialog")
	}
}

func TestOpenConfirm_HighRisk_ShowsConfirm(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	rb := domain.Runbook{
		ID:        "test.high",
		Name:      "high",
		RiskLevel: domain.RiskHigh,
	}
	cat := testCatalogs()[0].Catalog
	params := map[string]string{}

	result, _ := m.openConfirm(rb, cat, params)
	app := result.(App)
	if app.conf == nil {
		t.Error("high risk should show confirm dialog")
	}
	if app.state != stateConfirm {
		t.Error("state should be stateConfirm")
	}
}

// ---------------------------------------------------------------------------
// Init with version check
// ---------------------------------------------------------------------------

func TestInit_WithVersion(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.deps.Version = "1.0.0"
	m.deps.DopsDir = "/tmp/test"
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return commands when version is set")
	}
}

func TestInit_WithoutVersion(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.deps.Version = ""
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should still return sidebar init cmd")
	}
}

// ---------------------------------------------------------------------------
// resolveVars
// ---------------------------------------------------------------------------

func TestResolveVars_NilSelection(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.selected = nil
	m.selCat = nil

	resolved := m.resolveVars()
	if len(resolved) != 0 {
		t.Error("should return empty map when nothing selected")
	}
}

// ---------------------------------------------------------------------------
// handleKeyPress: ctrl+shift+p opens palette
// ---------------------------------------------------------------------------

func TestHandleKeyPress_CtrlShiftP_OpensPalette(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	msg := tea.KeyPressMsg{Text: "ctrl+shift+p"}
	result, _, handled := m.handleKeyPress(msg)
	if !handled {
		t.Error("ctrl+shift+p should be handled")
	}
	app := result.(App)
	if app.state != statePalette {
		t.Error("ctrl+shift+p should open palette")
	}
}

// ---------------------------------------------------------------------------
// handleAppMessage: more message types
// ---------------------------------------------------------------------------

func TestHandleAppMessage_RunbookExecuteMsg(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40

	rb := domain.Runbook{ID: "test.exec", Name: "exec",
		Parameters: []domain.Parameter{{Name: "x", Type: domain.ParamString, Required: true, Scope: "global"}}}
	cat := testCatalogs()[0].Catalog

	msg := sidebar.RunbookExecuteMsg{Runbook: rb, Catalog: cat}
	result, _, handled := m.handleAppMessage(msg)
	if !handled {
		t.Error("RunbookExecuteMsg should be handled")
	}
	app := result.(App)
	if app.wizard == nil {
		t.Error("RunbookExecuteMsg should open wizard")
	}
}

func TestHandleAppMessage_WizardCancelMsg(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateWizard

	result, _, handled := m.handleAppMessage(wizard.CancelMsg{})
	if !handled {
		t.Error("should be handled")
	}
	app := result.(App)
	if app.state != stateNormal {
		t.Error("should return to normal state")
	}
}

func TestHandleAppMessage_PaletteSelectMsg(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = statePalette

	result, _, handled := m.handleAppMessage(palette.PaletteSelectMsg{})
	if !handled {
		t.Error("should be handled")
	}
	app := result.(App)
	if app.state != stateNormal {
		t.Error("should return to normal state")
	}
}

func TestHandleAppMessage_CopyFlashExpired(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()

	_, _, handled := m.handleAppMessage(output.CopyFlashExpiredMsg{})
	if !handled {
		t.Error("should be handled")
	}
}

func TestHandleAppMessage_CopiedRegionFlash(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.output.SetCopiedHeader(true)

	result, _, handled := m.handleAppMessage(output.CopiedRegionFlashMsg{})
	if !handled {
		t.Error("should be handled")
	}
	app := result.(App)
	_ = app
}

// ---------------------------------------------------------------------------
// routeToComponent: more states
// ---------------------------------------------------------------------------

func TestRouteToComponent_StateNormal_SidebarFocus(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40
	m.state = stateNormal
	m.focus = focusSidebar

	key := tea.KeyPressMsg{Text: "j"}
	_, cmd := m.routeToComponent(key)
	_ = cmd // just ensure no panic
}

func TestRouteToComponent_StateNormal_OutputFocus(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.width = 120
	m.height = 40
	m.state = stateNormal
	m.focus = focusOutput

	key := tea.KeyPressMsg{Text: "j"}
	_, cmd := m.routeToComponent(key)
	_ = cmd
}

// ---------------------------------------------------------------------------
// themeColors: nil vs non-nil styles
// ---------------------------------------------------------------------------

func TestThemeColors_NilStyles(t *testing.T) {
	m := NewApp(testCatalogs(), nil)
	border, active := m.themeColors()
	// With nil styles, should return NoColor.
	if border != (lipgloss.NoColor{}) {
		t.Errorf("border should be NoColor with nil styles, got %v", border)
	}
	if active != (lipgloss.NoColor{}) {
		t.Errorf("active border should be NoColor with nil styles, got %v", active)
	}
}

func TestThemeColors_WithStyles(t *testing.T) {
	styles := testutil.TestStyles()
	m := NewApp(testCatalogs(), styles)
	border, active := m.themeColors()
	// With real styles, at least one should be non-NoColor.
	_ = border
	_ = active
	// Just ensure no panic; real styles may still use NoColor.
}

// ---------------------------------------------------------------------------
// Layout: sidebarWidth and computeLayout produce consistent dimensions
// ---------------------------------------------------------------------------

func TestComputeLayout_RightPanelNarrowerThanTotal(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.width = 120
	m.height = 40
	l := m.computeLayout()

	// Right panel (output+metadata) must be narrower than total width.
	if l.rightW >= m.width {
		t.Errorf("rightW=%d should be less than total width=%d", l.rightW, m.width)
	}
	// contentW should be less than rightW (border subtracted).
	if l.contentW >= l.rightW {
		t.Errorf("contentW=%d should be less than rightW=%d", l.contentW, l.rightW)
	}
	// sidebarW + rightW + borders + gap should approximate width.
	if l.sidebarW <= 0 || l.rightW <= 0 {
		t.Errorf("sidebarW=%d and rightW=%d should both be positive", l.sidebarW, l.rightW)
	}
}

// ---------------------------------------------------------------------------
// focusTargetFromMouse returns correct focus pane
// ---------------------------------------------------------------------------

func TestFocusTargetFromMouse_SidebarRegion(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.width = 120
	m.height = 40

	// Click in sidebar area (far left).
	click := tea.MouseClickMsg{X: layoutMarginLeft + 2, Y: layoutMarginTop + 2, Button: tea.MouseLeft}
	target, ok := m.focusTargetFromMouse(click)
	if !ok {
		t.Fatal("should detect mouse in sidebar")
	}
	if target != focusSidebar {
		t.Errorf("target = %d, want focusSidebar (%d)", target, focusSidebar)
	}
}

func TestFocusTargetFromMouse_OutputRegion(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.width = 120
	m.height = 40

	// Click far right — should be output area.
	click := tea.MouseClickMsg{X: 100, Y: layoutMarginTop + 2, Button: tea.MouseLeft}
	target, ok := m.focusTargetFromMouse(click)
	if !ok {
		t.Fatal("should detect mouse in output")
	}
	if target != focusOutput {
		t.Errorf("target = %d, want focusOutput (%d)", target, focusOutput)
	}
}

// ---------------------------------------------------------------------------
// handleAppMessage: SaveFieldMsg
// ---------------------------------------------------------------------------

func TestHandleAppMessage_SaveFieldMsg(t *testing.T) {
	m := NewApp(testCatalogs(), testutil.TestStyles())
	m.Init()
	m.state = stateWizard

	msg := wizard.SaveFieldMsg{
		Scope:       "global",
		ParamName:   "region",
		CatalogName: "default",
		RunbookName: "hello-world",
		Value:       "us-east-1",
	}

	result, cmd, handled := m.handleAppMessage(msg)
	if !handled {
		t.Error("SaveFieldMsg should be handled")
	}
	_ = result
	if cmd == nil {
		t.Fatal("should return a command with SaveFieldResultMsg")
	}
	// Execute the command — should return SaveFieldResultMsg.
	// Without real config/vault, it will error, but the message type is correct.
	resultMsg := cmd()
	if saveResult, ok := resultMsg.(wizard.SaveFieldResultMsg); !ok {
		t.Errorf("expected SaveFieldResultMsg, got %T", resultMsg)
	} else if m.deps.Config == nil && saveResult.Err == nil {
		// With nil config, should error.
		t.Error("expected error with nil config")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testHighlightStyle() lipgloss.Style {
	return lipgloss.NewStyle()
}
