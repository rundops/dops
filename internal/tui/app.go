package tui

import (
	"context"
	"fmt"
	"image/color"
	"strings"
	"time"

	"dops/internal/adapters"
	"dops/internal/catalog"
	"dops/internal/config"
	"dops/internal/domain"
	"dops/internal/history"
	"dops/internal/executor"
	"dops/internal/theme"
	"dops/internal/tui/confirm"
	"dops/internal/tui/footer"
	"dops/internal/tui/metadata"
	"dops/internal/tui/output"
	"dops/internal/tui/palette"
	"dops/internal/tui/sidebar"
	"dops/internal/tui/wizard"
	"dops/internal/update"
	"dops/internal/vars"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type viewState int

const (
	stateNormal viewState = iota
	stateWizard
	statePalette
	stateConfirm
	stateHelp
)

type focusTarget int

const (
	focusSidebar  focusTarget = iota
	focusOutput
)

// Layout margins and borders for rendering and mouse translation.
const (
	layoutMarginLeft   = 3
	layoutMarginTop    = 3
	layoutMarginBottom = 4
	layoutBorderSize   = 1 // one side of a rounded border
	layoutPadLeft      = 1 // sidebar content left padding
)

// Sidebar sizing constraints.
const (
	sidebarMinWidth = 30
	sidebarMaxWidth = 50
)

// Overlay sizing ratios.
const (
	overlayWidthRatio = 2 // overlay width = terminal width * overlayWidthRatio / overlayWidthDenom
	overlayWidthDenom = 3
	overlayMinWidth   = 50
)

// UI feedback timing.
const copyFlashDuration = 1500 * time.Millisecond

// layoutDims holds pre-computed layout dimensions for the main view.
// Computed once per render/resize cycle, shared across all layout-dependent methods.
type layoutDims struct {
	innerW          int
	sidebarW        int // sidebar logical width
	rightW          int
	contentW        int // content width inside bordered right panels
	panelRows       int
	sidebarContentH int
	sidebarRenderedH int
	sidebarRenderedW int
	metaRenderedH   int
	outputTotalH    int
	outputContentH  int
	outputInnerW    int
	gap             int
	borderSize      int
}

// computeLayout derives all panel dimensions from the terminal size and
// current metadata content. This eliminates duplicated layout math across
// resizeAll, viewNormal, outputPaneBounds, and click handlers.
func (m App) computeLayout() layoutDims {
	gap := 1
	borderSize := layoutBorderSize * 2

	innerW := clamp(m.width-layoutMarginLeft, 1)
	sidebarW := sidebarWidth(innerW)
	rightW := clamp(innerW-sidebarW-borderSize-gap, 1)
	contentW := clamp(rightW-borderSize, 1)
	panelRows := clamp(m.height-layoutMarginTop-1-layoutMarginBottom, 1) // -1 for footer

	sidebarContentH := clamp(panelRows-borderSize-1, 3)

	// Render sidebar/metadata to measure actual pixel sizes.
	sidebarView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		PaddingLeft(1).
		Width(sidebarW).
		Height(sidebarContentH).
		Render("")
	// Measure actual rendered height — don't estimate, since lipgloss
	// Height() behavior with borders can vary.
	sidebarRenderedH := lipgloss.Height(sidebarView)
	sidebarRenderedW := lipgloss.Width(sidebarView)

	metaContent := metadata.Render(metadata.RenderParams{
		Runbook: m.selected,
		Catalog: m.selCat,
		Width:   contentW,
		Copied:  m.copiedFlash,
		Styles:  m.deps.Styles,
	})
	metaView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(contentW).
		Render(metaContent)
	metaRenderedH := lipgloss.Height(metaView)

	outputTotalH := clamp(sidebarRenderedH-metaRenderedH, 3)
	outputContentH := clamp(outputTotalH-borderSize, 1)
	outputInnerW := clamp(contentW-borderSize, 1)

	return layoutDims{
		innerW:           innerW,
		sidebarW:         sidebarW,
		rightW:           rightW,
		contentW:         contentW,
		panelRows:        panelRows,
		sidebarContentH:  sidebarContentH,
		sidebarRenderedH: sidebarRenderedH,
		sidebarRenderedW: sidebarRenderedW,
		metaRenderedH:    metaRenderedH,
		outputTotalH:     outputTotalH,
		outputContentH:   outputContentH,
		outputInnerW:     outputInnerW,
		gap:              gap,
		borderSize:       borderSize,
	}
}

// ProgramRef holds a reference to the tea.Program that can be set after
// program creation. Since AppDeps is copied by value into App, the pointer
// to ProgramRef is shared across all copies, allowing late binding.
type ProgramRef struct {
	P *tea.Program
}

type AppDeps struct {
	Styles     *theme.Styles
	Store      config.ConfigStore
	Runner     executor.Runner
	LogWriter  *adapters.LogWriter
	Config     *domain.Config
	Catalogs   []catalog.CatalogWithRunbooks
	AltScreen  bool
	DryRun     bool
	ProgramRef *ProgramRef
	Version    string                // current build version for update checks
	DopsDir    string                // ~/.dops directory for cache files
	Vault      domain.VaultStore    // encrypted parameter storage
	History    history.ExecutionStore // execution history recording
}

type copiedFlashMsg struct{}

// updateAvailableMsg is sent when a newer version of dops is found.
type updateAvailableMsg struct {
	Version string
}

type App struct {
	sidebar     sidebar.Model
	output      output.Model
	wizard      *wizard.Model
	pal         *palette.Model
	conf        *confirm.Model
	selected    *domain.Runbook
	selCat      *domain.Catalog
	deps        AppDeps
	state       viewState
	width       int
	height      int
	copiedFlash      bool // show "Copied to Clipboard!" in metadata
	focus            focusTarget
	cancelExec       context.CancelFunc
	execRunning      bool
	execRecord       *domain.ExecutionRecord // current execution being recorded
	execLineCount    int                      // output line count for current execution
	execLastLine     string                   // last non-empty output line
	updateAvailable  string                   // non-empty if a newer version exists (e.g. "0.2.0")
}

func NewApp(catalogs []catalog.CatalogWithRunbooks, styles *theme.Styles) App {
	return NewAppWithDeps(AppDeps{
		Styles:   styles,
		Catalogs: catalogs,
	})
}

func NewAppWithDeps(deps AppDeps) App {
	return App{
		sidebar: sidebar.New(deps.Catalogs, 20, deps.Styles),
		output:  output.New(60, 20, deps.Styles),
		deps:    deps,
		// width/height start at 0 — View() returns empty until WindowSizeMsg arrives
	}
}

func (m *App) SetConfig(cfg *domain.Config) {
	m.deps.Config = cfg
}

// Test query methods — expose internal state for assertions.
func (m App) Selected() *domain.Runbook      { return m.selected }
func (m App) SelectedCatalog() *domain.Catalog { return m.selCat }
func (m App) ViewState() viewState            { return m.state }
func (m App) Width() int                      { return m.width }
func (m App) Height() int                     { return m.height }
func (m App) HasWizard() bool                 { return m.wizard != nil }
func (m App) HasPalette() bool                { return m.pal != nil }
func (m App) Output() output.Model            { return m.output }


func (m App) Init() tea.Cmd {
	cmds := []tea.Cmd{m.sidebar.Init()}

	if m.deps.Version != "" && m.deps.DopsDir != "" {
		version := m.deps.Version
		dopsDir := m.deps.DopsDir
		cmds = append(cmds, func() tea.Msg {
			r := update.Check(version, dopsDir)
			if r.Available {
				return updateAvailableMsg{Version: r.Latest}
			}
			return nil
		})
	}

	return tea.Batch(cmds...)
}

// resizeAll computes layout dimensions from the current terminal size and
// persists them on the sidebar and output models. Must be called from Update
// (not View) so changes survive across message cycles.
func (m *App) resizeAll() {
	l := m.computeLayout()
	m.sidebar.SetHeight(l.sidebarContentH)
	m.output.SetSize(l.outputInnerW, l.outputContentH)
}

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeAll()
		return m, nil
	case tea.KeyPressMsg:
		if result, cmd, handled := m.handleKeyPress(msg); handled {
			return result, cmd
		}
	}

	// Handle domain messages (selection, execution, wizard, copy flash, etc.).
	if result, cmd, handled := m.handleAppMessage(msg); handled {
		return result, cmd
	}

	// Switch focus on hover: any mouse event over a pane focuses it.
	if m.state == stateNormal {
		if target, ok := m.focusTargetFromMouse(msg); ok {
			m.focus = target
		}
	}

	return m.routeToComponent(msg)
}

