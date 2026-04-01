package tui

import (
	"strings"
	"time"

	"dops/internal/tui/metadata"
	"dops/internal/tui/output"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

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
	sidebarW := sidebarWidth(innerW) + layoutBorderSize*2 + layoutPadLeft
	inBounds = mx >= layoutMarginLeft && mx < layoutMarginLeft+sidebarW &&
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
	sidebarW := sidebarWidth(innerW) + layoutBorderSize*2 + layoutPadLeft
	sidebarRight := layoutMarginLeft + sidebarW

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
