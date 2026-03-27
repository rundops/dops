package tui

import (
	"context"
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"time"

	"dops/internal/adapters"
	"dops/internal/catalog"
	"dops/internal/config"
	"dops/internal/domain"
	"dops/internal/executor"
	"dops/internal/theme"
	"dops/internal/tui/confirm"
	"dops/internal/tui/footer"
	"dops/internal/tui/help"
	"dops/internal/tui/metadata"
	"dops/internal/tui/output"
	"dops/internal/tui/palette"
	"dops/internal/tui/sidebar"
	"dops/internal/tui/wizard"
	"dops/internal/update"
	"dops/internal/vars"
	"dops/internal/vault"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

// Layout constants shared between rendering and mouse translation.
const (
	layoutMarginLeft   = 3
	layoutMarginTop    = 3
	layoutMarginBottom = 4
	layoutBorderSize   = 1 // one side of a rounded border
	layoutPadLeft      = 1 // sidebar content left padding
)

// Layout constants for overlay sizing and flash durations.
const (
	overlayWidthRatio = 2  // overlay width = terminal width * overlayWidthRatio / overlayWidthDenom
	overlayWidthDenom = 3
	overlayMinWidth   = 50
	sidebarMinWidth   = 30
	sidebarMaxWidth   = 50
	copyFlashDuration = 1500 * time.Millisecond
)

// layoutDims holds pre-computed layout dimensions for the main view.
// Computed once per render/resize cycle, shared across all layout-dependent methods.
type layoutDims struct {
	innerW          int
	sw              int // sidebar logical width
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
	sw := sidebarWidth(innerW)
	rightW := clamp(innerW-sw-borderSize-gap, 1)
	contentW := clamp(rightW-borderSize, 1)
	panelRows := clamp(m.height-layoutMarginTop-1-layoutMarginBottom, 1) // -1 for footer

	sidebarContentH := clamp(panelRows-borderSize-1, 3)

	// Render sidebar/metadata to measure actual pixel sizes.
	sidebarView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		PaddingLeft(1).
		Width(sw).
		Height(sidebarContentH).
		Render("")
	// Measure actual rendered height — don't estimate, since lipgloss
	// Height() behavior with borders can vary.
	sidebarRenderedH := lipgloss.Height(sidebarView)
	sidebarRenderedW := lipgloss.Width(sidebarView)

	metaContent := metadata.Render(m.selected, m.selCat, contentW, m.copiedFlash, m.deps.Styles)
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
		sw:               sw,
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

type executionDoneMsg struct {
	LogPath string
	Err     error
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
	Version    string       // current build version for update checks
	DopsDir    string       // ~/.dops directory for cache files
	Vault      *vault.Vault // encrypted parameter storage
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
	updateAvailable  string // non-empty if a newer version exists (e.g. "0.2.0")
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
		return m, nil, true

	case output.OutputLineMsg:
		m.output, _ = m.output.Update(msg)
		return m, nil, true

	case output.ExecutionDoneMsg:
		m.output, _ = m.output.Update(msg)
		return m, nil, true

	case wizard.WizardSubmitMsg:
		m.state = stateNormal
		m.wizard = nil
		result, cmd := m.openConfirm(msg.Runbook, msg.Catalog, msg.Params)
		return result, cmd, true

	case confirm.ConfirmAcceptMsg:
		m.state = stateNormal
		m.conf = nil
		result, cmd := m.startExecution(msg.Runbook, msg.Catalog, msg.Params)
		return result, cmd, true

	case confirm.ConfirmCancelMsg:
		m.state = stateNormal
		m.conf = nil
		return m, nil, true

	case wizard.WizardCancelMsg:
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

func (m App) startExecution(rb domain.Runbook, cat domain.Catalog, params map[string]string) (tea.Model, tea.Cmd) {
	m.output.Clear()
	resolved := m.resolveVars()
	cmdStr := wizard.BuildCommand(rb, params, resolved)
	m.output.SetCommand(cmdStr)
	m.resizeAll()

	if m.deps.Runner == nil {
		return m, nil
	}

	scriptPath := filepath.Join(expandTilde(cat.RunbookRoot()), rb.Name, rb.Script)

	if m.deps.DryRun {
		m.emitDryRun(scriptPath, params)
		return m, nil
	}

	env := buildEnv(params)
	logPath := m.createLogFile(cat.Name, rb.Name)

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelExec = cancel
	m.execRunning = true

	// Stream lines live via p.Send() if we have a program reference,
	// otherwise fall back to a tea.Cmd (tests).
	if prog := m.program(); prog != nil {
		go m.runStreaming(ctx, prog, scriptPath, env, logPath)
		return m, nil
	}

	return m, m.runBlocking(ctx, scriptPath, env, logPath)
}

// emitDryRun writes dry-run output to the output pane without executing.
func (m *App) emitDryRun(scriptPath string, params map[string]string) {
	m.output, _ = m.output.Update(output.OutputLineMsg{Text: "[DRY RUN] Would execute:"})
	m.output, _ = m.output.Update(output.OutputLineMsg{Text: fmt.Sprintf("  Script: %s", scriptPath)})
	m.output, _ = m.output.Update(output.OutputLineMsg{Text: ""})
	if len(params) > 0 {
		m.output, _ = m.output.Update(output.OutputLineMsg{Text: "  Environment:"})
		for k, v := range params {
			m.output, _ = m.output.Update(output.OutputLineMsg{
				Text: fmt.Sprintf("    %s=%s", strings.ToUpper(k), v),
			})
		}
	}
	m.output, _ = m.output.Update(output.ExecutionDoneMsg{})
}

// runStreaming executes a script in a goroutine, streaming lines to the
// tea.Program via Send(). Used for live output in the TUI.
func (m App) runStreaming(ctx context.Context, prog *tea.Program, scriptPath string, env map[string]string, logPath string) {
	lw := m.deps.LogWriter
	lines, errs := m.deps.Runner.Run(ctx, scriptPath, env)
	for line := range lines {
		if lw != nil {
			lw.WriteLine(line.Text)
		}
		prog.Send(output.OutputLineMsg{Text: line.Text, IsStderr: line.IsStderr})
	}
	if lw != nil {
		lw.Close()
	}
	prog.Send(executionDoneMsg{LogPath: logPath, Err: <-errs})
}

// runBlocking returns a tea.Cmd that executes a script synchronously.
// Used in tests where no tea.Program reference is available.
func (m App) runBlocking(ctx context.Context, scriptPath string, env map[string]string, logPath string) tea.Cmd {
	runner := m.deps.Runner
	lw := m.deps.LogWriter
	return func() tea.Msg {
		lines, errs := runner.Run(ctx, scriptPath, env)
		for line := range lines {
			if lw != nil {
				lw.WriteLine(line.Text)
			}
		}
		if lw != nil {
			lw.Close()
		}
		return executionDoneMsg{LogPath: logPath, Err: <-errs}
	}
}

// program returns the tea.Program reference, or nil if not set.
func (m App) program() *tea.Program {
	if m.deps.ProgramRef != nil {
		return m.deps.ProgramRef.P
	}
	return nil
}

// createLogFile creates a log file for the execution and returns its path.
func (m App) createLogFile(catalogName, runbookName string) string {
	if m.deps.LogWriter == nil {
		return ""
	}
	logPath, err := m.deps.LogWriter.Create(catalogName, runbookName, time.Now())
	if err != nil {
		return ""
	}
	return logPath
}

// buildEnv converts parameter keys to uppercase environment variable names.
func buildEnv(params map[string]string) map[string]string {
	env := make(map[string]string, len(params))
	for k, v := range params {
		env[strings.ToUpper(k)] = v
	}
	return env
}

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
	wiz := wizard.New(*m.selected, *m.selCat, resolved)
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
	c := confirm.New(rb, cat, params, m.width*overlayWidthRatio/overlayWidthDenom, m.deps.Styles)
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

func (m App) viewNormal() tea.View {
	l := m.computeLayout()

	// --- Theme colors ---
	var borderColor, activeBorderColor color.Color
	borderColor = lipgloss.NoColor{}
	activeBorderColor = lipgloss.NoColor{}
	if m.deps.Styles != nil {
		borderColor = m.deps.Styles.Border.GetForeground()
		activeBorderColor = m.deps.Styles.BorderActive.GetForeground()
	}

	// --- Sidebar ---
	sidebarBorderColor := borderColor
	if m.focus == focusSidebar {
		sidebarBorderColor = activeBorderColor
	}
	m.sidebar.SetHeight(l.sidebarContentH)
	sidebarView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(sidebarBorderColor).
		PaddingLeft(1).
		Width(l.sw).
		Height(l.sidebarContentH).
		Render(m.sidebar.View())

	// --- Metadata ---
	metaContent := metadata.Render(m.selected, m.selCat, l.contentW, m.copiedFlash, m.deps.Styles)

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

	// --- Compose ---
	rightPanel := lipgloss.JoinVertical(lipgloss.Left, metaView, outputView)
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
			bTop, bBottom, bLeft, bRight)
	}

	return tea.NewView(content)
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

func expandTilde(path string) string {
	return adapters.ExpandHome(path)
}

// clamp returns v if v >= min, otherwise min.
func clamp(v, min int) int {
	if v < min {
		return min
	}
	return v
}

// translateMouseForOutput converts terminal-absolute mouse coordinates to
// output content-relative coordinates (inside the output border).
func (m App) translateMouseForOutput(msg tea.Msg) tea.Msg {
	l := m.computeLayout()
	originX := layoutMarginLeft + l.sidebarRenderedW + l.gap + layoutBorderSize
	originY := layoutMarginTop + l.metaRenderedH + layoutBorderSize

	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		msg.X -= originX
		msg.Y -= originY
		return msg
	case tea.MouseReleaseMsg:
		msg.X -= originX
		msg.Y -= originY
		return msg
	case tea.MouseMotionMsg:
		msg.X -= originX
		msg.Y -= originY
		return msg
	case tea.MouseWheelMsg:
		msg.X -= originX
		msg.Y -= originY
		return msg
	}
	return msg
}

// outputPaneBounds computes the output pane's terminal-absolute bounds
// for selection highlight and text extraction confinement.
func (m App) outputPaneBounds() (top, bottom, left, right int) {
	l := m.computeLayout()
	padX := 1
	logW := max(1, l.outputInnerW-padX*2-1) // -padX*2 for left/right pad, -1 for scrollbar

	left = layoutMarginLeft + l.sidebarRenderedW + l.gap + layoutBorderSize + padX
	right = left + logW
	top = layoutMarginTop + l.metaRenderedH + layoutBorderSize
	bottom = top + l.outputContentH
	return
}

func (m App) extractSelectionFromView() string {
	sel := m.output.Selection()
	if !sel.Active || sel.IsEmpty() {
		return ""
	}

	view := m.viewNormal()
	content := view.Content
	startX, startY, endX, endY := sel.Bounds()

	// Confine extraction to output pane bounds.
	bTop, bBottom, bLeft, bRight := m.outputPaneBounds()
	if startY < bTop {
		startY = bTop
		startX = bLeft
	}
	if endY > bBottom {
		endY = bBottom
		endX = bRight
	}

	lines := strings.Split(content, "\n")
	var result []string
	for i := startY; i <= endY; i++ {
		if i < 0 || i >= len(lines) {
			continue
		}
		lineWidth := ansi.StringWidth(lines[i])
		if lineWidth == 0 {
			continue
		}

		lx := bLeft
		rx := min(lineWidth, bRight)
		if i == startY {
			lx = max(startX, bLeft)
		}
		if i == endY {
			rx = min(bRight, min(lineWidth, endX+1))
		}
		if lx >= rx {
			continue
		}

		selected := ansi.Cut(lines[i], lx, rx)
		plain := ansi.Strip(selected)
		plain = strings.TrimRight(plain, " ")
		if plain != "" {
			result = append(result, plain)
		}
	}

	return strings.TrimRight(strings.Join(result, "\n"), "\n ")
}

// applySelectionHighlight post-processes the full terminal view to highlight
// the selected text, confined within the output pane bounds.
func applySelectionHighlight(content string, sel output.TextSelection, styles *theme.Styles, boundsTop, boundsBottom, boundsLeft, boundsRight int) string {
	hlStyle := lipgloss.NewStyle()
	if styles != nil {
		hlStyle = lipgloss.NewStyle().
			Background(styles.Primary.GetForeground()).
			Foreground(styles.BackgroundElem.GetForeground())
	}

	startX, startY, endX, endY := sel.Bounds()

	// Clamp to output pane bounds.
	if startY < boundsTop {
		startY = boundsTop
		startX = boundsLeft
	}
	if endY > boundsBottom {
		endY = boundsBottom
		endX = boundsRight
	}
	if startY > endY {
		return content
	}

	lines := strings.Split(content, "\n")
	for i := range lines {
		if i < startY || i > endY {
			continue
		}

		lineWidth := ansi.StringWidth(lines[i])
		if lineWidth == 0 {
			continue
		}

		var lx, rx int
		if i == startY && i == endY {
			lx = max(startX, boundsLeft)
			rx = min(boundsRight, min(lineWidth, endX+1))
		} else if i == startY {
			lx = max(startX, boundsLeft)
			rx = min(lineWidth, boundsRight)
		} else if i == endY {
			lx = boundsLeft
			rx = min(boundsRight, min(lineWidth, endX+1))
		} else {
			lx = boundsLeft
			rx = min(lineWidth, boundsRight)
		}

		if lx >= rx || lx >= lineWidth {
			continue
		}

		before := ansi.Cut(lines[i], 0, lx)
		selected := ansi.Cut(lines[i], lx, rx)
		after := ansi.Cut(lines[i], rx, lineWidth)

		plain := ansi.Strip(selected)
		lines[i] = before + "\x1b[0m" + hlStyle.Render(plain) + after
	}

	return strings.Join(lines, "\n")
}

// injectBorderBadge replaces part of the top border row with a styled badge.
// The badge appears near the right end: ╭─── Copied to Clipboard! ──╮
func injectBorderBadge(rendered, text string, styles *theme.Styles) string {
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		return rendered
	}

	topLine := lines[0]
	topWidth := lipgloss.Width(topLine)

	// Style the badge text.
	badgeStyle := lipgloss.NewStyle()
	if styles != nil {
		badgeStyle = styles.Success
	}
	badge := " " + badgeStyle.Render(text) + " "
	badgeW := lipgloss.Width(badge)

	// Place badge near the right end (2 chars from right for the corner + margin).
	rightMargin := 3
	insertAt := topWidth - badgeW - rightMargin
	if insertAt < 2 {
		return rendered // not enough space
	}

	// Rebuild the top line: keep left portion, insert badge, keep right portion.
	left := ansi.Cut(topLine, 0, insertAt)
	right := ansi.Cut(topLine, insertAt+badgeW, topWidth)
	lines[0] = left + badge + right

	return strings.Join(lines, "\n")
}


