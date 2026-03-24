package wizard

import (
	"fmt"
	"strings"

	"dops/internal/config"
	"dops/internal/domain"
	"dops/internal/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type fieldMode int

const (
	modeTextInput fieldMode = iota
	modeSelect
	modeMultiSelect
	modeBoolean
)

type wizardPhase int

const (
	phaseInput   wizardPhase = iota // collecting user input
	phaseSave                       // asking "Save for future runs?"
)

type Model struct {
	runbook  domain.Runbook
	catalog  domain.Catalog
	resolved map[string]string   // pre-filled values from config
	params   []domain.Parameter  // ALL params (not just missing)
	current  int                 // index of current field
	values   map[string]string   // collected values
	input    textinput.Model     // for text/integer/password/filepath/resourceid
	cursor   int                 // for select/multiselect/boolean/save cursor
	checked  map[int]bool        // for multiselect toggles
	err      string              // validation error
	phase    wizardPhase         // current phase (input or save confirm)
	changed  bool                // whether user typed a new value (vs accepting pre-fill)
	prefill  map[string]bool     // tracks which fields had saved values
	width    int
	styles   *theme.Styles
	store    config.ConfigStore  // for saving values
	cfg      *domain.Config      // config to save into
}

func New(rb domain.Runbook, cat domain.Catalog, resolved map[string]string) Model {
	m := Model{
		runbook:  rb,
		catalog:  cat,
		resolved: resolved,
		values:   make(map[string]string),
		checked:  make(map[int]bool),
		prefill:  make(map[string]bool),
		params:   rb.Parameters, // ALL params, not just missing
	}

	// Mark which params have pre-filled values.
	for _, p := range m.params {
		if _, ok := resolved[p.Name]; ok {
			m.prefill[p.Name] = true
		}
	}

	if len(m.params) > 0 {
		m.initField(0)
	}
	return m
}

func (m *Model) SetStyles(s *theme.Styles) {
	m.styles = s
}

// SetStore provides config persistence for the "Save for future runs?" feature.
func (m *Model) SetStore(store config.ConfigStore, cfg *domain.Config) {
	m.store = store
	m.cfg = cfg
}

func (m *Model) initField(idx int) {
	if idx >= len(m.params) {
		return
	}
	m.current = idx
	m.cursor = 0
	m.err = ""
	m.phase = phaseInput
	m.changed = false
	p := m.params[idx]
	prefilled := m.resolved[p.Name]

	switch m.fieldMode(p) {
	case modeTextInput:
		ti := textinput.New()
		ti.Prompt = "> "
		if p.Secret && prefilled == "" {
			// New secret — use password echo mode.
			ti.EchoMode = textinput.EchoPassword
		} else if p.Secret && prefilled != "" {
			// Existing secret — don't use password echo, render dots manually.
			// EchoPassword will be enabled once user starts typing (handled in Update).
		} else if prefilled != "" {
			ti.SetValue(prefilled)
		} else if p.Default != nil {
			ti.SetValue(fmt.Sprintf("%v", p.Default))
		}
		m.input = ti
	case modeSelect:
		m.cursor = 0
		// Set cursor to the pre-filled option if available.
		if prefilled != "" {
			for i, opt := range p.Options {
				if opt == prefilled {
					m.cursor = i
					break
				}
			}
		}
	case modeMultiSelect:
		m.cursor = 0
		m.checked = make(map[int]bool)
		// Pre-check saved multi-select values.
		if prefilled != "" {
			selected := strings.Split(prefilled, ", ")
			selSet := make(map[string]bool)
			for _, s := range selected {
				selSet[strings.TrimSpace(s)] = true
			}
			for i, opt := range p.Options {
				if selSet[opt] {
					m.checked[i] = true
				}
			}
		}
	case modeBoolean:
		m.cursor = 1 // default No
		if prefilled == "true" {
			m.cursor = 0
		}
	}
}

func (m Model) fieldMode(p domain.Parameter) fieldMode {
	switch p.Type {
	case domain.ParamSelect:
		return modeSelect
	case domain.ParamMultiSelect:
		return modeMultiSelect
	case domain.ParamBoolean:
		return modeBoolean
	default:
		return modeTextInput
	}
}

func (m Model) Init() tea.Cmd {
	if len(m.params) == 0 {
		return nil
	}
	// Only focus the text input if the first field uses it.
	if m.fieldMode(m.params[0]) == modeTextInput {
		return m.input.Focus()
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if len(m.params) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.Code == tea.KeyEscape {
			return m, func() tea.Msg { return WizardCancelMsg{} }
		}

		// Save confirmation phase.
		if m.phase == phaseSave {
			return m.updateSaveConfirm(msg)
		}

		p := m.params[m.current]
		mode := m.fieldMode(p)

		switch mode {
		case modeTextInput:
			return m.updateTextInput(msg)
		case modeSelect:
			return m.updateSelect(msg, p)
		case modeMultiSelect:
			return m.updateMultiSelect(msg, p)
		case modeBoolean:
			return m.updateBoolean(msg, p)
		}
	}

	// Forward to textinput for cursor blink etc.
	if m.phase == phaseInput && m.current < len(m.params) && m.fieldMode(m.params[m.current]) == modeTextInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) updateTextInput(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	p := m.params[m.current]
	switch {
	case msg.Code == tea.KeyEnter:
		val := strings.TrimSpace(m.input.Value())
		// Secret field with empty input and saved value → keep saved value.
		if p.Secret && val == "" && m.prefill[p.Name] {
			m.values[p.Name] = m.resolved[p.Name]
			m.changed = false
			return m.advance()
		}
		if p.Required && val == "" {
			m.err = "required field"
			return m, nil
		}
		if p.Type == domain.ParamInteger && val != "" {
			for _, c := range val {
				if c < '0' || c > '9' {
					m.err = "must be an integer"
					return m, nil
				}
			}
		}
		m.values[p.Name] = val
		// Check if value was changed from pre-fill.
		m.changed = !m.prefill[p.Name] || val != m.resolved[p.Name]
		return m.advanceOrSave()
	case msg.String() == "shift+tab":
		return m.goBack()
	default:
		m.err = ""
		// Switch to password echo on first keystroke for secret pre-filled fields.
		if p.Secret && m.prefill[p.Name] && m.input.EchoMode != textinput.EchoPassword {
			m.input.EchoMode = textinput.EchoPassword
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m Model) updateSelect(msg tea.KeyPressMsg, p domain.Parameter) (Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyUp || msg.Text == "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case msg.Code == tea.KeyDown || msg.Text == "j":
		if m.cursor < len(p.Options)-1 {
			m.cursor++
		}
	case msg.Code == tea.KeyEnter:
		if m.cursor < len(p.Options) {
			val := p.Options[m.cursor]
			m.values[p.Name] = val
			m.changed = !m.prefill[p.Name] || val != m.resolved[p.Name]
			return m.advanceOrSave()
		}
	case msg.String() == "shift+tab":
		return m.goBack()
	}
	return m, nil
}

func (m Model) updateMultiSelect(msg tea.KeyPressMsg, p domain.Parameter) (Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyUp || msg.Text == "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case msg.Code == tea.KeyDown || msg.Text == "j":
		if m.cursor < len(p.Options)-1 {
			m.cursor++
		}
	case msg.Text == " ":
		m.checked[m.cursor] = !m.checked[m.cursor]
	case msg.Code == tea.KeyEnter:
		var selected []string
		for i, o := range p.Options {
			if m.checked[i] {
				selected = append(selected, o)
			}
		}
		val := strings.Join(selected, ", ")
		m.values[p.Name] = val
		m.changed = !m.prefill[p.Name] || val != m.resolved[p.Name]
		return m.advanceOrSave()
	case msg.String() == "shift+tab":
		return m.goBack()
	}
	return m, nil
}

func (m Model) updateBoolean(msg tea.KeyPressMsg, p domain.Parameter) (Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyLeft || msg.Text == "h" || msg.Code == tea.KeyTab:
		m.cursor = 0
	case msg.Code == tea.KeyRight || msg.Text == "l":
		m.cursor = 1
	case msg.Text == "y" || msg.Text == "Y":
		m.values[p.Name] = "true"
		m.changed = !m.prefill[p.Name] || "true" != m.resolved[p.Name]
		return m.advanceOrSave()
	case msg.Text == "n" || msg.Text == "N":
		m.values[p.Name] = "false"
		m.changed = !m.prefill[p.Name] || "false" != m.resolved[p.Name]
		return m.advanceOrSave()
	case msg.Code == tea.KeyEnter:
		val := "false"
		if m.cursor == 0 {
			val = "true"
		}
		m.values[p.Name] = val
		m.changed = !m.prefill[p.Name] || val != m.resolved[p.Name]
		return m.advanceOrSave()
	case msg.String() == "shift+tab":
		return m.goBack()
	}
	return m, nil
}

