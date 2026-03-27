package tui

import (
	"dops/internal/testutil"
	"dops/internal/tui/metadata"
	"testing"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// TestLayout_SidebarAlignsWithRightPanel verifies that the sidebar's rendered
// height matches metadata + output. This is a regression test — the P1
// refactor broke alignment by estimating sidebar height instead of measuring it.
func TestLayout_SidebarAlignsWithRightPanel(t *testing.T) {
	sizes := []struct {
		name   string
		width  int
		height int
	}{
		{"tiny", 64, 17},
		{"small", 80, 24},
		{"medium", 120, 40},
		{"large", 200, 60},
		{"wide", 220, 30},
		{"tall", 90, 55},
	}

	for _, sz := range sizes {
		t.Run(sz.name, func(t *testing.T) {
			m := NewApp(testCatalogs(), testutil.TestStyles())
			m.Init()

			result, _ := m.Update(tea.WindowSizeMsg{Width: sz.width, Height: sz.height})
			app := result.(App)
			l := app.computeLayout()

			// Render sidebar with actual content
			app.sidebar.SetHeight(l.sidebarContentH)
			sidebarView := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				PaddingLeft(1).
				Width(l.sw).
				Height(l.sidebarContentH).
				Render(app.sidebar.View())
			sidebarH := lipgloss.Height(sidebarView)

			// Render metadata
			metaContent := metadata.Render(app.selected, app.selCat, l.contentW, false, testutil.TestStyles())
			metaView := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Width(l.contentW).
				Render(metaContent)
			metaH := lipgloss.Height(metaView)

			// Derive output height the same way viewNormal does
			outputTotalH := clamp(sidebarH-metaH, 3)
			outputView := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Width(l.contentW).
				Height(outputTotalH).
				Render("")
			outputH := lipgloss.Height(outputView)

			rightH := metaH + outputH
			if sidebarH != rightH {
				t.Errorf("misaligned at %dx%d: sidebar=%d, meta(%d)+output(%d)=%d, diff=%d",
					sz.width, sz.height, sidebarH, metaH, outputH, rightH, sidebarH-rightH)
			}
		})
	}
}
