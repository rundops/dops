package cli

import (
	"fmt"
	"io"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Help style constants — bright, legible on dark terminals.
var (
	helpHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7aa2f7"))

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0caf5"))

	helpSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#9ece6a")).
				MarginTop(1)

	helpCmdStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7aa2f7"))

	helpFlagShortStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7aa2f7"))

	helpFlagLongStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#c0caf5")).
				Bold(true)

	helpFlagDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#565f89"))

	helpExampleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#c0caf5")).
				Background(lipgloss.Color("#1f2335")).
				Padding(0, 1)

	helpMutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89"))
)

// HelpFunc returns a cobra-compatible help function with styled output.
func HelpFunc(cmd *cobra.Command, args []string) {
	w := cmd.OutOrStdout()
	renderHelp(w, cmd)
}

func renderHelp(w io.Writer, cmd *cobra.Command) {
	fmt.Fprintln(w)

	// Header: name + description
	desc := cmd.Long
	if desc == "" {
		desc = cmd.Short
	}

	if cmd.HasParent() {
		fmt.Fprintf(w, "  %s\n", helpHeaderStyle.Render(cmd.CommandPath()))
		if desc != "" {
			fmt.Fprintf(w, "  %s\n", helpDescStyle.Render(desc))
		}
	} else {
		cat1 := helpMutedStyle.Render("/\\_/\\")
		cat2 := helpMutedStyle.Render("( =^.^= )")
		// Position ears at a fixed column, center face underneath.
		// Ears "/\_/\" = 5 chars, face "( =^.^= )" = 10 chars.
		// Face starts 2 chars left of ears to center it visually.
		earCol := 52
		faceCol := earCol - 2

		line1 := fmt.Sprintf("  %s", helpHeaderStyle.Render("do(ops) cli"))
		pad1 := earCol - lipgloss.Width(line1)
		if pad1 < 2 {
			pad1 = 2
		}
		fmt.Fprintf(w, "%s%s%s\n", line1, strings.Repeat(" ", pad1), cat1)

		line2 := ""
		if desc != "" {
			line2 = fmt.Sprintf("  %s", helpDescStyle.Render(desc))
		}
		pad2 := faceCol - lipgloss.Width(line2)
		if pad2 < 2 {
			pad2 = 2
		}
		fmt.Fprintf(w, "%s%s%s\n", line2, strings.Repeat(" ", pad2), cat2)
	}

	// Usage
	fmt.Fprintf(w, "\n%s\n\n", helpSectionStyle.Render("  USAGE"))

	usageLine := cmd.UseLine()
	if cmd.HasAvailableSubCommands() {
		usageLine = cmd.CommandPath() + " [command]"
	}
	fmt.Fprintf(w, "    %s\n", helpExampleStyle.Render(usageLine))

	// Examples
	if cmd.Example != "" {
		for _, line := range strings.Split(cmd.Example, "\n") {
			fmt.Fprintf(w, "    %s\n", helpExampleStyle.Render(line))
		}
	}

	// Commands — group into sections
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(w, "\n%s\n\n", helpSectionStyle.Render("  COMMANDS"))
		renderCommands(w, cmd)
	}

	// Flags
	if cmd.HasAvailableLocalFlags() {
		fmt.Fprintf(w, "\n%s\n\n", helpSectionStyle.Render("  FLAGS"))
		renderFlags(w, cmd)
	}

	// Inherited flags
	if cmd.HasAvailableInheritedFlags() {
		fmt.Fprintf(w, "\n%s\n\n", helpSectionStyle.Render("  GLOBAL FLAGS"))
		renderInheritedFlags(w, cmd)
	}

	// Footer
	if cmd.HasAvailableSubCommands() {
		fmt.Fprintf(w, "\n  %s\n",
			helpMutedStyle.Render(fmt.Sprintf(
				"Use \"%s [command] --help\" for more information about a command.",
				cmd.CommandPath())))
	}
	fmt.Fprintln(w)
}

func renderCommands(w io.Writer, parent *cobra.Command) {
	// Find max command name width for alignment.
	maxW := 0
	for _, c := range parent.Commands() {
		if !c.IsAvailableCommand() && c.Name() != "help" {
			continue
		}
		if len(c.Name()) > maxW {
			maxW = len(c.Name())
		}
	}

	for _, c := range parent.Commands() {
		if !c.IsAvailableCommand() && c.Name() != "help" {
			continue
		}
		name := helpCmdStyle.Render(c.Name())
		pad := strings.Repeat(" ", maxW-len(c.Name())+4)
		desc := helpFlagDescStyle.Render(c.Short)
		fmt.Fprintf(w, "    %s%s%s\n", name, pad, desc)
	}
}

func renderFlags(w io.Writer, cmd *cobra.Command) {
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		renderFlag(w, f)
	})
}

func renderInheritedFlags(w io.Writer, cmd *cobra.Command) {
	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		renderFlag(w, f)
	})
}

func renderFlag(w io.Writer, f *pflag.Flag) {
	if f.Hidden {
		return
	}

	var short string
	if f.Shorthand != "" {
		short = helpFlagShortStyle.Render("-"+f.Shorthand) + "  "
	} else {
		short = "    "
	}

	long := helpFlagLongStyle.Render("--" + f.Name)

	// Pad to align descriptions.
	nameW := len(f.Name)
	if f.Shorthand != "" {
		nameW += 4 // "-x  " prefix
	} else {
		nameW += 4 // "    " prefix
	}
	pad := ""
	if nameW < 20 {
		pad = strings.Repeat(" ", 20-nameW)
	} else {
		pad = "  "
	}

	desc := helpFlagDescStyle.Render(f.Usage)
	defVal := ""
	if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
		defVal = helpMutedStyle.Render(fmt.Sprintf(" (default: %s)", f.DefValue))
	}

	fmt.Fprintf(w, "    %s%s%s%s%s\n", short, long, pad, desc, defVal)
}