func (m Model) updateSaveConfirm(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyLeft || msg.Text == "h" || msg.Code == tea.KeyTab:
		m.cursor = 0 // Yes
	case msg.Code == tea.KeyRight || msg.Text == "l":
		m.cursor = 1 // No
	case msg.Text == "y" || msg.Text == "Y":
		m.saveCurrentField()
		return m.advance()
	case msg.Text == "n" || msg.Text == "N":
		return m.advance()
	case msg.Code == tea.KeyEnter:
		if m.cursor == 0 {
			m.saveCurrentField()
		}
		return m.advance()
	}
	return m, nil
}

// advanceOrSave shows the save prompt if value was changed, otherwise advances.
func (m Model) advanceOrSave() (Model, tea.Cmd) {
	if m.changed {
		m.phase = phaseSave
		m.cursor = 1 // default to "No"
		return m, nil
	}
	return m.advance()
}

func (m *Model) saveCurrentField() {
	if m.store == nil || m.cfg == nil {
		return
	}
	p := m.params[m.current]
	val := m.values[p.Name]

	var keyPath string
	switch p.Scope {
	case "global":
		keyPath = fmt.Sprintf("vars.global.%s", p.Name)
	case "catalog":
		keyPath = fmt.Sprintf("vars.catalog.%s.%s", m.catalog.Name, p.Name)
	case "runbook":
		keyPath = fmt.Sprintf("vars.catalog.%s.runbooks.%s.%s", m.catalog.Name, m.runbook.Name, p.Name)
	default:
		keyPath = fmt.Sprintf("vars.global.%s", p.Name)
	}
	config.Set(m.cfg, keyPath, val)
	m.store.Save(m.cfg)
}

