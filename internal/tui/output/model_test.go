package output

import (
	"dops/internal/theme"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func outputTestStyles() *theme.Styles {
	return theme.BuildStyles(&theme.ResolvedTheme{
		Name: "test",
		Colors: map[string]string{
			"background":        "#1a1b26",
			"backgroundPanel":   "#1f2335",
			"backgroundElement": "#292e42",
			"text":              "#c0caf5",
			"textMuted":         "#565f89",
			"primary":           "#7aa2f7",
			"border":            "#3b4261",
			"borderActive":      "#7aa2f7",
			"success":           "#9ece6a",
			"warning":           "#e0af68",
			"error":             "#f7768e",
			"risk.low":          "#9ece6a",
			"risk.medium":       "#e0af68",
			"risk.high":         "#f7768e",
			"risk.critical":     "#db4b4b",
		},
	})
}

func TestOutput_AppendLines(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m.SetCommand("dops run default.hello-world")

	m, _ = m.Update(OutputLineMsg{Text: "hello world", IsStderr: false})
	m, _ = m.Update(OutputLineMsg{Text: "error happened", IsStderr: true})

	if len(m.lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(m.lines))
	}
	if m.lines[0].Text != "hello world" {
		t.Errorf("line 0 = %q", m.lines[0].Text)
	}
	if !m.lines[1].IsStderr {
		t.Error("line 1 should be stderr")
	}
}

func TestOutput_ExecutionDone(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m, _ = m.Update(ExecutionDoneMsg{LogPath: "/tmp/test.log"})

	if m.logPath != "/tmp/test.log" {
		t.Errorf("logPath = %q", m.logPath)
	}
}

func TestOutput_ViewShowsCommand(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m.SetCommand("dops run default.hello-world --param greeting=world")

	view := m.View()
	if !strings.Contains(view, "dops run default.hello-world") {
		t.Error("view should show command")
	}
}

func TestOutput_ViewShowsLogPath(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m.SetCommand("dops run default.hello-world")
	m, _ = m.Update(ExecutionDoneMsg{LogPath: "/tmp/2026.01.01-010102-default-hello.log"})

	view := m.View()
	if !strings.Contains(view, "/tmp/2026.01.01-010102-default-hello.log") {
		t.Error("view should show log path")
	}
}

func TestOutput_ViewShowsStdout(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m.SetCommand("dops run default.hello-world")
	m, _ = m.Update(OutputLineMsg{Text: "hello from script", IsStderr: false})

	view := m.View()
	if !strings.Contains(view, "hello from script") {
		t.Error("view should show stdout")
	}
}

func TestOutput_Clear(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m.SetCommand("old command")
	m, _ = m.Update(OutputLineMsg{Text: "old line"})
	m, _ = m.Update(ExecutionDoneMsg{LogPath: "/tmp/old.log"})

	m.Clear()

	if m.command != "" || len(m.lines) != 0 || m.logPath != "" {
		t.Error("Clear should reset all fields")
	}
}

func TestOutput_HandleClick_Header(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m.SetCommand("dops run default.hello-world")

	lines := strings.Split(ansi.Strip(m.ViewWithSize(60, 20)), "\n")
	clickX, clickY := -1, -1
	for y, line := range lines {
		if idx := strings.Index(line, "$ dops run default.hello-world"); idx >= 0 {
			clickX = idx + 2
			clickY = y
			break
		}
	}
	if clickY == -1 {
		t.Fatal("header text not found in rendered output")
	}

	copyText, region := m.HandleClick(clickX, clickY, 60, 20)
	if region != "header" {
		t.Fatalf("region = %q, want header", region)
	}
	if copyText != "dops run default.hello-world" {
		t.Fatalf("copyText = %q", copyText)
	}
}

func TestOutput_HandleClick_Footer(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m.SetCommand("dops run default.hello-world")
	m, _ = m.Update(ExecutionDoneMsg{LogPath: "/tmp/test.log"})

	lines := strings.Split(ansi.Strip(m.ViewWithSize(60, 20)), "\n")
	clickX, clickY := -1, -1
	for y, line := range lines {
		if idx := strings.Index(line, "Saved to /tmp/test.log"); idx >= 0 {
			clickX = idx + len("Saved to ")/2
			clickY = y
			break
		}
	}
	if clickY == -1 {
		t.Fatal("footer text not found in rendered output")
	}

	copyText, region := m.HandleClick(clickX, clickY, 60, 20)
	if region != "footer" {
		t.Fatalf("region = %q, want footer", region)
	}
	if copyText != "/tmp/test.log" {
		t.Fatalf("copyText = %q", copyText)
	}
}

