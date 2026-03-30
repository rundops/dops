package wizard

import (
	"fmt"
	"strconv"
	"strings"

	"dops/internal/config"
	"dops/internal/domain"
	"dops/internal/theme"
	"dops/internal/vault"

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
	showAll  bool                // when true, show all fields including prefilled
	skipped  map[int]bool        // indices of auto-applied (skipped) fields
	width    int
	styles   *theme.Styles
	cfg      *domain.Config      // config to save into
	vault    *vault.Vault        // encrypted parameter storage
}

func New(rb domain.Runbook, cat domain.Catalog, resolved map[string]string) Model {
	return NewWithOptions(rb, cat, resolved, false)
}

func NewWithOptions(rb domain.Runbook, cat domain.Catalog, resolved map[string]string, promptAll bool) Model {
	m := Model{
		runbook:  rb,
		catalog:  cat,
		resolved: resolved,
		values:   make(map[string]string),
		checked:  make(map[int]bool),
		prefill:  make(map[string]bool),
		skipped:  make(map[int]bool),
		showAll:  promptAll,
		params:   rb.Parameters, // ALL params, not just missing
	}

	// Mark which params have pre-filled values.
	for _, p := range m.params {
		if _, ok := resolved[p.Name]; ok {
			m.prefill[p.Name] = true
		}
	}

	if len(m.params) > 0 {
		// Find the first non-skippable field.
		first := m.nextUnskipped(0)
		if first >= len(m.params) {
			// All fields skippable — auto-apply all and submit.
			m.applyAllSkipped()
			m.current = len(m.params) // signals completion
		} else {
			// Skip and auto-apply fields before the first non-skippable.
			for i := 0; i < first; i++ {
				if m.shouldSkipField(i) {
					m.skipped[i] = true
					m.values[m.params[i].Name] = m.resolved[m.params[i].Name]
				}
			}
			m.initField(first)
		}
	}
	return m
}

// shouldSkipField returns true if a field should be auto-applied (has a saved value
// and showAll is false).
func (m Model) shouldSkipField(idx int) bool {
	if m.showAll {
		return false
	}
	p := m.params[idx]
	return m.prefill[p.Name]
}

// nextUnskipped returns the next field index >= start that should not be skipped.
// Returns len(m.params) if all remaining fields are skippable.
func (m Model) nextUnskipped(start int) int {
	for i := start; i < len(m.params); i++ {
		if !m.shouldSkipField(i) {
			return i
		}
	}
	return len(m.params)
}

// applyAllSkipped marks all prefilled fields as skipped and applies their values.
func (m *Model) applyAllSkipped() {
	for i, p := range m.params {
		if m.prefill[p.Name] {
			m.skipped[i] = true
			m.values[p.Name] = m.resolved[p.Name]
		}
	}
}

// SkippedCount returns the number of auto-applied fields.
func (m Model) SkippedCount() int {
	return len(m.skipped)
}

func (m *Model) SetStyles(s *theme.Styles) {
	m.styles = s
}

// SetStore provides config persistence for the "Save for future runs?" feature.
// Values are saved to the encrypted vault.
func (m *Model) SetStore(cfg *domain.Config, vlt *vault.Vault) {
	m.cfg = cfg
	m.vault = vlt
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
		m.initTextField(p, prefilled)
	case modeSelect:
		m.initSelectField(p, prefilled)
	case modeMultiSelect:
		m.initMultiSelectField(p, prefilled)
	case modeBoolean:
		m.initBoolField(prefilled)
	}
}