func sidebarWidth(totalWidth int) int {
	w := totalWidth / 3
	if w < sidebarMinWidth {
		w = sidebarMinWidth
	}
	if w > sidebarMaxWidth {
		w = sidebarMaxWidth
	}
	return w
}

// sidebarBounds returns the origin and whether coordinates are within the sidebar.
func (m App) sidebarBounds(mx, my int) (originX, originY int, inBounds bool) {
	originY = layoutMarginTop + layoutBorderSize
	originX = layoutMarginLeft + layoutBorderSize + layoutPadLeft
	innerW := clamp(m.width-layoutMarginLeft, 1)
	sw := sidebarWidth(innerW) + layoutBorderSize*2 + layoutPadLeft
	inBounds = mx >= layoutMarginLeft && mx < layoutMarginLeft+sw &&
		my >= layoutMarginTop && my < m.height
	return
}

// translateMouseForSidebar converts terminal-absolute mouse coordinates to
// sidebar content-relative coordinates. Returns translated msg and inBounds.
func (m App) translateMouseForSidebar(msg tea.Msg) (tea.Msg, bool) {
	mx, my, ok := mouseCoords(msg)
	if !ok {
		return msg, false
	}
	originX, originY, inBounds := m.sidebarBounds(mx, my)

	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		msg.X -= originX
		msg.Y -= originY
		return msg, inBounds
	case tea.MouseMotionMsg:
		msg.X -= originX
		msg.Y -= originY
		return msg, inBounds
	case tea.MouseReleaseMsg:
		msg.X -= originX
		msg.Y -= originY
		return msg, inBounds
	case tea.MouseWheelMsg:
		return msg, inBounds
	}
	return msg, false
}