func TestOutput_ViewFooterRow(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m.SetCommand("dops run default.hello-world")
	m, _ = m.Update(ExecutionDoneMsg{LogPath: "/tmp/test.log"})

	view := m.ViewWithSize(60, 20)
	lines := strings.Split(view, "\n")
	totalH := len(lines)

	row := -1
	for i, line := range lines {
		if strings.Contains(line, "/tmp/test.log") {
			row = i
			break
		}
	}

	if row == -1 {
		t.Fatal("footer text not found in rendered output")
	}
	// Footer text is on the last row (no section borders).
	if row != totalH-1 {
		t.Fatalf("footer row = %d, want %d (totalH=%d)", row, totalH-1, totalH)
	}
	// Total rendered height must equal the requested height.
	if totalH != 20 {
		t.Fatalf("total rendered height = %d, want 20", totalH)
	}
}

func TestOutput_ViewEmptyBeforeSession(t *testing.T) {
	m := New(60, 20, outputTestStyles())

	view := m.View()
	// No session — output returns empty string. The app wraps in a border.
	if view != "" {
		t.Error("View() should return empty string before session")
	}
}

func TestOutput_ViewRendersAfterSetCommand(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m.SetCommand("dops run test")

	view := m.View()
	if view == "" {
		t.Error("View() should render after SetCommand")
	}
	plain := ansi.Strip(view)
	if !strings.Contains(plain, "$ dops run test") {
		t.Error("View() should show command after SetCommand")
	}
}

func TestOutput_CarriageReturnStripped(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m, _ = m.Update(OutputLineMsg{Text: "progress\r50%\r100%", IsStderr: false})

	if m.lines[0].Text != "progress50%100%" {
		t.Errorf("expected \\r stripped, got %q", m.lines[0].Text)
	}
}

func TestOutput_AtBottom_DefaultTrue(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	if !m.atBottom {
		t.Error("atBottom should be true by default")
	}
}

func TestOutput_AtBottom_DisabledOnScrollUp(t *testing.T) {
	m := New(60, 10, outputTestStyles())
	m.SetCommand("test")
	// Fill with enough lines to scroll
	for i := 0; i < 30; i++ {
		m, _ = m.Update(OutputLineMsg{Text: "line", IsStderr: false})
	}

	if !m.atBottom {
		t.Fatal("should be at bottom after appending")
	}

	// Scroll up
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.atBottom {
		t.Error("atBottom should be false after scrolling up")
	}
}

func TestOutput_HorizontalScroll_LKey(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m.SetCommand("test")
	m, _ = m.Update(OutputLineMsg{Text: strings.Repeat("x", 200), IsStderr: false})

	if m.xOffset != 0 {
		t.Fatal("xOffset should start at 0")
	}

	// Scroll right
	m, _ = m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	if m.xOffset != horizontalScrollStep {
		t.Errorf("xOffset = %d, want %d", m.xOffset, horizontalScrollStep)
	}

	// Scroll left
	m, _ = m.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	if m.xOffset != 0 {
		t.Errorf("xOffset = %d, want 0 after scroll left", m.xOffset)
	}
}

func TestOutput_HorizontalScroll_Clamped(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m.SetCommand("test")
	m, _ = m.Update(OutputLineMsg{Text: "short", IsStderr: false})

	// Try to scroll right on a short line — should stay at 0
	m, _ = m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	if m.xOffset != 0 {
		t.Errorf("xOffset should be clamped to 0 for short content, got %d", m.xOffset)
	}
}

func TestOutput_ScrollbarProportional(t *testing.T) {
	m := New(60, 10, outputTestStyles())
	m.SetCommand("test")
	for i := 0; i < 100; i++ {
		m, _ = m.Update(OutputLineMsg{Text: "line", IsStderr: false})
	}

	bodyH := m.bodyHeight()
	if bodyH <= 0 {
		t.Fatal("bodyH should be positive")
	}

	// At bottom, last row of scrollbar should be thumb
	lastChar := m.scrollbarCharAt(bodyH - 1)
	if lastChar != "█" {
		t.Errorf("at bottom, last scrollbar row should be thumb, got %q", lastChar)
	}

	// Scroll to top
	m.vp.SetYOffset(0)
	m.atBottom = false
	firstChar := m.scrollbarCharAt(0)
	if firstChar != "█" {
		t.Errorf("at top, first scrollbar row should be thumb, got %q", firstChar)
	}
}

func TestOutput_MaxLineWidthTracked(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m, _ = m.Update(OutputLineMsg{Text: "short", IsStderr: false})
	m, _ = m.Update(OutputLineMsg{Text: strings.Repeat("x", 200), IsStderr: false})

	if m.maxLineWidth != 200 {
		t.Errorf("maxLineWidth = %d, want 200", m.maxLineWidth)
	}
}

func TestOutput_ClearResetsNewFields(t *testing.T) {
	m := New(60, 20, outputTestStyles())
	m.SetCommand("test")
	m, _ = m.Update(OutputLineMsg{Text: strings.Repeat("x", 200), IsStderr: false})
	m.xOffset = 10
	m.atBottom = false

	m.Clear()

	if m.xOffset != 0 {
		t.Error("Clear should reset xOffset")
	}
	if !m.atBottom {
		t.Error("Clear should reset atBottom to true")
	}
	if m.maxLineWidth != 0 {
		t.Error("Clear should reset maxLineWidth")
	}
}
