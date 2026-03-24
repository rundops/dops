package tui

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"dops/internal/adapters"
	"dops/internal/catalog"
	"dops/internal/config"
	"dops/internal/domain"
	"dops/internal/executor"
	"dops/internal/theme"
	"dops/internal/tui/footer"
	"dops/internal/tui/metadata"
	"dops/internal/tui/output"
	"dops/internal/tui/palette"
	"dops/internal/tui/sidebar"
	"dops/internal/tui/wizard"
	"dops/internal/vars"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type viewState int

const (
	stateNormal viewState = iota
	stateWizard
	statePalette
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

var tuiANSIPattern = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

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
	ProgramRef *ProgramRef
}

type copiedFlashMsg struct{}

type App struct {
	sidebar     sidebar.Model
	output      output.Model
	wizard      *wizard.Model
	pal         *palette.Model
	selected    *domain.Runbook
	selCat      *domain.Catalog
	deps        AppDeps
	state       viewState
	width       int
	height      int
	copiedFlash bool // show "Copied to Clipboard" in metadata
	focus       focusTarget
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


func (m App) Init() tea.Cmd {
	return m.sidebar.Init()
}

// resizeAll computes layout dimensions from the current terminal size and
// persists them on the sidebar and output models. Must be called from Update
// (not View) so changes survive across message cycles.
func (m *App) resizeAll() {
	footerH := 1
	gap := 1
	borderSize := layoutBorderSize * 2

	innerW := clamp(m.width-layoutMarginLeft, 1)
	sw := sidebarWidth(innerW)
	rightW := clamp(innerW-sw-borderSize-gap, 1)
	contentW := clamp(rightW-borderSize, 1)
	panelRows := clamp(m.height-layoutMarginTop-footerH-layoutMarginBottom, 1)

	// Sidebar
	sidebarContentH := clamp(panelRows-borderSize-1, 3)
	m.sidebar.SetHeight(sidebarContentH)

	// Metadata height estimate (render to measure).
	metaContent := metadata.Render(m.selected, m.selCat, contentW, m.copiedFlash, m.deps.Styles)
	metaView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(contentW).
		Render(metaContent)
	metaRenderedH := lipgloss.Height(metaView)
	sidebarRenderedH := sidebarContentH + borderSize

	// Output — pass content dimensions (inside the outer border the app renders).
	outputTotalH := clamp(sidebarRenderedH-metaRenderedH, 3)
	outputContentH := clamp(outputTotalH-borderSize, 1)  // subtract outer border top+bottom
	outputInnerW := clamp(contentW-borderSize, 1)         // subtract outer border left+right
	m.output.SetSize(outputInnerW, outputContentH)
}

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeAll()
		return m, nil

	case tea.KeyPressMsg:
		if m.state == stateNormal {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "ctrl+shift+p":
				return m.openPalette()
			case "tab", "shift+tab":
				if m.focus == focusSidebar {
					m.focus = focusOutput
				} else {
					m.focus = focusSidebar
				}
				return m, nil
			}
		}

	case sidebar.RunbookSelectedMsg:
		rb := msg.Runbook
		cat := msg.Catalog
		m.selected = &rb
		m.selCat = &cat
		// Don't clear output — keep last execution visible until a new one starts.
		return m, nil

	case sidebar.RunbookExecuteMsg:
		rb := msg.Runbook
		cat := msg.Catalog
		m.selected = &rb
		m.selCat = &cat
		return m.openWizard()

	case executionDoneMsg:
		m.output, _ = m.output.Update(output.ExecutionDoneMsg{LogPath: msg.LogPath, Err: msg.Err})
		return m, nil

	case output.OutputLineMsg:
		m.output, _ = m.output.Update(msg)
		return m, nil

	case output.ExecutionDoneMsg:
		m.output, _ = m.output.Update(msg)
		return m, nil

	case wizard.WizardSubmitMsg:
		m.state = stateNormal
		m.wizard = nil
		return m.startExecution(msg.Runbook, msg.Catalog, msg.Params)

	case wizard.WizardCancelMsg:
		m.state = stateNormal
		m.wizard = nil
		return m, nil

	case palette.PaletteSelectMsg:
		m.state = stateNormal
		m.pal = nil
		return m, nil

	case palette.PaletteCancelMsg:
		m.state = stateNormal
		m.pal = nil
		return m, nil

	case copiedFlashMsg:
		m.copiedFlash = false
		return m, nil

	case output.CopiedHeaderFlashMsg:
		m.output.SetCopiedHeader(false)
		return m, nil

	case output.CopiedFooterFlashMsg:
		m.output.SetCopiedFooter(false)
		return m, nil
	}

	// Switch focus on hover: any mouse event over a pane focuses it.
	if m.state == stateNormal {
		if target, ok := m.focusTargetFromMouse(msg); ok {
			m.focus = target
		}
	}

	// Route to focused component
	switch m.state {
	case stateNormal:
		// Check for click-to-copy targets
		if isMouseClick(msg) {
			if _, cmd := m.handleMetadataClick(msg); cmd != nil {
				m.copiedFlash = true
				return m, cmd
			}
			if cmd := m.handleOutputClick(msg); cmd != nil {
				return m, cmd
			}
		}

		// Route based on focus target.
		if m.focus == focusOutput {
			// Mouse events still go to sidebar if in sidebar area.
			if isMouseMsg(msg) {
				translated, inSidebar := m.translateMouseForSidebar(msg)
				if inSidebar {
					var cmd tea.Cmd
					m.sidebar, cmd = m.sidebar.Update(translated)
					return m, cmd
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.output, cmd = m.output.Update(msg)
			return m, cmd
		}

		translated, inSidebar := m.translateMouseForSidebar(msg)
		if !inSidebar {
			// Mouse event outside sidebar — don't forward
			if isMouseMsg(msg) {
				return m, nil
			}
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
	}

	return m, nil
}

func (m App) startExecution(rb domain.Runbook, cat domain.Catalog, params map[string]string) (tea.Model, tea.Cmd) {
	m.output.Clear()
	cmdStr := wizard.BuildCommand(rb, params)
	m.output.SetCommand(cmdStr)
	m.resizeAll() // Recompute dimensions after command changes header height

	if m.deps.Store != nil && m.deps.Config != nil {
		for _, p := range rb.Parameters {
			val, ok := params[p.Name]
			if !ok {
				continue
			}
			var keyPath string
			switch p.Scope {
			case "global":
				keyPath = fmt.Sprintf("vars.global.%s", p.Name)
			case "catalog":
				keyPath = fmt.Sprintf("vars.catalog.%s.%s", cat.Name, p.Name)
			case "runbook":
				keyPath = fmt.Sprintf("vars.catalog.%s.runbooks.%s.%s", cat.Name, rb.Name, p.Name)
			default:
				keyPath = fmt.Sprintf("vars.global.%s", p.Name)
			}
			config.Set(m.deps.Config, keyPath, val)
		}
		m.deps.Store.Save(m.deps.Config)
	}

	if m.deps.Runner == nil {
		return m, nil
	}

	catPath := expandTilde(cat.Path)
	scriptPath := filepath.Join(catPath, rb.Name, rb.Script)

	var logPath string
	if m.deps.LogWriter != nil {
		lp, err := m.deps.LogWriter.Create(cat.Name, rb.Name, time.Now())
		if err == nil {
			logPath = lp
		}
	}

	env := make(map[string]string)
	for k, v := range params {
		env[strings.ToUpper(k)] = v
	}

	var prog *tea.Program
	if m.deps.ProgramRef != nil {
		prog = m.deps.ProgramRef.P
	}
	runner := m.deps.Runner
	lw := m.deps.LogWriter
	finalLogPath := logPath

	// If we have a program reference, stream lines live via p.Send().
	// Otherwise fall back to returning a single done message (e.g. in tests).
	if prog != nil {
		go func() {
			lines, errs := runner.Run(context.Background(), scriptPath, env)
			for line := range lines {
				if lw != nil {
					lw.WriteLine(line.Text)
				}
				prog.Send(output.OutputLineMsg{
					Text:     line.Text,
					IsStderr: line.IsStderr,
				})
			}
			if lw != nil {
				lw.Close()
			}
			err := <-errs
			prog.Send(executionDoneMsg{LogPath: finalLogPath, Err: err})
		}()
		return m, nil
	}

	// Fallback: no program reference (tests). Collect and return as done msg.
	return m, func() tea.Msg {
		lines, errs := runner.Run(context.Background(), scriptPath, env)
		for line := range lines {
			if lw != nil {
				lw.WriteLine(line.Text)
			}
		}
		if lw != nil {
			lw.Close()
		}
		err := <-errs
		return executionDoneMsg{LogPath: finalLogPath, Err: err}
	}
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

	if wizard.ShouldSkip(m.selected.Parameters, resolved) {
		return m.startExecution(*m.selected, *m.selCat, resolved)
	}

	wiz := wizard.New(*m.selected, *m.selCat, resolved)
	m.wizard = &wiz
	m.state = stateWizard
	return m, wiz.Init()
}

func (m App) resolveVars() map[string]string {
	if m.deps.Config == nil || m.selected == nil || m.selCat == nil {
		return make(map[string]string)
	}
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

	if m.state == stateWizard && m.wizard != nil {
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
	// --- Layout variables (derived from package-level constants) ---
	footerH    := 1 // footer height
	gap        := 1 // space between sidebar and right panel
	borderSize := layoutBorderSize * 2 // top + bottom (or left + right)

	// --- Dimension budget ---
	innerW    := clamp(m.width - layoutMarginLeft, 1)
	sw        := sidebarWidth(innerW)
	rightW    := clamp(innerW - sw - borderSize - gap, 1)
	contentW  := clamp(rightW - borderSize, 1) // content width inside bordered panels
	panelRows := clamp(m.height - layoutMarginTop - footerH - layoutMarginBottom, 1)

	// --- Theme colors ---
	var borderColor, activeBorderColor color.Color
	borderColor = lipgloss.NoColor{}
	activeBorderColor = lipgloss.NoColor{}
	if m.deps.Styles != nil {
		borderColor = m.deps.Styles.Border.GetForeground()
		activeBorderColor = m.deps.Styles.BorderActive.GetForeground()
	}

	// Focus-aware border colors.
	sidebarBorderColor := borderColor
	if m.focus == focusSidebar {
		sidebarBorderColor = activeBorderColor
	}

	// --- Sidebar: anchor panel ---
	sidebarContentH := clamp(panelRows - borderSize - 1, 3) // -1 accounts for border rendering offset
	m.sidebar.SetHeight(sidebarContentH)

	sidebarView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(sidebarBorderColor).
		PaddingLeft(1).
		Width(sw).
		Height(sidebarContentH).
		Render(m.sidebar.View())

	sidebarRenderedH := lipgloss.Height(sidebarView)

	// --- Metadata: bordered, auto-height ---
	metaContent := metadata.Render(m.selected, m.selCat, contentW, m.copiedFlash, m.deps.Styles)
	metaView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(contentW).
		Render(metaContent)

	metaRenderedH := lipgloss.Height(metaView)

	// --- Output: persistent outer border with content inside ---
	outputBorderColor := borderColor
	if m.focus == focusOutput {
		outputBorderColor = activeBorderColor
	}
	outputTotalH := clamp(sidebarRenderedH-metaRenderedH, 3)
	outputContentH := clamp(outputTotalH-borderSize, 1)
	outputInnerW := clamp(contentW-borderSize, 1) // content width inside the outer border
	m.output.SetFocused(m.focus == focusOutput)
	outputView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(outputBorderColor).
		Width(contentW).
		Height(outputTotalH).
		Render(m.output.ViewWithSize(outputInnerW, outputContentH))

	// --- Compose panels ---
	rightPanel := lipgloss.JoinVertical(lipgloss.Left, metaView, outputView)

	body := lipgloss.NewStyle().
		MarginLeft(layoutMarginLeft).
		MarginTop(layoutMarginTop).
		Render(lipgloss.JoinHorizontal(lipgloss.Top,
			sidebarView,
			strings.Repeat(" ", gap),
			rightPanel,
		))

	// --- Footer ---
	footerView := lipgloss.NewStyle().
		MarginLeft(layoutMarginLeft - 1).
		Render(footer.Render(appFooterState(m.state), m.width-layoutMarginLeft, m.deps.Styles))

	// --- Outer container: enforce exact terminal dimensions ---
	content := lipgloss.JoinVertical(lipgloss.Left, body, footerView)
	content = lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(content)

	return tea.NewView(content)
}

func (m App) viewWizardOverlay() tea.View {
	wizView := m.wizard.View()

	overlayW := m.width * 2 / 3
	if overlayW < 50 {
		overlayW = 50
	}

	overlay := lipgloss.NewStyle().
		Width(overlayW).
		Render(wizView)

	content := lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		overlay,
	)

	footerView := footer.Render(footer.StateWizard, m.width, m.deps.Styles)
	content = lipgloss.JoinVertical(lipgloss.Left, content, footerView)

	return tea.NewView(content)
}

func (m App) viewPaletteOverlay() tea.View {
	palView := m.pal.View()

	overlay := lipgloss.NewStyle().
		Width(m.width).
		Render(palView)

	footerView := footer.Render(footer.StatePalette, m.width, m.deps.Styles)
	content := lipgloss.JoinVertical(lipgloss.Left, overlay, footerView)

	return tea.NewView(content)
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// clamp returns v if v >= min, otherwise min.
func clamp(v, min int) int {
	if v < min {
		return min
	}
	return v
}

func sidebarWidth(totalWidth int) int {
	w := totalWidth / 3
	if w < 30 {
		w = 30
	}
	if w > 50 {
		w = 50
	}
	return w
}

// translateMouseForSidebar converts terminal-absolute mouse coordinates to
// sidebar content-relative coordinates (Y=0 is the first item row).
// Returns the translated message and whether the click was within the sidebar bounds.
func (m App) translateMouseForSidebar(msg tea.Msg) (tea.Msg, bool) {
	originY := layoutMarginTop + layoutBorderSize
	originX := layoutMarginLeft + layoutBorderSize + layoutPadLeft
	innerW := clamp(m.width-layoutMarginLeft, 1)
	sw := sidebarWidth(innerW) + layoutBorderSize*2 + layoutPadLeft // total sidebar width including border+pad

	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		inBounds := msg.X >= layoutMarginLeft && msg.X < layoutMarginLeft+sw &&
			msg.Y >= layoutMarginTop && msg.Y < m.height
		msg.X -= originX
		msg.Y -= originY
		return msg, inBounds
	case tea.MouseMotionMsg:
		inBounds := msg.X >= layoutMarginLeft && msg.X < layoutMarginLeft+sw &&
			msg.Y >= layoutMarginTop && msg.Y < m.height
		msg.X -= originX
		msg.Y -= originY
		return msg, inBounds
	}
	return msg, true
}

// focusTargetFromMouse returns which pane a mouse event is over.
// Returns the target and true if the event is a mouse event, false otherwise.
func (m App) focusTargetFromMouse(msg tea.Msg) (focusTarget, bool) {
	var mx, my int
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		mx, my = msg.X, msg.Y
	case tea.MouseMotionMsg:
		mx, my = msg.X, msg.Y
	case tea.MouseWheelMsg:
		mx, my = msg.X, msg.Y
	default:
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
	case tea.MouseClickMsg, tea.MouseMotionMsg:
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

	// Replicate viewNormal layout to get exact pixel positions
	innerW := clamp(m.width-layoutMarginLeft, 1)
	sw := sidebarWidth(innerW)
	gap := 1
	borderSize := layoutBorderSize * 2
	rightW := clamp(innerW-sw-borderSize-gap, 1)
	contentW := clamp(rightW-borderSize, 1)

	// Render sidebar to measure its actual width
	sidebarView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		PaddingLeft(layoutPadLeft).
		Width(sw).
		Render("")
	sidebarRenderedW := lipgloss.Width(sidebarView)

	// Render metadata to measure its actual height
	metaContent := metadata.Render(m.selected, m.selCat, contentW, false, m.deps.Styles)
	metaView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(contentW).
		Render(metaContent)
	metaH := lipgloss.Height(metaView)

	metaYStart := layoutMarginTop
	// Path is the last content line before bottom border
	pathLineY := metaYStart + metaH - layoutBorderSize - 1
	if click.Y != pathLineY {
		return "", nil
	}

	// Path text X: marginLeft + sidebar width + gap + metadata border left + leading space
	pathXStart := layoutMarginLeft + sidebarRenderedW + gap + layoutBorderSize + 1
	pathXEnd := pathXStart + len(location)
	if click.X < pathXStart || click.X >= pathXEnd {
		return "", nil
	}

	return location, tea.Batch(
		tea.SetClipboard(location),
		tea.Tick(2*time.Second, func(time.Time) tea.Msg {
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

	lines := strings.Split(tuiANSIPattern.ReplaceAllString(m.viewNormal().Content, ""), "\n")
	if click.Y < 0 || click.Y >= len(lines) {
		return nil
	}
	line := lines[click.Y]

	var copyText, region string
	if cmd := m.output.Command(); cmd != "" {
		target := "$ " + cmd
		if idx := strings.Index(line, target); idx >= 0 {
			start := lipgloss.Width(line[:idx])
			end := start + lipgloss.Width(target)
			if click.X >= start && click.X < end {
				copyText, region = cmd, "header"
			}
		}
	}
	if region == "" {
		if logPath := m.output.LogPath(); logPath != "" {
			target := "Saved to " + logPath
			if idx := strings.Index(line, target); idx >= 0 {
				start := lipgloss.Width(line[:idx])
				end := start + lipgloss.Width(target)
				if click.X >= start && click.X < end {
					copyText, region = logPath, "footer"
				}
			}
		}
	}
	if region == "" {
		return nil
	}

	switch region {
	case "header":
		m.output.SetCopiedHeader(true)
		return tea.Batch(
			tea.SetClipboard(copyText),
			tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return output.CopiedHeaderFlashMsg{}
			}),
		)
	case "footer":
		m.output.SetCopiedFooter(true)
		return tea.Batch(
			tea.SetClipboard(copyText),
			tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return output.CopiedFooterFlashMsg{}
			}),
		)
	}

	return nil
}

func appFooterState(s viewState) footer.State {
	switch s {
	case stateWizard:
		return footer.StateWizard
	case statePalette:
		return footer.StatePalette
	default:
		return footer.StateNormal
	}
}
