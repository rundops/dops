package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dops/internal/adapters"
	catpkg "dops/internal/catalog"
	"dops/internal/cli"
	"dops/internal/config"
	"dops/internal/executor"
	"dops/internal/theme"
	"dops/internal/tui"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

func newRootCmd(dopsDir string) *cobra.Command {
	root := &cobra.Command{
		Use:           "dops",
		Short:         "Developer Operations TUI",
		Long:          "A terminal user interface for browsing and executing operational runbooks.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchTUI(dopsDir)
		},
	}

	root.AddCommand(newVersionCmd())
	root.AddCommand(newConfigCmd(dopsDir))
	root.AddCommand(newRunCmd(dopsDir))
	root.AddCommand(newCatalogCmd(dopsDir))
	root.AddCommand(newMCPCmd(dopsDir))

	return root
}

func launchTUI(dopsDir string) error {
	configPath := filepath.Join(dopsDir, "config.json")
	themesDir := filepath.Join(dopsDir, "themes")
	fs := adapters.NewOSFileSystem()
	store := config.NewFileStore(fs, configPath)

	cfg, err := store.EnsureDefaults()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Load theme
	themeLoader := theme.NewFileLoader(fs, themesDir)
	tf, err := themeLoader.Load(cfg.Theme)
	if err != nil {
		return fmt.Errorf("load theme: %w", err)
	}

	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	resolved, err := theme.Resolve(tf, isDark)
	if err != nil {
		return fmt.Errorf("resolve theme: %w", err)
	}

	styles := theme.BuildStyles(resolved)

	// Load catalogs
	loader := catpkg.NewDiskLoader(fs)
	catalogs, err := loader.LoadAll(cfg.Catalogs, cfg.Defaults.MaxRiskLevel)
	if err != nil {
		return fmt.Errorf("load catalogs: %w", err)
	}

	runner := executor.NewScriptRunner()
	logWriter := adapters.NewLogWriter("/tmp")

	altScreen := os.Getenv("DOPS_NO_ALT_SCREEN") == ""
	progRef := &tui.ProgramRef{}

	app := tui.NewAppWithDeps(tui.AppDeps{
		Styles:     styles,
		Store:      store,
		Runner:     runner,
		LogWriter:  logWriter,
		Config:     cfg,
		Catalogs:   catalogs,
		AltScreen:  altScreen,
		ProgramRef: progRef,
	})
	p := tea.NewProgram(app)
	progRef.P = p
	_, err = p.Run()
	return err
}

func Execute() {
	// DOPS_HOME overrides the default ~/.dops directory.
	// Useful for Docker containers and custom installations.
	dopsDir := os.Getenv("DOPS_HOME")
	if dopsDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		dopsDir = filepath.Join(home, ".dops")
	}
	cmd := newRootCmd(dopsDir)

	if err := cmd.Execute(); err != nil {
		title, detail := splitError(err)
		cli.FormatError(os.Stderr, title, detail)
		os.Exit(1)
	}
}

func splitError(err error) (string, string) {
	msg := err.Error()
	if i := strings.Index(msg, ": "); i > 0 {
		return strings.Title(msg[:i]), msg[i+2:]
	}
	return msg, ""
}