func (m Model) advance() (Model, tea.Cmd) {
	next := m.current + 1
	if next >= len(m.params) {
		return m, func() tea.Msg {
			return WizardSubmitMsg{
				Runbook: m.runbook,
				Catalog: m.catalog,
				Params:  m.collectParams(),
			}
		}
	}
	m.initField(next)
	if m.fieldMode(m.params[next]) == modeTextInput {
		return m, m.input.Focus()
	}
	return m, nil
}

func (m Model) goBack() (Model, tea.Cmd) {
	if m.current == 0 {
		return m, nil
	}
	prev := m.current - 1
	delete(m.values, m.params[prev].Name)
	m.initField(prev)
	if m.fieldMode(m.params[prev]) == modeTextInput {
		return m, m.input.Focus()
	}
	return m, nil
}

// FooterHints returns context-sensitive keybinding hints.
func (m Model) FooterHints() string {
	if len(m.params) == 0 || m.current >= len(m.params) {
		return ""
	}

	if m.phase == phaseSave {
		return "← → toggle  enter confirm"
	}

	p := m.params[m.current]
	switch m.fieldMode(p) {
	case modeSelect:
		return "↑↓ navigate  enter select  esc cancel"
	case modeMultiSelect:
		return "Space toggle  Up/Down navigate  Enter confirm"
	case modeBoolean:
		return "← → toggle  enter confirm  esc cancel"
	default:
		hint := "enter next"
		if p.Secret && m.prefill[p.Name] {
			hint = "enter accept  type to override"
		}
		if m.current > 0 {
			hint += "  shift+tab back"
		}
		return hint + "  esc cancel"
	}
}

func (m Model) View() string {
	if len(m.params) == 0 {
		return ""
	}

	var b strings.Builder

	// --- Header ---
	cmd := BuildCommand(m.runbook, m.collectParams(), m.resolved)
	b.WriteString(m.renderHeader(cmd))
	b.WriteString("\n")

	// --- Completed fields ---
	for i := 0; i < m.current; i++ {
		p := m.params[i]
		val := m.values[p.Name]
		if p.Secret {
			val = "••••••••••"
		}
		b.WriteString(m.renderCompletedField(p.Name, val))
		b.WriteString("\n")
	}
	if m.current > 0 {
		b.WriteString("\n")
	}

	// --- Current field or save prompt ---
	if m.phase == phaseSave {
		p := m.params[m.current]
		val := m.values[p.Name]
		if p.Secret {
			val = "••••••••••"
		}
		b.WriteString(m.renderCompletedField(p.Name, val))
		b.WriteString("\n\n")
		b.WriteString(m.renderSavePrompt())
	} else {
		p := m.params[m.current]
		b.WriteString(m.renderCurrentField(p))
	}

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(m.renderError(m.err))
	}

	b.WriteString("\n\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m Model) renderHeader(cmd string) string {
	successStyle := lipgloss.NewStyle()
	textStyle := lipgloss.NewStyle().Bold(true)
	if m.styles != nil {
		successStyle = m.styles.Success
		textStyle = m.styles.Text.Bold(true)
	}
	return successStyle.Render("$") + " " + textStyle.Render(cmd) + "\n"
}

