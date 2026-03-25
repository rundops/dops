package footer

import (
	"dops/internal/theme"

	lipgloss "charm.land/lipgloss/v2"
)

type State int

const (
	StateNormal  State = iota
	StateWizard
	StateRunning
	StatePalette
	StateConfirm
	StateHelp
)

type binding struct {
	key  string
	desc string
}

// Render returns the footer bar with keybindings on the left.
// If updateVersion is non-empty, a right-aligned update notice is appended.
func Render(state State, width int, styles *theme.Styles, updateVersion string) string {
	var bindings []binding

	switch state {
	case StateNormal:
		bindings = []binding{
			{"↑↓", "navigate"},
			{"enter", "run"},
			{"/", "search"},
			{"ctrl+shift+p", "palette"},
			{"q", "quit"},
		}
	case StateWizard:
		bindings = []binding{
			{"tab", "next"},
			{"shift+tab", "prev"},
			{"enter", "submit"},
			{"esc", "cancel"},
		}
	case StateRunning:
		bindings = []binding{
			{"", "Running..."},
			{"esc", "cancel"},
		}
	case StatePalette:
		bindings = []binding{
			{"↑↓", "select"},
			{"enter", "confirm"},
			{"esc", "close"},
		}
	case StateConfirm:
		bindings = []binding{
			{"enter", "confirm"},
			{"esc", "cancel"},
		}
	case StateHelp:
		bindings = []binding{
			{"?", "close"},
			{"esc", "close"},
		}
	}

	keyStyle := lipgloss.NewStyle()
	descStyle := lipgloss.NewStyle()

	if styles != nil {
		keyStyle = styles.Primary
		descStyle = styles.TextMuted
	}

	var left string
	for i, b := range bindings {
		if i > 0 {
			left += descStyle.Render(" • ")
		}
		if b.key != "" {
			left += keyStyle.Render(b.key) + " " + descStyle.Render(b.desc)
		} else {
			left += descStyle.Render(b.desc)
		}
	}
	left = "  " + left

	if updateVersion == "" {
		return lipgloss.NewStyle().Width(width).Render(left)
	}

	// Build right-aligned update notice.
	warnStyle := lipgloss.NewStyle()
	boldStyle := lipgloss.NewStyle().Bold(true)
	if styles != nil {
		warnStyle = styles.Warning
		boldStyle = styles.Warning.Bold(true)
	}
	right := warnStyle.Render("Update available! Run: ") + boldStyle.Render("brew upgrade dops")

	rightW := lipgloss.Width(right)
	leftW := lipgloss.Width(left)
	gap := width - leftW - rightW
	if gap < 1 {
		// Not enough room — just show keybindings.
		return lipgloss.NewStyle().Width(width).Render(left)
	}

	row := left + lipgloss.NewStyle().Width(gap).Render("") + right
	return lipgloss.NewStyle().Width(width).Render(row)
}
