package output

import (
	"fmt"
	"image/color"
	"strings"

	"dops/internal/theme"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// horizontalScrollStep is how many columns h/l scroll per press.
const horizontalScrollStep = 8

type Model struct {
	command      string
	lines        []OutputLineMsg
	logPath      string
	width        int // total outer width (including section borders)
	height       int // total outer height (including section borders)
	vp           viewport.Model
	xOffset      int
	maxLineWidth int
	atBottom     bool
	searching    bool
	navigating   bool
	searchQuery  string
	matchLines   []int
	matchCount   int
	matchIndex   int
	copiedHeader bool
	copiedFooter bool
	copyFlash    bool // show "Copied to Clipboard!" badge on output pane
	copyLock     bool // prevents concurrent copy operations
	commandLineH int  // wrapped command line height in rows
	focused      bool
	selection    TextSelection
	styles       *theme.Styles
}

func New(width, height int, styles *theme.Styles) Model {
	m := Model{
		width:        width,
		height:       height,
		styles:       styles,
		atBottom:     true,
		commandLineH: 1,
	}
	m.vp = viewport.New()
	m.resizeViewport()
	return m
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.updateCommandLineH()
	m.resizeViewport()
}

func (m *Model) SetFocused(v bool) {
	m.focused = v
}

func (m *Model) Clear() {
	m.command = ""
	m.lines = nil
	m.logPath = ""
	m.copiedHeader = false
	m.copiedFooter = false
	m.xOffset = 0
	m.maxLineWidth = 0
	m.atBottom = true
	m.commandLineH = 1
	m.vp.SetContent("")
	m.clearSearch()
}

func (m *Model) SetCommand(cmd string) {
	m.command = cmd
	m.updateCommandLineH()
	m.resizeViewport()
}

func (m Model) Command() string        { return m.command }
func (m Model) LogPath() string        { return m.logPath }
func (m Model) CopyFlash() bool         { return m.copyFlash }
func (m *Model) SetCopyFlash(v bool)    { m.copyFlash = v; if !v { m.copyLock = false } }

// TryCopy attempts to start a copy operation and show the output badge.
// Returns false if a copy is already in progress (lock still held).
func (m *Model) TryCopy() bool {
	if m.copyLock {
		return false
	}
	m.copyLock = true
	m.copyFlash = true // show badge on output pane
	return true
}

// TryLock acquires the copy lock without showing the output badge.
// Use for copies that display their badge elsewhere (e.g. metadata pane).
func (m *Model) TryLock() bool {
	if m.copyLock {
		return false
	}
	m.copyLock = true
	return true
}
func (m Model) Selection() TextSelection { return m.selection }

// Test query methods — expose internal state for assertions without
// coupling tests to struct field names.
func (m Model) Lines() []OutputLineMsg { return m.lines }
func (m Model) AtBottom() bool        { return m.atBottom }
func (m Model) XOffset() int          { return m.xOffset }
func (m Model) MaxLineWidth() int     { return m.maxLineWidth }
func (m Model) BodyHeight() int       { return m.bodyHeight() }
func (m Model) Searching() bool       { return m.searching }
func (m Model) Navigating() bool      { return m.navigating }
func (m Model) MatchCount() int       { return m.matchCount }
func (m Model) MatchIndex() int       { return m.matchIndex }
func (m Model) ScrollbarCharAt(row int) string { return m.scrollbarCharAt(row) }
func (m *Model) SetCopiedHeader(v bool) { m.copiedHeader = v }
func (m *Model) SetCopiedFooter(v bool) { m.copiedFooter = v }

func (m Model) HasSession() bool { return m.command != "" }

// HandleClick checks if a click hits the header or footer text.
func (m Model) HandleClick(x, y, width, height int) (copyText string, region string) {
	m.width = width
	m.height = height
	rendered := m.View()
	if ct, r := hitTestRenderedText(rendered, x, y, "$ "+m.command, m.command, "header"); r != "" {
		return ct, r
	}
	if m.logPath != "" && !m.searching && !m.navigating {
		return hitTestRenderedText(rendered, x, y, "Saved to "+m.logPath, m.logPath, "footer")
	}
	return "", ""
}

func hitTestRenderedText(rendered string, x, y int, target, copyText, region string) (string, string) {
	if target == "" {
		return "", ""
	}
	lines := strings.Split(rendered, "\n")
	if y < 0 || y >= len(lines) {
		return "", ""
	}
	line := ansi.Strip(lines[y])
	idx := strings.Index(line, target)
	if idx == -1 {
		return "", ""
	}
	start := lipgloss.Width(line[:idx])
	end := start + lipgloss.Width(target)
	if x < start || x >= end {
		return "", ""
	}
	return copyText, region
}

// ---------------------------------------------------------------------------
// Height / width calculations (matching legacy chromeHeight/contentHeight)
// ---------------------------------------------------------------------------

// bodyHeight returns the number of visible log lines.
// Conservative estimate using minimum header height (1 row).
func (m Model) bodyHeight() int {
	return max(1, m.height-6) // header(1) + footer(1) + 2 gaps + top pad + bottom pad
}

// textWidth returns usable character width for log lines.
func (m Model) textWidth() int {
	// 2 padding + 2 indent + 1 scrollbar
	return max(1, m.width-5)
}

func (m *Model) updateCommandLineH() {
	// Header is always 1 row (truncated to fit).
	m.commandLineH = 1
}

func (m *Model) resizeViewport() {
	m.vp.SetWidth(max(1, m.textWidth()))
	m.vp.SetHeight(max(1, m.bodyHeight()))
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case OutputLineMsg:
		msg.Text = strings.ReplaceAll(msg.Text, "\r", "")
		m.lines = append(m.lines, msg)
		if lw := ansi.StringWidth(msg.Text); lw > m.maxLineWidth {
			m.maxLineWidth = lw
		}
		m.syncViewportContent()
		if m.atBottom {
			m.vp.GotoBottom()
		}
		return m, nil

	case ExecutionDoneMsg:
		m.logPath = msg.LogPath
		return m, nil

	case tea.KeyPressMsg:
		if m.navigating {
			return m.updateNavigating(msg), nil
		}
		if m.searching {
			return m.updateSearching(msg), nil
		}
		return m.updateNormal(msg)

	case tea.MouseClickMsg:
		// Store selection in rendered-output coordinates (no offset math).
		// highlightSelection operates directly on these coordinates.
		m.selection.Reset()
		m.selection.Active = true
		m.selection.AnchorX = msg.X
		m.selection.AnchorY = msg.Y
		m.selection.FocusX = msg.X
		m.selection.FocusY = msg.Y
		return m, nil

	case tea.MouseMotionMsg:
		if m.selection.Active {
			m.selection.FocusX = msg.X
			m.selection.FocusY = msg.Y
		}
		return m, nil

	case tea.MouseReleaseMsg:
		if m.selection.Active {
			m.selection.FocusX = msg.X
			m.selection.FocusY = msg.Y

			if !m.selection.IsEmpty() {
				// Signal the app to extract text and copy to clipboard.
				// The app has access to the full rendered view for extraction.
				return m, func() tea.Msg { return SelectionCompleteMsg{} }
			}
		}
		return m, nil

	case CopyFlashExpiredMsg:
		m.copyFlash = false
		m.copyLock = false
		m.selection.Reset()
		return m, nil
	}

	return m.forwardToViewport(msg)
}

