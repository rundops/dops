package footer

import (
	"dops/internal/testutil"
	"strings"
	"testing"
)

func TestRender_Normal(t *testing.T) {
	out := Render(StateNormal, 60, testutil.TestStyles(), "")
	if !strings.Contains(out, "enter") {
		t.Error("normal state should show enter keybind")
	}
	if !strings.Contains(out, "q") {
		t.Error("normal state should show quit keybind")
	}
}

func TestRender_Running(t *testing.T) {
	out := Render(StateRunning, 60, testutil.TestStyles(), "")
	if !strings.Contains(out, "running") && !strings.Contains(out, "Running") {
		t.Error("running state should indicate execution")
	}
}

func TestRender_UpdateAvailable(t *testing.T) {
	out := Render(StateNormal, 120, testutil.TestStyles(), "0.2.0")
	if !strings.Contains(out, "brew upgrade dops") {
		t.Error("should show brew upgrade notice when update is available")
	}
	if !strings.Contains(out, "enter") {
		t.Error("should still show keybindings alongside update notice")
	}
}

func TestRender_UpdateAvailable_NarrowTerminal(t *testing.T) {
	out := Render(StateNormal, 40, testutil.TestStyles(), "0.2.0")
	if strings.Contains(out, "brew upgrade") {
		t.Error("should hide update notice when terminal is too narrow")
	}
}