// handleKeyPress processes keyboard input for normal and help states.
// Returns (model, cmd, handled).
func (m App) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if m.state == stateNormal {
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit, true
		case "ctrl+shift+p":
			result, cmd := m.openPalette()
			return result, cmd, true
		case "tab", "shift+tab":
			if m.focus == focusSidebar {
				m.focus = focusOutput
			} else {
				m.focus = focusSidebar
			}
			return m, nil, true
		case "ctrl+x":
			if m.execRunning && m.cancelExec != nil {
				m.cancelExec()
			}
			return m, nil, true
		case "?":
			m.state = stateHelp
			return m, nil, true
		}
	}
	if m.state == stateHelp {
		if msg.String() == "?" || msg.String() == "escape" || msg.Code == tea.KeyEscape {
			m.state = stateNormal
		}
		return m, nil, true
	}
	return m, nil, false
}

// handleAppMessage processes typed domain messages (sidebar selection,
// execution lifecycle, wizard/confirm/palette transitions, copy flash).
// Returns (model, cmd, handled).
func (m App) handleAppMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case sidebar.RunbookSelectedMsg:
		rb := msg.Runbook
		cat := msg.Catalog
		m.selected = &rb
		m.selCat = &cat
		return m, nil, true

	case sidebar.RunbookExecuteMsg:
		rb := msg.Runbook
		cat := msg.Catalog
		m.selected = &rb
		m.selCat = &cat
		result, cmd := m.openWizard()
		return result, cmd, true

	case executionDoneMsg:
		m.execRunning = false
		m.cancelExec = nil
		m.output, _ = m.output.Update(output.ExecutionDoneMsg{LogPath: msg.LogPath, Err: msg.Err})
		// Record execution to history.
		if m.execRecord != nil && m.deps.History != nil {
			exitCode := 0
			if msg.Err != nil {
				exitCode = 1
			}
			m.execRecord.Complete(exitCode, m.execLineCount, m.execLastLine)
			_ = m.deps.History.Record(m.execRecord)
			m.execRecord = nil
		}
		return m, nil, true

	case output.OutputLineMsg:
		m.output, _ = m.output.Update(msg)
		m.execLineCount++
		if text := strings.TrimSpace(msg.Text); text != "" {
			m.execLastLine = text
		}
		return m, nil, true

	case output.ExecutionDoneMsg:
		m.output, _ = m.output.Update(msg)
		return m, nil, true

	case wizard.SaveFieldMsg:
		var saveErr error
		if m.deps.Config == nil || m.deps.Vault == nil {
			saveErr = fmt.Errorf("persistence not configured")
		} else {
			keyPath := vars.VarKeyPath(msg.Scope, msg.ParamName, msg.CatalogName, msg.RunbookName)
			if err := config.Set(m.deps.Config, keyPath, msg.Value); err != nil {
				saveErr = err
			} else if err := m.deps.Vault.Save(&m.deps.Config.Vars); err != nil {
				saveErr = err
			}
		}
		return m, func() tea.Msg { return wizard.SaveFieldResultMsg{Err: saveErr} }, true

	case wizard.SubmitMsg:
		m.state = stateNormal
		m.wizard = nil
		result, cmd := m.openConfirm(msg.Runbook, msg.Catalog, msg.Params)
		return result, cmd, true

	case confirm.AcceptMsg:
		m.state = stateNormal
		m.conf = nil
		result, cmd := m.startExecution(msg.Runbook, msg.Catalog, msg.Params)
		return result, cmd, true

	case confirm.CancelMsg:
		m.state = stateNormal
		m.conf = nil
		return m, nil, true

	case wizard.CancelMsg:
		m.state = stateNormal
		m.wizard = nil
		return m, nil, true

	case palette.PaletteSelectMsg:
		m.state = stateNormal
		m.pal = nil
		return m, nil, true

	case palette.PaletteCancelMsg:
		m.state = stateNormal
		m.pal = nil
		return m, nil, true

	case copiedFlashMsg:
		m.copiedFlash = false
		m.output.SetCopyFlash(false)
		return m, nil, true

	case output.CopyFlashExpiredMsg:
		m.output, _ = m.output.Update(msg)
		return m, nil, true

	case output.CopiedRegionFlashMsg:
		m.output.SetCopiedHeader(false)
		m.output.SetCopiedFooter(false)
		return m, nil, true

	case output.SelectionCompleteMsg:
		text := m.extractSelectionFromView()
		if text != "" && m.output.TryCopy() {
			return m, tea.Batch(
				tea.SetClipboard(text),
				tea.Tick(copyFlashDuration, func(time.Time) tea.Msg {
					return output.CopyFlashExpiredMsg{}
				}),
			), true
		}
		return m, nil, true

	case updateAvailableMsg:
		m.updateAvailable = msg.Version
		return m, nil, true
	}

	return m, nil, false
}

