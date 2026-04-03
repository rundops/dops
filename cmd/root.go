package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dops/internal/adapters"
	"dops/internal/cli"
	"dops/internal/config"
	"dops/internal/domain"
	"dops/internal/executor"
	"dops/internal/tui"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

func newRootCmd(dopsDir string) *cobra.Command {
	root := &cobra.Command{
		Use:           "dops",
		Short:         "Developer Operations TUI",
		Long:          "a runbook toolkit for operators and AI agents.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchTUI(dopsDir)
		},
	}

	root.AddCommand(newVersionCmd())
	root.AddCommand(newInitCmd(dopsDir))
	root.AddCommand(newConfigCmd(dopsDir))
	root.AddCommand(newRunCmd(dopsDir))
	root.AddCommand(newCatalogCmd(dopsDir))
	root.AddCommand(newMCPCmd(dopsDir))
	root.AddCommand(newOpenCmd(dopsDir))
	root.AddCommand(newHistoryCmd(dopsDir))

	// Styled help for all commands.
	root.SetHelpFunc(cli.HelpFunc)

	return root
}

func launchTUI(dopsDir string) error {
	deps, err := loadDeps(dopsDir)
	if err != nil {
		return err
	}

	// Migrate vars from config.json to vault.json (one-time).
	configPath := filepath.Join(dopsDir, "config.json")
	if err := migrateVarsToVault(configPath, deps.Vault, deps.FS); err != nil {
		return fmt.Errorf("vault migration: %w", err)
	}

	runner := executor.NewScriptRunner()
	logWriter := adapters.NewLogWriter(os.TempDir())

	altScreen := os.Getenv("DOPS_NO_ALT_SCREEN") == ""
	progRef := &tui.ProgramRef{}

	app := tui.NewAppWithDeps(tui.AppDeps{
		Styles:     deps.Styles,
		Store:      deps.Store,
		Runner:     runner,
		LogWriter:  logWriter,
		Config:     deps.Cfg,
		Catalogs:   deps.Catalogs,
		AltScreen:  altScreen,
		ProgramRef: progRef,
		Version:    version,
		DopsDir:    dopsDir,
		Vault:      deps.Vault,
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
		return titleCase(msg[:i]), msg[i+2:]
	}
	return msg, ""
}

// titleCase capitalizes the first letter of s. Replaces deprecated strings.Title
// without pulling in golang.org/x/text for a single call site.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// migrateVarsToVault moves saved parameter values from config.json to vault.json.
// This is a one-time migration for users upgrading from v0.2.0 to v0.3.0.
// If vault.json already exists or config.json has no vars, this is a no-op.
func migrateVarsToVault(configPath string, vlt domain.VaultStore, fs config.FileSystem) error {
	if vlt.Exists() {
		return nil
	}

	data, err := fs.ReadFile(configPath)
	if err != nil {
		return nil // no config.json → nothing to migrate
	}

	// Check if config.json contains a "vars" key.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil // malformed config → skip migration
	}

	varsRaw, ok := raw["vars"]
	if !ok {
		return nil // no vars in config
	}

	// Parse the vars.
	var vars domain.Vars
	if err := json.Unmarshal(varsRaw, &vars); err != nil {
		return fmt.Errorf("parse vars from config.json: %w", err)
	}

	// Skip if vars are empty.
	if len(vars.Global) == 0 && len(vars.Catalog) == 0 {
		return nil
	}

	// Save vars to vault.
	if err := vlt.Save(&vars); err != nil {
		return fmt.Errorf("save vars to vault: %w", err)
	}

	// Remove vars from config.json and rewrite.
	delete(raw, "vars")
	cleaned, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("rewrite config.json: %w", err)
	}
	if err := fs.WriteFile(configPath, cleaned, 0o644); err != nil {
		return fmt.Errorf("write cleaned config.json: %w", err)
	}

	return nil
}
