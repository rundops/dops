package footer

import (
	"dops/internal/testutil"
	"strings"
	"testing"
)

func TestRender_Normal(t *testing.T) {
	out := Render(StateNormal, 60, testutil.TestStyles())
	if !strings.Contains(out, "enter") {
		t.Error("normal state should show enter keybind")
	}
	if !strings.Contains(out, "q") {
		t.Error("normal state should show quit keybind")
	}
}

func TestRender_Running(t *testing.T) {
	out := Render(StateRunning, 60, testutil.TestStyles())
	if !strings.Contains(out, "running") && !strings.Contains(out, "Running") {
		t.Error("running state should indicate execution")
	}
}