func (m Model) updateNormal(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch {
	case msg.Text == "/" || msg.String() == "/":
		m.searching = true
		m.searchQuery = ""
		return m, nil
	case msg.Text == "h" || msg.Code == tea.KeyLeft:
		m.xOffset = max(0, m.xOffset-horizontalScrollStep)
		return m, nil
	case msg.Text == "l" || msg.Code == tea.KeyRight:
		tw := m.textWidth()
		maxScroll := max(0, m.maxLineWidth-tw)
		m.xOffset = min(m.xOffset+horizontalScrollStep, maxScroll)
		return m, nil
	case msg.Text == "y":
		if m.selection.Active && !m.selection.IsEmpty() {
			return m, func() tea.Msg { return SelectionCompleteMsg{} }
		}
		return m, nil
	}
	// Clear selection on scroll keys.
	m.selection.Reset()
	return m.forwardToViewport(msg)
}

func (m Model) forwardToViewport(msg tea.Msg) (Model, tea.Cmd) {
	prevOffset := m.vp.YOffset()
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	newOffset := m.vp.YOffset()
	if newOffset < prevOffset {
		m.atBottom = false
	} else if newOffset > prevOffset {
		m.atBottom = m.vp.AtBottom()
	}
	return m, cmd
}

func (m Model) updateSearching(msg tea.KeyPressMsg) Model {
	switch {
	case msg.Code == tea.KeyEscape:
		m.clearSearch()
	case msg.Code == tea.KeyEnter:
		if m.matchCount > 0 {
			m.navigating = true
			m.searching = false
			m.matchIndex = 0
			m.scrollToMatch()
		}
	case msg.Code == tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.applySearch()
		}
	default:
		if msg.Text != "" {
			m.searchQuery += msg.Text
			m.applySearch()
		}
	}
	return m
}

