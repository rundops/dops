package tui

import (
	"image/color"

	"dops/internal/domain"
	"dops/internal/tui/confirm"
	"dops/internal/tui/footer"
	"dops/internal/tui/help"
	"dops/internal/tui/palette"
	"dops/internal/tui/wizard"
	"dops/internal/vars"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

func (m App) openPalette() (tea.Model, tea.Cmd) {
	p := palette.New(m.width)
	m.pal = &p
	m.state = statePalette
	return m, nil
}

func (m App) openWizard() (tea.Model, tea.Cmd) {
	if m.selected == nil || m.selCat == nil {
		return m, nil
	}

	resolved := m.resolveVars()

	// If no parameters at all, go straight to confirm.
	if len(m.selected.Parameters) == 0 {
		return m.openConfirm(*m.selected, *m.selCat, resolved)
	}

	// Always show wizard — pre-fills saved values, user can accept or override.
	wiz := wizard.New(wizard.WizardConfig{
		Runbook:  *m.selected,
		Catalog:  *m.selCat,
		Resolved: resolved,
	})
	wiz.SetStyles(m.deps.Styles)
	if m.deps.Config != nil {
		wiz.SetStore(m.deps.Config, m.deps.Vault)
	}
	m.wizard = &wiz
	m.state = stateWizard
	return m, wiz.Init()
}

func (m App) openConfirm(rb domain.Runbook, cat domain.Catalog, params map[string]string) (tea.Model, tea.Cmd) {
	// Low/Medium risk: skip confirmation, execute immediately.
	if rb.RiskLevel == domain.RiskLow || rb.RiskLevel == domain.RiskMedium || rb.RiskLevel == "" {
		return m.startExecution(rb, cat, params)
	}
	c := confirm.New(confirm.Params{
		Runbook:  rb,
		Catalog:  cat,
		Resolved: params,
		Width:    m.width * overlayWidthRatio / overlayWidthDenom,
		Styles:   m.deps.Styles,
	})
	m.conf = &c
	m.state = stateConfirm
	return m, nil
}

func (m App) resolveVars() map[string]string {
	if m.deps.Config == nil || m.selected == nil || m.selCat == nil {
		return make(map[string]string)
	}
	// Vault stores all values as plaintext inside the encrypted blob,
	// so no per-value decryption is needed.
	resolver := vars.NewDefaultResolver()
	return resolver.Resolve(m.deps.Config, m.selCat.Name, m.selected.Name, m.selected.Parameters)
}

func (m App) viewHelpOverlay() tea.View {
	var helpFocus help.FocusTarget
	if m.focus == focusOutput {
		helpFocus = help.FocusOutput
	}
	helpView := help.Render(helpFocus, m.width/2, m.deps.Styles)

	content := lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		helpView,
	)

	footerView := footer.Render(footer.StateHelp, m.width, m.deps.Styles, "")
	content = lipgloss.JoinVertical(lipgloss.Left, content, footerView)

	return tea.NewView(content)
}

func (m App) viewConfirmOverlay() tea.View {
	confView := m.conf.View()

	overlayW := m.width * overlayWidthRatio / overlayWidthDenom
	if overlayW < overlayMinWidth {
		overlayW = overlayMinWidth
	}

	var primaryFg color.Color
	primaryFg = lipgloss.NoColor{}
	if m.deps.Styles != nil {
		primaryFg = m.deps.Styles.Primary.GetForeground()
	}

	overlay := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(primaryFg).
		Padding(1, 2).
		Width(overlayW).
		Render(confView)

	content := lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		overlay,
	)

	footerView := footer.Render(footer.StateConfirm, m.width, m.deps.Styles, "")
	content = lipgloss.JoinVertical(lipgloss.Left, content, footerView)

	return tea.NewView(content)
}

func (m App) viewWizardOverlay() tea.View {
	wizView := m.wizard.View()

	overlayW := m.width * overlayWidthRatio / overlayWidthDenom
	if overlayW < overlayMinWidth {
		overlayW = overlayMinWidth
	}

	// Left accent bar only — transparent background (no Background() set).
	var primaryFg color.Color
	primaryFg = lipgloss.NoColor{}
	if m.deps.Styles != nil {
		primaryFg = m.deps.Styles.Primary.GetForeground()
	}

	overlay := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(primaryFg).
		Padding(1, 2).
		Width(overlayW).
		Render(wizView)

	content := lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		overlay,
	)

	footerView := footer.Render(footer.StateWizard, m.width, m.deps.Styles, "")
	content = lipgloss.JoinVertical(lipgloss.Left, content, footerView)

	return tea.NewView(content)
}

func (m App) viewPaletteOverlay() tea.View {
	palView := m.pal.View()

	overlay := lipgloss.NewStyle().
		Width(m.width).
		Render(palView)

	footerView := footer.Render(footer.StatePalette, m.width, m.deps.Styles, "")
	content := lipgloss.JoinVertical(lipgloss.Left, overlay, footerView)

	return tea.NewView(content)
}