// routeToComponent forwards messages to the currently active component
// (sidebar, output, wizard, palette, or confirm overlay).
func (m App) routeToComponent(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateNormal:
		// Check for click-to-copy targets.
		if isMouseClick(msg) {
			if _, cmd := m.handleMetadataClick(msg); cmd != nil {
				if m.output.TryLock() {
					m.copiedFlash = true
					return m, cmd
				}
			}
			if cmd := m.handleOutputClick(msg); cmd != nil {
				return m, cmd
			}
		}

		// Route based on focus target.
		if m.focus == focusOutput {
			if isMouseMsg(msg) {
				// Sidebar clicks still go to sidebar.
				translated, inSidebar := m.translateMouseForSidebar(msg)
				if inSidebar {
					var cmd tea.Cmd
					m.sidebar, cmd = m.sidebar.Update(translated)
					return m, cmd
				}
				var cmd tea.Cmd
				m.output, cmd = m.output.Update(msg)
				return m, cmd
			}
			var cmd tea.Cmd
			m.output, cmd = m.output.Update(msg)
			return m, cmd
		}

		translated, inSidebar := m.translateMouseForSidebar(msg)
		if !inSidebar && isMouseMsg(msg) {
			return m, nil
		}
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(translated)
		return m, cmd

	case stateWizard:
		if m.wizard != nil {
			var cmd tea.Cmd
			wiz := *m.wizard
			wiz, cmd = wiz.Update(msg)
			m.wizard = &wiz
			return m, cmd
		}

	case statePalette:
		if m.pal != nil {
			var cmd tea.Cmd
			p := *m.pal
			p, cmd = p.Update(msg)
			m.pal = &p
			return m, cmd
		}

	case stateConfirm:
		if m.conf != nil {
			var cmd tea.Cmd
			c := *m.conf
			c, cmd = c.Update(msg)
			m.conf = &c
			return m, cmd
		}
	}

	return m, nil
}