func (m Model) updateNavigating(msg tea.KeyPressMsg) Model {
	switch {
	case msg.Code == tea.KeyEscape:
		m.clearSearch()
	case msg.Text == "n":
		if m.matchCount > 0 {
			m.matchIndex = (m.matchIndex + 1) % m.matchCount
			m.scrollToMatch()
		}
	case msg.Text == "N":
		if m.matchCount > 0 {
			m.matchIndex = (m.matchIndex - 1 + m.matchCount) % m.matchCount
			m.scrollToMatch()
		}
	}
	return m
}

func (m *Model) applySearch() {
	m.matchLines = nil
	m.matchCount = 0
	m.matchIndex = 0
	if m.searchQuery == "" {
		return
	}
	q := strings.ToLower(m.searchQuery)
	for i, line := range m.lines {
		if strings.Contains(strings.ToLower(line.Text), q) {
			m.matchLines = append(m.matchLines, i)
			m.matchCount++
		}
	}
}

func (m *Model) clearSearch() {
	m.searching = false
	m.navigating = false
	m.searchQuery = ""
	m.matchLines = nil
	m.matchCount = 0
	m.matchIndex = 0
}

// visibleLineTexts returns the plain text of currently visible log lines.
func (m Model) visibleLineTexts() []string {
	bodyH := m.bodyHeight()
	yOffset := m.vp.YOffset()
	result := make([]string, bodyH)
	for i := range bodyH {
		idx := yOffset + i
		if idx < len(m.lines) {
			result[i] = m.lines[idx].Text
		}
	}
	return result
}

func (m *Model) scrollToMatch() {
	if m.matchIndex < 0 || m.matchIndex >= len(m.matchLines) {
		return
	}
	m.vp.SetYOffset(m.matchLines[m.matchIndex])
	m.atBottom = false
}

func (m *Model) syncViewportContent() {
	var sb strings.Builder
	for i := range m.lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteByte(' ')
	}
	m.vp.SetContent(sb.String())
}

// ---------------------------------------------------------------------------
// View — three stacked bordered sections
// ---------------------------------------------------------------------------

func (m Model) ViewWithSize(width, height int) string {
	m.width = width
	m.height = height
	return m.View()
}

// viewColors holds resolved theme colors for rendering.
type viewColors struct {
	text, muted, stderr, success, primary, bgElem color.Color
}

func (m Model) resolveColors() viewColors {
	c := viewColors{
		text:    lipgloss.NoColor{},
		muted:   lipgloss.NoColor{},
		stderr:  lipgloss.NoColor{},
		success: lipgloss.NoColor{},
		primary: lipgloss.NoColor{},
		bgElem:  lipgloss.NoColor{},
	}
	if m.styles != nil {
		c.text = m.styles.Text.GetForeground()
		c.muted = m.styles.TextMuted.GetForeground()
		c.stderr = m.styles.Error.GetForeground()
		c.success = m.styles.Success.GetForeground()
		c.primary = m.styles.Primary.GetForeground()
		c.bgElem = m.styles.BackgroundElem.GetForeground()
	}
	return c
}

