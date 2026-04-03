package help

import (
	"dops/internal/theme"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// FocusTarget mirrors the app's focus type without importing tui package.
type FocusTarget int

const (
	FocusSidebar FocusTarget = iota
	FocusOutput
)

type binding struct {
	key  string
	desc string
}

// Render returns the help overlay content based on which pane is focused.
func Render(focus FocusTarget, width int, styles *theme.Styles) string {
	keyStyle := lipgloss.NewStyle().Bold(true)
	descStyle := lipgloss.NewStyle()
	titleStyle := lipgloss.NewStyle().Bold(true)

	if styles != nil {
		keyStyle = styles.Primary.Bold(true)
		descStyle = styles.TextMuted
		titleStyle = styles.Text.Bold(true)
	}

	globalBindings := []binding{
		{"tab", "switch pane focus"},
		{"ctrl+shift+p", "command palette"},
		{"ctrl+x", "stop execution"},
		{"q / ctrl+c", "quit"},
	}

	sidebarBindings := []binding{
		{"↑ / ↓", "navigate runbooks"},
		{"← / →", "collapse / expand catalog"},
		{"enter", "run selected runbook"},
		{"/", "search runbooks"},
		{"esc", "clear search"},
	}

	outputBindings := []binding{
		{"↑ / ↓ / j / k", "scroll one line"},
		{"pgup / pgdn", "scroll one page"},
		{"h / l", "scroll left / right"},
		{"/", "search output"},
		{"n / N", "next / prev match"},
		{"esc", "clear search"},
	}

	w := width
	if w < 40 {
		w = 40
	}

	var sb strings.Builder

	renderSection := func(title string, bindings []binding) {
		sb.WriteString("\n")
		sb.WriteString("  " + titleStyle.Render(title) + "\n")
		sb.WriteString("\n")
		for _, b := range bindings {
			key := keyStyle.Render(b.key)
			pad := strings.Repeat(" ", max(1, 22-lipgloss.Width(b.key)))
			sb.WriteString("    " + key + pad + descStyle.Render(b.desc) + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString("  " + titleStyle.Render("Keyboard Shortcuts") + "\n")

	if focus == FocusSidebar {
		renderSection("Sidebar", sidebarBindings)
	} else {
		renderSection("Output", outputBindings)
	}

	renderSection("Global", globalBindings)
	sb.WriteString("\n")
	sb.WriteString("  " + descStyle.Render("Press ? or esc to close") + "\n")
	sb.WriteString("\n")

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(w)

	if styles != nil {
		border = border.BorderForeground(styles.Border.GetForeground())
	}

	return border.Render(sb.String())
}