// mouseCoords extracts X, Y from any mouse message type.
// Returns (0, 0, false) if msg is not a mouse event.
func mouseCoords(msg tea.Msg) (x, y int, ok bool) {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		return msg.X, msg.Y, true
	case tea.MouseReleaseMsg:
		return msg.X, msg.Y, true
	case tea.MouseMotionMsg:
		return msg.X, msg.Y, true
	case tea.MouseWheelMsg:
		return msg.X, msg.Y, true
	}
	return 0, 0, false
}

// focusTargetFromMouse returns which pane a mouse event is over.
func (m App) focusTargetFromMouse(msg tea.Msg) (focusTarget, bool) {
	mx, my, ok := mouseCoords(msg)
	if !ok {
		return 0, false
	}

	innerW := clamp(m.width-layoutMarginLeft, 1)
	sw := sidebarWidth(innerW) + layoutBorderSize*2 + layoutPadLeft
	sidebarRight := layoutMarginLeft + sw

	if mx >= layoutMarginLeft && mx < sidebarRight && my >= layoutMarginTop {
		// Sidebar: only steal focus on click, not hover.
		if _, isClick := msg.(tea.MouseClickMsg); isClick {
			return focusSidebar, true
		}
		return m.focus, false
	}
	if mx >= sidebarRight && my >= layoutMarginTop {
		return focusOutput, true
	}
	return m.focus, false
}