func (m App) View() tea.View {
	// Guard: before WindowSizeMsg arrives, width/height are defaults (80x24).
	// In alt screen, BubbleTea sends WindowSizeMsg on startup, but View()
	// may be called first. Return minimal content to avoid broken layout.
	if m.width == 0 || m.height == 0 {
		v := tea.NewView("")
		v.AltScreen = m.deps.AltScreen
		return v
	}

	var v tea.View

	if m.state == stateHelp {
		v = m.viewHelpOverlay()
	} else if m.state == stateConfirm && m.conf != nil {
		v = m.viewConfirmOverlay()
	} else if m.state == stateWizard && m.wizard != nil {
		v = m.viewWizardOverlay()
	} else if m.state == statePalette && m.pal != nil {
		v = m.viewPaletteOverlay()
	} else {
		v = m.viewNormal()
	}

	v.AltScreen = m.deps.AltScreen
	v.MouseMode = tea.MouseModeCellMotion
	if m.deps.Styles != nil {
		v.BackgroundColor = m.deps.Styles.Background.GetForeground()
	}
	return v
}

func (m App) themeColors() (borderColor, activeBorderColor color.Color) {
	borderColor = lipgloss.NoColor{}
	activeBorderColor = lipgloss.NoColor{}
	if m.deps.Styles != nil {
		borderColor = m.deps.Styles.Border.GetForeground()
		activeBorderColor = m.deps.Styles.BorderActive.GetForeground()
	}
	return borderColor, activeBorderColor
}

