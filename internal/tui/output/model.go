package output

import (
	"fmt"
	"image/color"
	"strings"
	"time"

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
	copyFlash    bool // show "✓ Copied" badge in log top-right
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

func (m Model) Command() string       { return m.command }
func (m Model) LogPath() string        { return m.logPath }
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
// Header(1) + gap(1) + logTopPad(1) + log + logBottomPad(1) + gap(1) + Footer(1) = height.
func (m Model) bodyHeight() int {
	return max(1, m.height-6) // minus header + footer + 2 gaps + top pad + bottom pad
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
		// Start text selection in the log area.
		// Coordinates are output-local (translated by the app).
		// Layout: header(row 0) + gap(row 1) + topPad(row 2) → first line at row 3
		// Column: padX(1) + indent(2) → text starts at col 3
		logTop := 3
		logCol := 3
		row := msg.Y - logTop
		col := msg.X - logCol
		if row >= 0 && row < m.bodyHeight() {
			m.selection.Reset()
			m.selection.Active = true
			m.selection.AnchorX = max(0, col)
			m.selection.AnchorY = row
			m.selection.FocusX = max(0, col)
			m.selection.FocusY = row
		} else {
			m.selection.Reset()
		}
		return m, nil

	case tea.MouseMotionMsg:
		// In CellMotion mode, motion events are only sent when a button
		// is held, so no need to check msg.Button.
		if m.selection.Active {
			logTop := 3
			logCol := 3
			m.selection.FocusX = max(0, msg.X-logCol)
			m.selection.FocusY = max(0, msg.Y-logTop)
		}
		return m, nil

	case tea.MouseReleaseMsg:
		if m.selection.Active {
			logTop := 3
			logCol := 3
			m.selection.FocusX = max(0, msg.X-logCol)
			m.selection.FocusY = max(0, msg.Y-logTop)

			if !m.selection.IsEmpty() {
				text := m.selection.ExtractText(m.visibleLineTexts())
				if text != "" {
					m.copyFlash = true
					return m, tea.Batch(
						tea.SetClipboard(text),
						tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg {
							return CopyFlashExpiredMsg{}
						}),
					)
				}
			}
		}
		return m, nil

	case CopyFlashExpiredMsg:
		m.copyFlash = false
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
			text := m.selection.ExtractText(m.visibleLineTexts())
			m.selection.Reset()
			if text != "" {
				return m, tea.SetClipboard(text)
			}
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

func (m Model) View() string {
	// --- Resolve colors ---
	var textFg, mutedFg, stderrFg, successFg, primaryFg color.Color
	var bgElemColor color.Color
	textFg = lipgloss.NoColor{}
	mutedFg = lipgloss.NoColor{}
	stderrFg = lipgloss.NoColor{}
	successFg = lipgloss.NoColor{}
	primaryFg = lipgloss.NoColor{}
	bgElemColor = lipgloss.NoColor{}
	if m.styles != nil {
		textFg = m.styles.Text.GetForeground()
		mutedFg = m.styles.TextMuted.GetForeground()
		stderrFg = m.styles.Error.GetForeground()
		successFg = m.styles.Success.GetForeground()
		primaryFg = m.styles.Primary.GetForeground()
		bgElemColor = m.styles.BackgroundElem.GetForeground()
	}

	if !m.HasSession() {
		return ""
	}

	// No section borders — the app renders the outer border. Content is flat.
	// m.width is the content width INSIDE the app's outer border.
	// Reserve 1 char left + 1 char right padding across all sections.
	padX := 1
	cw := max(1, m.width-padX*2)
	tw := max(1, cw-3) // line width for log text (cw - 2 indent - 1 scrollbar)

	// === Header: 1 row ===
	var headerLine string
	if m.copiedHeader {
		headerLine = lipgloss.NewStyle().Foreground(successFg).Render("Copied to Clipboard!")
	} else {
		dollar := lipgloss.NewStyle().Foreground(successFg).Render("$")
		cmd := lipgloss.NewStyle().Foreground(textFg).Render(" " + m.command)
		headerLine = ansi.Truncate(dollar+cmd, cw, "")
	}
	headerBox := lipgloss.NewStyle().Width(cw).Render(headerLine)

	// === Footer: 1 row ===
	var footerLine string
	if m.copiedFooter {
		footerLine = lipgloss.NewStyle().Foreground(successFg).Render("Copied to Clipboard!")
	} else if m.logPath != "" && !m.searching && !m.navigating {
		footerLine = lipgloss.NewStyle().Foreground(mutedFg).Render("Saved to " + m.logPath)
	}
	footerBox := lipgloss.NewStyle().Width(cw).Render(footerLine)

	// === Log: fills remaining height ===
	logContentStyle := lipgloss.NewStyle().Background(bgElemColor).Foreground(textFg)
	logStderrStyle := lipgloss.NewStyle().Background(bgElemColor).Foreground(stderrFg)
	logSuccessStyle := lipgloss.NewStyle().Background(bgElemColor).Foreground(successFg)
	thumbStyle := lipgloss.NewStyle().Background(bgElemColor).Foreground(primaryFg)
	// Selection highlight: primary background with dark foreground (matches legacy).
	selectionStyle := lipgloss.NewStyle().Background(primaryFg).Foreground(bgElemColor)

	// Header(1) + gap(1) + logTopPad(1) + visibleLines + logBottomPad(1) + gap(1) + Footer(1) = height.
	logTopPad := 1
	logBottomPad := 1
	logH := max(1, m.height-4-logTopPad-logBottomPad)
	searchBarH := 0
	if m.searching || m.navigating {
		searchBarH = 2
	}
	flashPadH := 0
	if m.copyFlash {
		flashPadH = 2 // blank row above + blank row below the badge
	}
	visibleH := max(1, logH-searchBarH-flashPadH)
	logW := max(1, cw-1) // 1 col for scrollbar

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
	if m.copyFlash {
		// Blank row above badge to push it down from the top edge.
		logLines = append(logLines, blankLine)
		// Show badge right-aligned with 1-char right padding.
		badgeText := "Copied to Clipboard!"
		badge := lipgloss.NewStyle().
			Background(bgElemColor).
			Foreground(successFg).
			Render(badgeText)
		badgeW := ansi.StringWidth(badge)
		rightPad := 1
		pad := logW - badgeW - rightPad
		if pad > 0 {
			logLines = append(logLines, logContentStyle.Render(strings.Repeat(" ", pad))+badge+logContentStyle.Render(" "))
		} else {
			logLines = append(logLines, badge)
		}
		// Blank row below badge for spacing from log content.
		logLines = append(logLines, blankLine)
	} else {
		logLines = append(logLines, blankLine) // top padding inside log
	}
	for i := range visibleH {
		idx := yOffset + i
		if idx < len(m.lines) {
			line := m.lines[idx]
			lineW := tw
			if needsScrollbar {
				lineW--
			}
			lineW = max(1, lineW)

			raw := line.Text
			rawWidth := ansi.StringWidth(raw)
			var visible string
			if m.xOffset > 0 || rawWidth > lineW {
				endCol := min(m.xOffset+lineW, rawWidth)
				if m.xOffset >= rawWidth {
					visible = ""
				} else {
					visible = ansi.Cut(raw, m.xOffset, endCol)
				}
			} else {
				visible = ansi.Truncate(raw, lineW, "")
			}

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

	logLines = append(logLines, blankLine) // bottom padding inside log

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
	logBox := lipgloss.JoinHorizontal(lipgloss.Top, contentStr, scrollbar)

	// Gap rows between sections (empty, terminal background).
	gap := lipgloss.NewStyle().Width(cw).Render("")

	inner := lipgloss.JoinVertical(lipgloss.Left, headerBox, gap, logBox, gap, footerBox)
	result := lipgloss.NewStyle().PaddingLeft(padX).PaddingRight(padX).Render(inner)
	return m.highlightSelection(result, selectionStyle)
}

func (m Model) matchLineSet() map[int]bool {
	set := make(map[int]bool, len(m.matchLines))
	for _, idx := range m.matchLines {
		set[idx] = true
	}
	return set
}

// highlightSelection post-processes the rendered output to apply the selection
// style to the character range covered by the current selection. Uses ANSI-aware
// Cut to split styled lines, matching the legacy implementation.
func (m Model) highlightSelection(rendered string, hlStyle lipgloss.Style) string {
	if !m.selection.Active || m.selection.IsEmpty() {
		return rendered
	}

	// Selection coordinates are relative to visible log content (row 0 = first
	// visible line). The rendered output has: header(row 0), gap(row 1),
	// topPad(row 2), then log content starting at row 3.
	logStartRow := 3 // offset from rendered output row 0 to first log content row
	startX, startY, endX, endY := m.selection.Bounds()

	// Shift selection to rendered output coordinates.
	startY += logStartRow
	endY += logStartRow
	// Shift X for the 1-col padding + 2-char indent.
	padIndent := 3
	startX += padIndent
	endX += padIndent

	lines := strings.Split(rendered, "\n")
	for i := range lines {
		if i < startY || i > endY {
			continue
		}

		lineWidth := ansi.StringWidth(lines[i])
		if lineWidth == 0 {
			continue
		}

		lx := 0
		rx := lineWidth
		if i == startY {
			lx = startX
		}
		if i == endY {
			rx = min(lineWidth, endX+1)
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
		return "█"
	}
	return "░"
}