func (m Model) View() string {
	if !m.HasSession() {
		return ""
	}

	c := m.resolveColors()
	padX := 1
	cw := max(1, m.width-padX*2)

	headerBox := m.renderHeader(cw, c)
	footerBox := m.renderFooterSection(cw, c)
	logBox := m.renderLogSection(cw, lipgloss.Height(headerBox), c)

	gap := lipgloss.NewStyle().Width(cw).Render("")
	inner := lipgloss.JoinVertical(lipgloss.Left, headerBox, gap, logBox, gap, footerBox)
	return lipgloss.NewStyle().PaddingLeft(padX).PaddingRight(padX).Render(inner)
}

// renderHeader builds the command header, wrapping at --param boundaries.
func (m Model) renderHeader(cw int, c viewColors) string {
	dollarFg := c.success
	cmdFg := c.text
	if m.copiedHeader {
		cmdFg = c.success
	}

	dollarStyle := lipgloss.NewStyle().Foreground(dollarFg)
	cmdStyle := lipgloss.NewStyle().Foreground(cmdFg)
	lineW := cw - 2 // "$ " prefix width

	if ansi.StringWidth(m.command) <= lineW {
		return dollarStyle.Render("$") + cmdStyle.Render(" "+m.command)
	}

	// Split at --param boundaries for clean wrapping.
	parts := strings.SplitAfter(m.command, " --param")
	var lines []string
	current := parts[0]
	for _, part := range parts[1:] {
		candidate := strings.TrimSuffix(current, " --param") + " --param" + part
		if ansi.StringWidth(candidate) <= lineW {
			current = candidate
		} else {
			lines = append(lines, strings.TrimSuffix(current, " --param"))
			current = "--param" + part
		}
	}
	lines = append(lines, current)

	var headerLines []string
	for i, l := range lines {
		l = strings.TrimSpace(l)
		if i == 0 {
			headerLines = append(headerLines, dollarStyle.Render("$")+cmdStyle.Render(" "+l))
		} else {
			headerLines = append(headerLines, cmdStyle.Render("  "+l))
		}
	}
	return strings.Join(headerLines, "\n")
}

// renderFooterSection builds the footer row showing the log path.
func (m Model) renderFooterSection(cw int, c viewColors) string {
	var footerLine string
	if m.logPath != "" && !m.searching && !m.navigating {
		label := lipgloss.NewStyle().Foreground(c.muted).Render("Saved to ")
		pathFg := c.muted
		if m.copiedFooter {
			pathFg = c.success
		}
		path := lipgloss.NewStyle().Foreground(pathFg).Render(m.logPath)
		footerLine = label + path
	}
	return lipgloss.NewStyle().Width(cw).Render(footerLine)
}