func (m App) renderSidebar(l layoutDims) string {
	borderColor, activeBorderColor := m.themeColors()

	sidebarBorderColor := borderColor
	if m.focus == focusSidebar {
		sidebarBorderColor = activeBorderColor
	}
	m.sidebar.SetHeight(l.sidebarContentH)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(sidebarBorderColor).
		PaddingLeft(1).
		Width(l.sidebarW).
		Height(l.sidebarContentH).
		Render(m.sidebar.View())
}

func (m App) renderRightPanel(l layoutDims, sidebarView string) string {
	borderColor, activeBorderColor := m.themeColors()

	// --- Metadata ---
	metaContent := metadata.Render(metadata.RenderParams{
		Runbook: m.selected,
		Catalog: m.selCat,
		Width:   l.contentW,
		Copied:  m.copiedFlash,
		Styles:  m.deps.Styles,
	})

	// Cap metadata height so the output pane keeps a minimum of 3 rows.
	// The meta border adds 2 rows, so max content lines = sidebarH - 3(output min) - 2(border).
	actualSidebarH := lipgloss.Height(sidebarView)
	metaMaxContentH := clamp(actualSidebarH-3-l.borderSize, 1)
	metaLines := strings.Split(metaContent, "\n")
	if len(metaLines) > metaMaxContentH {
		metaContent = strings.Join(metaLines[:metaMaxContentH], "\n")
	}

	metaView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(l.contentW).
		Render(metaContent)
	if m.copiedFlash {
		metaView = injectBorderBadge(metaView, "Copied to Clipboard!", m.deps.Styles)
	}

	// --- Output ---
	// Derive output height from actual rendered sidebar and metadata heights.
	actualMetaH := lipgloss.Height(metaView)
	outputTotalH := clamp(actualSidebarH-actualMetaH, 3)
	outputContentH := clamp(outputTotalH-l.borderSize, 1)

	outputBorderColor := borderColor
	if m.focus == focusOutput {
		outputBorderColor = activeBorderColor
	}
	m.output.SetFocused(m.focus == focusOutput)
	outputView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(outputBorderColor).
		Width(l.contentW).
		Height(outputTotalH).
		Render(m.output.ViewWithSize(l.outputInnerW, outputContentH))
	if m.output.CopyFlash() {
		outputView = injectBorderBadge(outputView, "Copied to Clipboard!", m.deps.Styles)
	}

	return lipgloss.JoinVertical(lipgloss.Left, metaView, outputView)
}

func (m App) viewNormal() tea.View {
	l := m.computeLayout()

	sidebarView := m.renderSidebar(l)
	rightPanel := m.renderRightPanel(l, sidebarView)

	// --- Compose ---
	panels := lipgloss.JoinHorizontal(lipgloss.Top,
		sidebarView,
		strings.Repeat(" ", l.gap),
		rightPanel,
	)
	body := lipgloss.NewStyle().
		MarginLeft(layoutMarginLeft).
		MarginTop(layoutMarginTop).
		Render(panels)

	// Measure the actual rendered panel width so the footer's update badge
	// right-aligns exactly with the output pane's right border.
	footerW := lipgloss.Width(panels) - 1 // align badge inside the output pane's right border
	footerView := lipgloss.NewStyle().
		MarginLeft(layoutMarginLeft - 1).
		Render(footer.Render(appFooterState(m.state), footerW, m.deps.Styles, m.updateAvailable))

	content := lipgloss.JoinVertical(lipgloss.Left, body, footerView)
	content = lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(content)

	// Selection highlight confined to the output pane.
	sel := m.output.Selection()
	if sel.Active && !sel.IsEmpty() {
		bTop, bBottom, bLeft, bRight := m.outputPaneBounds()
		content = applySelectionHighlight(content, sel, m.deps.Styles,
			selectionBounds{top: bTop, bottom: bBottom, left: bLeft, right: bRight})
	}

	return tea.NewView(content)
}

// clamp returns v if v >= min, otherwise min.
func clamp(v, min int) int {
	if v < min {
		return min
	}
	return v
}

func appFooterState(s viewState) footer.State {
	switch s {
	case stateWizard:
		return footer.StateWizard
	case statePalette:
		return footer.StatePalette
	case stateConfirm:
		return footer.StateConfirm
	case stateHelp:
		return footer.StateHelp
	default:
		return footer.StateNormal
	}
}