func isMouseMsg(msg tea.Msg) bool {
	switch msg.(type) {
	case tea.MouseClickMsg, tea.MouseMotionMsg, tea.MouseReleaseMsg, tea.MouseWheelMsg:
		return true
	}
	return false
}

func isMouseClick(msg tea.Msg) bool {
	_, ok := msg.(tea.MouseClickMsg)
	return ok
}

// handleMetadataClick checks if a mouse click landed on the path/URL line
// in the metadata panel. If so, returns the location string and a cmd to copy it.
func (m App) handleMetadataClick(msg tea.Msg) (string, tea.Cmd) {
	click, ok := msg.(tea.MouseClickMsg)
	if !ok {
		return "", nil
	}

	location := metadata.Location(m.selected, m.selCat)
	if location == "" {
		return "", nil
	}

	l := m.computeLayout()

	// Path is the last content line before the bottom border.
	pathLineY := layoutMarginTop + l.metaRenderedH - layoutBorderSize - 1
	if click.Y != pathLineY {
		return "", nil
	}

	pathXStart := layoutMarginLeft + l.sidebarRenderedW + l.gap + layoutBorderSize + 1
	pathXEnd := pathXStart + len(location)
	if click.X < pathXStart || click.X >= pathXEnd {
		return "", nil
	}

	return location, tea.Batch(
		tea.SetClipboard(location),
		tea.Tick(copyFlashDuration, func(time.Time) tea.Msg {
			return copiedFlashMsg{}
		}),
	)
}