// renderLogSection builds the scrollable log area with search bar.
func (m Model) renderLogSection(cw, headerH int, c viewColors) string {
	tw := max(1, cw-3) // cw - 2 indent - 1 scrollbar

	logContentStyle := lipgloss.NewStyle().Background(c.bgElem).Foreground(c.text)
	logStderrStyle := lipgloss.NewStyle().Background(c.bgElem).Foreground(c.stderr)
	logSuccessStyle := lipgloss.NewStyle().Background(c.bgElem).Foreground(c.success)
	thumbStyle := lipgloss.NewStyle().Background(c.bgElem).Foreground(c.primary)

	logTopPad := 1
	logBottomPad := 1
	logH := max(1, m.height-headerH-3-logTopPad-logBottomPad)
	searchBarH := 0
	if m.searching || m.navigating {
		searchBarH = 2
	}
	visibleH := max(1, logH-searchBarH)
	logW := max(1, cw-1)

	blankLine := logContentStyle.Width(logW).Render("")

	yOffset := m.vp.YOffset()
	if searchBarH > 0 && len(m.lines) > visibleH {
		maxOff := len(m.lines) - visibleH
		if yOffset > maxOff {
			yOffset = maxOff
		}
		if m.atBottom {
			yOffset = maxOff
		}
	}

	needsScrollbar := len(m.lines) > visibleH

	logLines := make([]string, 0, logH+logTopPad)
	logLines = append(logLines, blankLine) // top padding
	for i := range visibleH {
		idx := yOffset + i
		if idx < len(m.lines) {
			line := m.lines[idx]
			lineW := tw
			if needsScrollbar {
				lineW--
			}
			lineW = max(1, lineW)

			visible := m.truncateLine(line.Text, lineW)
			lineText := "  " + visible
			if line.IsStderr {
				logLines = append(logLines, logStderrStyle.Width(logW).Render(lineText))
			} else {
				logLines = append(logLines, logContentStyle.Width(logW).Render(lineText))
			}
		} else {
			logLines = append(logLines, blankLine)
		}
	}
	logLines = append(logLines, blankLine) // bottom padding

	if m.searching {
		logLines = append(logLines, logContentStyle.Width(logW).Render("  "+fmt.Sprintf("Search: %s▎", m.searchQuery)))
		logLines = append(logLines, blankLine)
	}
	if m.navigating && m.matchCount > 0 {
		matchInfo := fmt.Sprintf("[%d/%d]", m.matchIndex+1, m.matchCount)
		logLines = append(logLines, logSuccessStyle.Width(logW).Render("  "+m.searchQuery+" "+matchInfo+"  n/N next/prev  esc clear"))
		logLines = append(logLines, blankLine)
	}

	contentStr := strings.Join(logLines, "\n")
	scrollbar := m.renderScrollbar(logH+logTopPad+logBottomPad, yOffset, visibleH, logContentStyle, thumbStyle)
	return lipgloss.JoinHorizontal(lipgloss.Top, contentStr, scrollbar)
}

// truncateLine applies horizontal scrolling and truncation to a single line.
func (m Model) truncateLine(raw string, lineW int) string {
	rawWidth := ansi.StringWidth(raw)
	if m.xOffset > 0 || rawWidth > lineW {
		if m.xOffset >= rawWidth {
			return ""
		}
		endCol := min(m.xOffset+lineW, rawWidth)
		return ansi.Cut(raw, m.xOffset, endCol)
	}
	return ansi.Truncate(raw, lineW, "")
}



// renderScrollbar builds the scrollbar column with a pill-shaped thumb.
func (m Model) renderScrollbar(contentH, yOffset, visibleH int, trackStyle, thumbStyle lipgloss.Style) string {
	total := len(m.lines)
	if total <= visibleH {
		lines := make([]string, contentH)
		for i := range lines {
			lines[i] = trackStyle.Render(" ")
		}
		return strings.Join(lines, "\n")
	}

	thumbHeight := max(1, (contentH*contentH)/total)
	maxOffset := total - visibleH
	var thumbPos int
	if yOffset >= maxOffset {
		thumbPos = contentH - thumbHeight
	} else {
		thumbPos = (yOffset * contentH) / total
	}
	thumbPos = max(0, min(thumbPos, contentH-thumbHeight))

	var sb strings.Builder
	for i := range contentH {
		if i >= thumbPos && i < thumbPos+thumbHeight {
			sb.WriteString(thumbStyle.Render("▐"))
		} else {
			sb.WriteString(trackStyle.Render(" "))
		}
		if i < contentH-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// scrollbarCharAt returns the scrollbar character for a given row (used by tests).
func (m Model) scrollbarCharAt(visibleRow int) string {
	total := len(m.lines)
	bh := m.bodyHeight()
	if total <= bh {
		return " "
	}
	thumbHeight := max(1, (bh*bh)/total)
	maxOffset := total - bh
	yOffset := m.vp.YOffset()
	var thumbPos int
	if yOffset >= maxOffset {
		thumbPos = bh - thumbHeight
	} else {
		thumbPos = (yOffset * bh) / total
	}
	thumbPos = max(0, min(thumbPos, bh-thumbHeight))
	if visibleRow >= thumbPos && visibleRow < thumbPos+thumbHeight {
		return "▐" // must match renderScrollbar thumb glyph
	}
	return " " // must match renderScrollbar track glyph
}
