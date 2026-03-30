package palette

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type Command struct {
	Name        string
	Description string
}

type Model struct {
	commands []Command
	filtered []Command
	query    string
	cursor   int
	width    int
}

func New(width int) Model {
	cmds := defaultCommands()
	return Model{
		commands: cmds,
		filtered: cmds,
		width:    width,
	}
}

func defaultCommands() []Command {
	return []Command{
		{Name: "theme: set", Description: "Pick from available themes"},
		{Name: "config: set", Description: "Set a config value by key path"},
		{Name: "config: view", Description: "Display current config (secrets masked)"},
		{Name: "config: delete", Description: "Remove a saved input by key path"},
		{Name: "secrets: re-encrypt", Description: "Re-encrypt all secrets with current key"},
	}
}

// Test query method.
func (m Model) Cursor() int { return m.cursor }

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case msg.Code == tea.KeyEscape:
			return m, func() tea.Msg { return PaletteCancelMsg{} }

		case msg.Code == tea.KeyEnter:
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				selected := m.filtered[m.cursor]
				return m, func() tea.Msg { return PaletteSelectMsg{Command: selected} }
			}
			return m, nil

		case msg.Code == tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil

		case msg.Code == tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case msg.Code == tea.KeyBackspace:
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.applyFilter()
			}
			return m, nil

		default:
			if msg.Text != "" {
				m.query += msg.Text
				m.applyFilter()
			}
			return m, nil
		}
	}

	return m, nil
}

func (m *Model) applyFilter() {
	m.cursor = 0
	if m.query == "" {
		m.filtered = m.commands
		return
	}

	q := strings.ToLower(m.query)
	m.filtered = nil
	for _, cmd := range m.commands {
		if strings.Contains(strings.ToLower(cmd.Name), q) ||
			strings.Contains(strings.ToLower(cmd.Description), q) {
			m.filtered = append(m.filtered, cmd)
		}
	}
}

func (m Model) Filtered() []Command {
	return m.filtered
}

func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("  > %s\n", m.query))
	sb.WriteString(strings.Repeat("─", m.width) + "\n")

	for i, cmd := range m.filtered {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		fmt.Fprintf(&sb, "%s%-20s  %s\n", prefix, cmd.Name, cmd.Description)
	}

	if len(m.filtered) == 0 {
		sb.WriteString("  No matching commands\n")
	}

	return sb.String()
}
