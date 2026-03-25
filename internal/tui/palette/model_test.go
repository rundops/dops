package palette

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func pressKey(m Model, key string) (Model, tea.Cmd) {
	var msg tea.KeyPressMsg
	switch key {
	case "down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp}
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "escape":
		msg = tea.KeyPressMsg{Code: tea.KeyEscape}
	default:
		if len(key) == 1 {
			msg = tea.KeyPressMsg{Code: rune(key[0]), Text: key}
		}
	}
	return m.Update(msg)
}

func typeQuery(m Model, query string) Model {
	for _, ch := range query {
		msg := tea.KeyPressMsg{Code: ch, Text: string(ch)}
		m, _ = m.Update(msg)
	}
	return m
}

func TestPalette_InitialState(t *testing.T) {
	m := New(80)

	if len(m.Filtered()) == 0 {
		t.Error("should show all commands initially")
	}

	if len(m.Filtered()) != len(defaultCommands()) {
		t.Errorf("filtered = %d, want %d", len(m.Filtered()), len(defaultCommands()))
	}
}

func TestPalette_FilterByQuery(t *testing.T) {
	m := New(80)
	m = typeQuery(m, "theme")

	filtered := m.Filtered()
	for _, cmd := range filtered {
		if !strings.Contains(strings.ToLower(cmd.Name), "theme") {
			t.Errorf("filtered command %q should contain 'theme'", cmd.Name)
		}
	}

	if len(filtered) == 0 {
		t.Error("should have at least one match for 'theme'")
	}
}

func TestPalette_FilterNoMatches(t *testing.T) {
	m := New(80)
	m = typeQuery(m, "zzzznothing")

	if len(m.Filtered()) != 0 {
		t.Errorf("expected 0 matches, got %d", len(m.Filtered()))
	}
}

func TestPalette_NavigateDown(t *testing.T) {
	m := New(80)

	if m.Cursor() != 0 {
		t.Fatalf("initial cursor = %d", m.Cursor())
	}

	m, _ = pressKey(m, "down")
	if m.Cursor() != 1 {
		t.Errorf("after down: cursor = %d, want 1", m.Cursor())
	}
}

func TestPalette_NavigateUp(t *testing.T) {
	m := New(80)

	m, _ = pressKey(m, "down")
	m, _ = pressKey(m, "down")
	m, _ = pressKey(m, "up")

	if m.Cursor() != 1 {
		t.Errorf("after down,down,up: cursor = %d, want 1", m.Cursor())
	}
}

func TestPalette_NavigateClamps(t *testing.T) {
	m := New(80)

	// Up at top stays at 0
	m, _ = pressKey(m, "up")
	if m.Cursor() != 0 {
		t.Errorf("up at top: cursor = %d, want 0", m.Cursor())
	}

	// Down past end stays at last
	for i := 0; i < 20; i++ {
		m, _ = pressKey(m, "down")
	}
	max := len(m.Filtered()) - 1
	if m.Cursor() != max {
		t.Errorf("down past end: cursor = %d, want %d", m.Cursor(), max)
	}
}

func TestPalette_EnterEmitsSelect(t *testing.T) {
	m := New(80)

	_, cmd := pressKey(m, "enter")
	if cmd == nil {
		t.Fatal("enter should produce a command")
	}

	msg := cmd()
	sel, ok := msg.(PaletteSelectMsg)
	if !ok {
		t.Fatalf("expected PaletteSelectMsg, got %T", msg)
	}

	if sel.Command.Name == "" {
		t.Error("selected command should have a name")
	}
}

func TestPalette_EscapeEmitsCancel(t *testing.T) {
	m := New(80)

	_, cmd := pressKey(m, "escape")
	if cmd == nil {
		t.Fatal("escape should produce a command")
	}

	msg := cmd()
	if _, ok := msg.(PaletteCancelMsg); !ok {
		t.Fatalf("expected PaletteCancelMsg, got %T", msg)
	}
}

func TestPalette_FilterResetssCursor(t *testing.T) {
	m := New(80)

	// Move cursor down
	m, _ = pressKey(m, "down")
	m, _ = pressKey(m, "down")

	// Type filter — cursor should reset to 0
	m = typeQuery(m, "config")
	if m.Cursor() != 0 {
		t.Errorf("cursor after filter = %d, want 0", m.Cursor())
	}
}

func TestPalette_ViewNotEmpty(t *testing.T) {
	m := New(80)
	view := m.View()

	if len(view) == 0 {
		t.Error("view should not be empty")
	}
}

func TestPalette_ViewShowsQuery(t *testing.T) {
	m := New(80)
	m = typeQuery(m, "sec")

	view := m.View()
	if !strings.Contains(view, "sec") {
		t.Error("view should show the query")
	}
}

func TestPalette_SelectAfterFilter(t *testing.T) {
	m := New(80)
	m = typeQuery(m, "theme")

	_, cmd := pressKey(m, "enter")
	if cmd == nil {
		t.Fatal("enter should produce a command")
	}

	msg := cmd()
	sel, ok := msg.(PaletteSelectMsg)
	if !ok {
		t.Fatalf("expected PaletteSelectMsg, got %T", msg)
	}

	if !strings.Contains(strings.ToLower(sel.Command.Name), "theme") {
		t.Errorf("selected command = %q, should contain 'theme'", sel.Command.Name)
	}
}