func (m Model) renderCompletedField(name, value string) string {
	mutedStyle := lipgloss.NewStyle()
	if m.styles != nil {
		mutedStyle = m.styles.TextMuted
	}
	padded := name + ":"
	if len(padded) < 15 {
		padded += strings.Repeat(" ", 15-len(padded))
	} else {
		padded += " "
	}
	return mutedStyle.Render(padded + value)
}

func (m Model) renderCurrentField(p domain.Parameter) string {
	primaryStyle := lipgloss.NewStyle().Bold(true)
	textStyle := lipgloss.NewStyle()
	if m.styles != nil {
		primaryStyle = m.styles.Primary.Bold(true)
		textStyle = m.styles.Text
	}

	var b strings.Builder
	b.WriteString(primaryStyle.Render(p.Name+":") + "\n\n")

	switch m.fieldMode(p) {
	case modeTextInput:
		// Secret field with saved value and empty input: show dots instead of textinput.
		if p.Secret && m.prefill[p.Name] && m.input.Value() == "" {
			mutedStyle := lipgloss.NewStyle()
			if m.styles != nil {
				mutedStyle = m.styles.TextMuted
			}
			b.WriteString(textStyle.Render("> ") + mutedStyle.Render("••••••••••  (enter to keep, type to override)"))
		} else {
			b.WriteString(textStyle.Render(m.input.View()))
		}
	case modeSelect:
		for i, opt := range p.Options {
			if i == m.cursor {
				b.WriteString(primaryStyle.Render("> ") + textStyle.Render(opt) + "\n")
			} else {
				b.WriteString("  " + textStyle.Render(opt) + "\n")
			}
		}
	case modeMultiSelect:
		for i, opt := range p.Options {
			check := "[ ]"
			if m.checked[i] {
				check = "[x]"
			}
			if i == m.cursor {
				b.WriteString(primaryStyle.Render("> "+check) + " " + textStyle.Render(opt) + "\n")
			} else {
				b.WriteString("  " + check + " " + textStyle.Render(opt) + "\n")
			}
		}
	case modeBoolean:
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
		b.WriteString("  " + yesStyle.Render("Yes") + "  " + noStyle.Render("No") + "\n")
	}

	return b.String()
}

func (m Model) renderSavePrompt() string {
	primaryStyle := lipgloss.NewStyle().Bold(true)
	if m.styles != nil {
		primaryStyle = m.styles.Primary.Bold(true)
	}

	var b strings.Builder
	b.WriteString(primaryStyle.Render("Save for future runs?") + "\n\n")

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
	b.WriteString("  " + yesStyle.Render("Yes") + "  " + noStyle.Render("No") + "\n")

	return b.String()
}

func (m Model) renderError(msg string) string {
	errStyle := lipgloss.NewStyle()
	if m.styles != nil {
		errStyle = m.styles.Error
	}
	return errStyle.Render("! " + msg)
}

func (m Model) renderFooter() string {
	mutedStyle := lipgloss.NewStyle()
	if m.styles != nil {
		mutedStyle = m.styles.TextMuted
	}
	return mutedStyle.Render(m.FooterHints())
}

func (m Model) collectParams() map[string]string {
	result := make(map[string]string)
	for k, v := range m.resolved {
		result[k] = v
	}
	for k, v := range m.values {
		if v != "" {
			result[k] = v
		}
	}
	return result
}

// ShouldSkip returns true when all required parameters are already resolved.
func ShouldSkip(params []domain.Parameter, resolved map[string]string) bool {
	for _, p := range params {
		if !p.Required {
			continue
		}
		if _, ok := resolved[p.Name]; !ok {
			return false
		}
	}
	return true
}

// MissingParams returns parameters that are not yet resolved.
func MissingParams(params []domain.Parameter, resolved map[string]string) []domain.Parameter {
	var missing []domain.Parameter
	for _, p := range params {
		if _, ok := resolved[p.Name]; !ok {
			missing = append(missing, p)
		}
	}
	return missing
}

// BuildCommand formats the dops run command for display.
// Only includes --param flags for values NOT in the config (overrides only).
func BuildCommand(rb domain.Runbook, params map[string]string, configParams ...map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "dops run %s", rb.ID)

	// If config params provided, only show params that differ or aren't in config.
	var saved map[string]string
	if len(configParams) > 0 {
		saved = configParams[0]
	}

	for _, p := range rb.Parameters {
		v, ok := params[p.Name]
		if !ok {
			continue
		}
		// Skip if the value is from config (unchanged).
		if saved != nil {
			if sv, inConfig := saved[p.Name]; inConfig && sv == v {
				continue
			}
		}
		if p.Secret {
			fmt.Fprintf(&b, " --param %s=••••••••••", p.Name)
		} else {
			fmt.Fprintf(&b, " --param %s=%s", p.Name, v)
		}
	}
	return b.String()
}
