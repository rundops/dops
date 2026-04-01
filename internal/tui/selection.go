package tui

import (
	"strings"

	"dops/internal/theme"
	"dops/internal/tui/output"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

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

// selectionBounds defines the rectangular region of the output pane within
// the full terminal view, used to clamp text selection highlighting.
type selectionBounds struct {
	top, bottom, left, right int
}

// applySelectionHighlight post-processes the full terminal view to highlight
// the selected text, confined within the output pane bounds.
func applySelectionHighlight(content string, sel output.TextSelection, styles *theme.Styles, bounds selectionBounds) string {
	hlStyle := lipgloss.NewStyle()
	if styles != nil {
		hlStyle = lipgloss.NewStyle().
			Background(styles.Primary.GetForeground()).
			Foreground(styles.BackgroundElem.GetForeground())
	}

	startX, startY, endX, endY := sel.Bounds()

	// Clamp to output pane bounds.
	if startY < bounds.top {
		startY = bounds.top
		startX = bounds.left
	}
	if endY > bounds.bottom {
		endY = bounds.bottom
		endX = bounds.right
	}
	if startY > endY {
		return content
	}

	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = highlightLine(lines[i], i, startX, startY, endX, endY, bounds, hlStyle)
	}

	return strings.Join(lines, "\n")
}

// highlightLine applies selection highlighting to a single line if it falls
// within the selection range. It returns the line unchanged when outside the
// range or when the line has zero visible width.
func highlightLine(line string, i, startX, startY, endX, endY int, bounds selectionBounds, hlStyle lipgloss.Style) string {
	if i < startY || i > endY {
		return line
	}

	lineWidth := ansi.StringWidth(line)
	if lineWidth == 0 {
		return line
	}

	var lx, rx int
	if i == startY && i == endY {
		lx = max(startX, bounds.left)
		rx = min(bounds.right, min(lineWidth, endX+1))
	} else if i == startY {
		lx = max(startX, bounds.left)
		rx = min(lineWidth, bounds.right)
	} else if i == endY {
		lx = bounds.left
		rx = min(bounds.right, min(lineWidth, endX+1))
	} else {
		lx = bounds.left
		rx = min(lineWidth, bounds.right)
	}

	if lx >= rx || lx >= lineWidth {
		return line
	}

	before := ansi.Cut(line, 0, lx)
	selected := ansi.Cut(line, lx, rx)
	after := ansi.Cut(line, rx, lineWidth)

	plain := ansi.Strip(selected)
	return before + "\x1b[0m" + hlStyle.Render(plain) + after
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
