package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dops/internal/domain"
	"dops/internal/theme"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Location returns the raw path or URL string for a runbook's catalog.
func Location(rb *domain.Runbook, cat *domain.Catalog) string {
	if rb == nil || cat == nil {
		return ""
	}
	if cat.URL != "" {
		return cat.URL
	}
	if cat.Path != "" {
		return cat.RunbookRoot() + "/" + rb.Name + "/runbook.yaml"
	}
	return ""
}

// Render returns the metadata content WITHOUT a border.
// The parent layout wraps it in a border for consistent alignment.
// When copied is true, the location line shows "Copied to Clipboard!" instead.
func Render(rb *domain.Runbook, cat *domain.Catalog, width int, copied bool, styles *theme.Styles) string {
	if rb == nil {
		return "  No runbook selected"
	}

	nameStyle := lipgloss.NewStyle().Bold(true)
	descStyle := lipgloss.NewStyle()
	mutedStyle := lipgloss.NewStyle()
	successStyle := lipgloss.NewStyle()

	if styles != nil {
		nameStyle = styles.Text.Bold(true)
		descStyle = styles.TextMuted
		mutedStyle = styles.TextMuted
		successStyle = styles.Success
	}

	var b strings.Builder
	fmt.Fprintf(&b, " %s %s\n", nameStyle.Render(rb.Name), mutedStyle.Render(rb.Version))
	fmt.Fprintf(&b, " %s\n", riskBadge(rb.RiskLevel, styles))
	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, " %s", descStyle.Render(rb.Description))

	if cat != nil {
		b.WriteString("\n\n")
		location := Location(rb, cat)
		// Truncate to available width (minus 2 for leading space and border padding)
		// to prevent wrapping that misaligns the metadata and output panes.
		locW := max(1, width-2)
		location = ansi.Truncate(location, locW, "…")
		if copied {
			// Flash the path green on copy — don't replace the text.
			fmt.Fprintf(&b, " %s", successStyle.Render(location))
		} else {
			linkStyle := mutedStyle
			if cat.URL != "" {
				linkStyle = linkStyle.Hyperlink(cat.URL)
			} else {
				linkStyle = linkStyle.Hyperlink("file://" + expandPath(location))
			}
			fmt.Fprintf(&b, " %s", linkStyle.Render(location))
		}
	}

	return b.String()
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func riskBadge(level domain.RiskLevel, styles *theme.Styles) string {
	label := string(level)
	if styles == nil {
		return label
	}
	switch level {
	case domain.RiskLow:
		return styles.RiskLow.Render(label)
	case domain.RiskMedium:
		return styles.RiskMedium.Render(label)
	case domain.RiskHigh:
		return styles.RiskHigh.Render(label)
	case domain.RiskCritical:
		return styles.RiskCritical.Render(label)
	default:
		return label
	}
}