// initTextField sets up the text input widget for text, integer, number,
// float, password, filepath, and resourceid parameter types.
func (m *Model) initTextField(p domain.Parameter, prefilled string) {
	ti := textinput.New()
	ti.Focus()
	ti.Prompt = "> "
	// Style textinput to match theme foreground colors (no background —
	// transparent, inherits from terminal background).
	if m.styles != nil {
		textFg := m.styles.Text.GetForeground()
		mutedFg := m.styles.TextMuted.GetForeground()
		ti.SetStyles(textinput.Styles{
			Focused: textinput.StyleState{
				Text:        lipgloss.NewStyle().Foreground(textFg),
				Prompt:      lipgloss.NewStyle().Foreground(textFg),
				Placeholder: lipgloss.NewStyle().Foreground(mutedFg),
				Suggestion:  lipgloss.NewStyle().Foreground(mutedFg),
			},
			Blurred: textinput.StyleState{
				Text:        lipgloss.NewStyle().Foreground(mutedFg),
				Prompt:      lipgloss.NewStyle().Foreground(mutedFg),
				Placeholder: lipgloss.NewStyle().Foreground(mutedFg),
				Suggestion:  lipgloss.NewStyle().Foreground(mutedFg),
			},
		})
	}
	if p.Secret && prefilled == "" {
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
}

// initSelectField positions the cursor on the pre-filled option (if any)
// for single-select parameters.
func (m *Model) initSelectField(p domain.Parameter, prefilled string) {
	m.cursor = 0
	if prefilled != "" {
		for i, opt := range p.Options {
			if opt == prefilled {
				m.cursor = i
				break
			}
		}
	}
}

// initMultiSelectField resets the checked map and pre-checks any previously
// saved multi-select values.
func (m *Model) initMultiSelectField(p domain.Parameter, prefilled string) {
	m.cursor = 0
	m.checked = make(map[int]bool)
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
}

// initBoolField sets the cursor to Yes (0) or No (1) based on the
// pre-filled value.
func (m *Model) initBoolField(prefilled string) {
	m.cursor = 1 // default No
	if prefilled == "true" {
		m.cursor = 0
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
	// All fields were auto-applied — submit immediately.
	if m.current >= len(m.params) {
		return func() tea.Msg {
			return SubmitMsg{
				Runbook: m.runbook,
				Catalog: m.catalog,
				Params:  m.collectParams(),
			}
		}
	}
	// Only focus the text input if the first field uses it.
	if m.fieldMode(m.params[m.current]) == modeTextInput {
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
			return m, func() tea.Msg { return CancelMsg{} }
		}

		// Ctrl+E: reveal all skipped fields.
		if msg.Text == "e" && msg.Mod == tea.ModCtrl && !m.showAll && len(m.skipped) > 0 {
			m.showAll = true
			m.skipped = make(map[int]bool) // clear skipped set
			return m, nil
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
		switch p.Type {
		case domain.ParamInteger:
			if val != "" {
				if _, err := strconv.Atoi(val); err != nil {
					m.err = "must be an integer"
					return m, nil
				}
			}
		case domain.ParamNumber:
			if val != "" {
				n, err := strconv.Atoi(val)
				if err != nil {
					m.err = "must be a number"
					return m, nil
				}
				if n < 0 {
					m.err = "must be a non-negative number"
					return m, nil
				}
			}
		case domain.ParamFloat:
			if val != "" {
				if _, err := strconv.ParseFloat(val, 64); err != nil {
					m.err = "must be a decimal number"
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
// Skips the prompt entirely for local/unscoped parameters since they aren't persisted.
func (m Model) advanceOrSave() (Model, tea.Cmd) {
	if m.changed {
		scope := m.params[m.current].Scope
		if scope == "" || scope == "local" {
			return m.advance()
		}
		m.phase = phaseSave
		m.cursor = 1 // default to "No"
		return m, nil
	}
	return m.advance()
}

func (m *Model) saveCurrentField() {
	if m.vault == nil || m.cfg == nil {
		return
	}
	p := m.params[m.current]
	val := m.values[p.Name]

	// Set the value in the in-memory config (vault stores plaintext — no per-value encryption).
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

	if err := config.Set(m.cfg, keyPath, val); err != nil {
		m.err = fmt.Sprintf("save failed: %v", err)
		return
	}
	if err := m.vault.Save(&m.cfg.Vars); err != nil {
		m.err = fmt.Sprintf("save failed: %v", err)
	}
}

func (m Model) advance() (Model, tea.Cmd) {
	next := m.current + 1

	// Skip prefilled fields.
	for next < len(m.params) && m.shouldSkipField(next) {
		m.skipped[next] = true
		m.values[m.params[next].Name] = m.resolved[m.params[next].Name]
		next++
	}

	if next >= len(m.params) {
		return m, func() tea.Msg {
			return SubmitMsg{
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
	// Skip back over auto-applied fields.
	prev := m.current - 1
	for prev > 0 && m.skipped[prev] {
		prev--
	}
	if m.skipped[prev] {
		return m, nil // all previous fields are skipped
	}
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
		if len(m.skipped) > 0 && !m.showAll {
			hint += "  ctrl+e edit all"
		}
		return hint + "  esc cancel"
	}
}

func (m Model) View() string {
	if len(m.params) == 0 {
		return ""
	}

	var sb strings.Builder

	// --- Header ---
	cmd := BuildCommand(m.runbook, m.collectParams(), m.resolved)
	sb.WriteString(m.renderHeader(cmd))
	sb.WriteString("\n")

	// --- Skipped fields summary ---
	if len(m.skipped) > 0 {
		msg := fmt.Sprintf("Applied %d saved value", len(m.skipped))
		if len(m.skipped) > 1 {
			msg += "s"
		}
		msg += ". Ctrl+E to edit all."
		sb.WriteString(m.mutedStyle().Render(msg))
		sb.WriteString("\n\n")
	}

	// --- Completed fields (excluding skipped) ---
	hasCompleted := false
	for i := 0; i < m.current; i++ {
		if m.skipped[i] {
			continue
		}
		p := m.params[i]
		val := m.values[p.Name]
		if p.Secret {
			val = "••••••••••"
		}
		sb.WriteString(m.renderCompletedField(p.Name, val))
		sb.WriteString("\n")
		hasCompleted = true
	}
	if hasCompleted {
		sb.WriteString("\n")
	}

	// --- Current field or save prompt ---
	if m.phase == phaseSave {
		p := m.params[m.current]
		val := m.values[p.Name]
		if p.Secret {
			val = "••••••••••"
		}
		sb.WriteString(m.renderCompletedField(p.Name, val))
		sb.WriteString("\n\n")
		sb.WriteString(m.renderSavePrompt())
	} else {
		p := m.params[m.current]
		sb.WriteString(m.renderCurrentField(p))
	}

	if m.err != "" {
		sb.WriteString("\n")
		sb.WriteString(m.renderError(m.err))
	}

	sb.WriteString("\n\n")
	sb.WriteString(m.renderFooter())

	return sb.String()
}

// style helpers — return a safe zero style when m.styles is nil.

func (m Model) successStyle() lipgloss.Style {
	if m.styles != nil {
		return m.styles.Success
	}
	return lipgloss.NewStyle()
}

func (m Model) textStyle() lipgloss.Style {
	if m.styles != nil {
		return m.styles.Text
	}
	return lipgloss.NewStyle()
}

func (m Model) primaryStyle() lipgloss.Style {
	if m.styles != nil {
		return m.styles.Primary.Bold(true)
	}
	return lipgloss.NewStyle().Bold(true)
}

func (m Model) mutedStyle() lipgloss.Style {
	if m.styles != nil {
		return m.styles.TextMuted
	}
	return lipgloss.NewStyle()
}

func (m Model) errorStyle() lipgloss.Style {
	if m.styles != nil {
		return m.styles.Error
	}
	return lipgloss.NewStyle()
}

// renderToggle renders a [Yes] / [No] toggle with the given cursor position.
// cursor == 0 highlights Yes, cursor == 1 highlights No.
func (m Model) renderToggle(cursor int) string {
	activeStyle := lipgloss.NewStyle()
	inactiveStyle := lipgloss.NewStyle()
	if m.styles != nil {
		activeStyle = lipgloss.NewStyle().
			Background(m.styles.Primary.GetForeground()).
			Foreground(m.styles.Background.GetForeground()).
			Padding(0, 1)
		inactiveStyle = m.styles.TextMuted.Padding(0, 1)
	}
	yesStyle, noStyle := activeStyle, inactiveStyle
	if cursor != 0 {
		yesStyle, noStyle = inactiveStyle, activeStyle
	}
	return "  " + yesStyle.Render("Yes") + "  " + noStyle.Render("No") + "\n"
}

func (m Model) renderHeader(cmd string) string {
	return m.successStyle().Render("$") + " " + m.textStyle().Bold(true).Render(cmd) + "\n"
}

func (m Model) renderCompletedField(name, value string) string {
	padded := name + ":"
	if len(padded) < 15 {
		padded += strings.Repeat(" ", 15-len(padded))
	} else {
		padded += " "
	}
	return m.mutedStyle().Render(padded + value)
}

func (m Model) renderCurrentField(p domain.Parameter) string {
	primary := m.primaryStyle()
	text := m.textStyle()

	var sb strings.Builder
	sb.WriteString(primary.Render(p.Name+":") + "\n\n")

	switch m.fieldMode(p) {
	case modeTextInput:
		if p.Secret && m.prefill[p.Name] && m.input.Value() == "" {
			sb.WriteString(text.Render("> ") + m.mutedStyle().Render("••••••••••  (enter to keep, type to override)"))
		} else {
			// textinput has its own styles with panelBg — don't wrap in another Render.
			sb.WriteString(m.input.View())
		}
	case modeSelect:
		for i, opt := range p.Options {
			if i == m.cursor {
				sb.WriteString(primary.Render("> ") + text.Render(opt) + "\n")
			} else {
				sb.WriteString("  " + text.Render(opt) + "\n")
			}
		}
	case modeMultiSelect:
		for i, opt := range p.Options {
			check := "[ ]"
			if m.checked[i] {
				check = "[x]"
			}
			if i == m.cursor {
				sb.WriteString(primary.Render("> "+check) + " " + text.Render(opt) + "\n")
			} else {
				sb.WriteString("  " + check + " " + text.Render(opt) + "\n")
			}
		}
	case modeBoolean:
		sb.WriteString(m.renderToggle(m.cursor))
	}

	return sb.String()
}

func (m Model) renderSavePrompt() string {
	var sb strings.Builder
	sb.WriteString(m.primaryStyle().Render("Save for future runs?") + "\n\n")
	sb.WriteString(m.renderToggle(m.cursor))
	return sb.String()
}

func (m Model) renderError(msg string) string {
	return m.errorStyle().Render("! " + msg)
}

func (m Model) renderFooter() string {
	return m.mutedStyle().Render(m.FooterHints())
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

// missingParams returns parameters that are not yet resolved.
func missingParams(params []domain.Parameter, resolved map[string]string) []domain.Parameter {
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
	var sb strings.Builder
	fmt.Fprintf(&sb, "dops run %s", rb.ID)

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
			fmt.Fprintf(&sb, " --param %s=••••••••••", p.Name)
		} else {
			fmt.Fprintf(&sb, " --param %s=%s", p.Name, v)
		}
	}
	return sb.String()
}
