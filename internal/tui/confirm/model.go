package confirm

import (
	"fmt"
	"strings"

	"dops/internal/domain"
	"dops/internal/theme"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type Model struct {
	runbook domain.Runbook
	catalog domain.Catalog
	params  map[string]string
	risk    domain.RiskLevel
	input   string
	cursor  int // 0 = Yes, 1 = No (for high risk toggle)
	width   int
	styles  *theme.Styles
}

func New(rb domain.Runbook, cat domain.Catalog, params map[string]string, width int, styles *theme.Styles) Model {
	return Model{
		runbook: rb,
		catalog: cat,
		params:  params,
		risk:    rb.RiskLevel,
		cursor:  1, // default to No
		width:   width,
		styles:  styles,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case msg.Code == tea.KeyEscape:
			return m, func() tea.Msg { return ConfirmCancelMsg{} }

		case msg.Code == tea.KeyEnter:
			if m.risk == domain.RiskHigh {
				if m.cursor == 0 {
					return m, m.accept()
				}
				return m, func() tea.Msg { return ConfirmCancelMsg{} }
			}
			if m.isConfirmed() {
				return m, m.accept()
			}

		case msg.Code == tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}

		default:
			if m.risk == domain.RiskHigh {
				// y/N toggle or direct key.
				switch {
				case msg.Code == tea.KeyLeft || msg.Text == "h" || msg.Code == tea.KeyTab:
					m.cursor = 0
				case msg.Code == tea.KeyRight || msg.Text == "l":
					m.cursor = 1
				case msg.Text == "y" || msg.Text == "Y":
					return m, m.accept()
				case msg.Text == "n" || msg.Text == "N":
					return m, func() tea.Msg { return ConfirmCancelMsg{} }
				}
			} else if msg.Text != "" {
				// Critical: accumulate typed input.
				m.input += msg.Text
			}
		}
	}
	return m, nil
}

func (m Model) accept() func() tea.Msg {
	return func() tea.Msg {
		return ConfirmAcceptMsg{
			Runbook: m.runbook,
			Catalog: m.catalog,
			Params:  m.params,
		}
	}
}

func (m Model) isConfirmed() bool {
	switch m.risk {
	case domain.RiskHigh:
		return false
	case domain.RiskCritical:
		return strings.TrimSpace(m.input) == m.runbook.ID
	default:
		return true
	}
}

func (m Model) View() string {
	primaryStyle := lipgloss.NewStyle().Bold(true)
	mutedStyle := lipgloss.NewStyle()
	textStyle := lipgloss.NewStyle()
	warningStyle := lipgloss.NewStyle()
	errorStyle := lipgloss.NewStyle()
	successStyle := lipgloss.NewStyle()

	if m.styles != nil {
		primaryStyle = m.styles.Primary.Bold(true)
		mutedStyle = m.styles.TextMuted
		textStyle = m.styles.Text
		warningStyle = m.styles.Warning
		errorStyle = m.styles.Error.Bold(true)
		successStyle = m.styles.Success
	}

	var sb strings.Builder

	// Header: $ dops run <id>
	sb.WriteString(successStyle.Render("$") + " " + textStyle.Bold(true).Render(fmt.Sprintf("dops run %s", m.runbook.ID)))
	sb.WriteString("\n\n")

	// Risk warning
	riskLabel := strings.ToUpper(string(m.risk))
	switch m.risk {
	case domain.RiskHigh:
		sb.WriteString(warningStyle.Render(fmt.Sprintf("⚠  %s RISK", riskLabel)))
	case domain.RiskCritical:
		sb.WriteString(errorStyle.Render(fmt.Sprintf("⚠  %s RISK", riskLabel)))
	}
	sb.WriteString("\n\n")

	// Runbook ID in muted
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("Runbook: %s", m.runbook.ID)))
	sb.WriteString("\n\n")

	// Prompt
	switch m.risk {
	case domain.RiskHigh:
		sb.WriteString(primaryStyle.Render("Confirm execution?"))
		sb.WriteString("\n\n")

		// [Yes] [No] toggle — same style as wizard boolean/save prompt.
		yesStyle := lipgloss.NewStyle()
		noStyle := lipgloss.NewStyle()
		if m.styles != nil {
			if m.cursor == 0 {
				yesStyle = lipgloss.NewStyle().
					Background(m.styles.Primary.GetForeground()).
					Foreground(m.styles.Background.GetForeground()).
					Padding(0, 1)
				noStyle = m.styles.TextMuted.Padding(0, 1)
			} else {
				yesStyle = m.styles.TextMuted.Padding(0, 1)
				noStyle = lipgloss.NewStyle().
					Background(m.styles.Primary.GetForeground()).
					Foreground(m.styles.Background.GetForeground()).
					Padding(0, 1)
			}
		}
		sb.WriteString("  " + yesStyle.Render("Yes") + "  " + noStyle.Render("No"))

	case domain.RiskCritical:
		sb.WriteString(primaryStyle.Render("Type the runbook ID to confirm:"))
		sb.WriteString("\n")
		sb.WriteString(mutedStyle.Render(m.runbook.ID))
		sb.WriteString("\n\n")
		sb.WriteString(textStyle.Render(fmt.Sprintf("> %s▎", m.input)))
	}

	sb.WriteString("\n\n")

	// Footer hints
	switch m.risk {
	case domain.RiskHigh:
		sb.WriteString(mutedStyle.Render("← → toggle  enter confirm  esc cancel"))
	case domain.RiskCritical:
		sb.WriteString(mutedStyle.Render("enter confirm  esc cancel"))
	}

	return sb.String()
}