// handleOutputClick checks if a mouse click landed on the output header (command)
// or footer (log path) text. If so, copies the value to clipboard.
func (m *App) handleOutputClick(msg tea.Msg) tea.Cmd {
	click, ok := msg.(tea.MouseClickMsg)
	if !ok {
		return nil
	}

	lines := strings.Split(ansi.Strip(m.viewNormal().Content), "\n")
	if click.Y < 0 || click.Y >= len(lines) {
		return nil
	}
	line := lines[click.Y]

	var copyText, region string

	// Check footer FIRST — specific text match on "Saved to <path>".
	if logPath := m.output.LogPath(); logPath != "" {
		if strings.Contains(line, "Saved to") || strings.Contains(line, logPath) {
			copyText, region = logPath, "footer"
		}
	}

	// Check header — any line containing "$ " or "--param" in the output area.
	if region == "" {
		if cmd := m.output.Command(); cmd != "" {
			if strings.Contains(line, "$ ") || strings.Contains(line, "--param") {
				copyText, region = cmd, "header"
			}
		}
	}

	if region == "" {
		return nil
	}

	// Guard: reject if a copy is already in progress.
	if !m.output.TryCopy() {
		return nil
	}

	// Green flash on the clicked region.
	switch region {
	case "header":
		m.output.SetCopiedHeader(true)
	case "footer":
		m.output.SetCopiedFooter(true)
	}
	return tea.Batch(
		tea.SetClipboard(copyText),
		// Short highlight flash (500ms).
		tea.Tick(copyFlashDuration, func(time.Time) tea.Msg {
			return output.CopiedRegionFlashMsg{}
		}),
		// Badge stays longer (1.5s).
		tea.Tick(copyFlashDuration, func(time.Time) tea.Msg {
			return output.CopyFlashExpiredMsg{}
		}),
	)
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
