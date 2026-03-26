package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"dops/internal/adapters"
	"dops/internal/config"
	"dops/internal/domain"

	"github.com/spf13/cobra"
)

const helloWorldRunbook = `name: hello-world
version: 1.0.0
description: Prints a greeting message to stdout
risk_level: low
script: script.sh
parameters:
  - name: greeting
    type: string
    required: true
    description: Greeting message
    scope: runbook
    default: Hello
  - name: name
    type: string
    required: true
    description: Name to greet
    scope: runbook
`

const helloWorldScript = `#!/bin/sh
set -eu

GREETING="${GREETING:?greeting is required}"
NAME="${NAME:?name is required}"

echo "${GREETING}, ${NAME}!"
`

func newInitCmd(dopsDir string) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize dops configuration",
		Long:  "Set up the ~/.dops directory with default config. If no catalogs exist, scaffolds a hello-world runbook to get started.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, dopsDir)
		},
	}
}

func runInit(cmd *cobra.Command, dopsDir string) error {
	out := cmd.OutOrStdout()
	fs := adapters.NewOSFileSystem()
	configPath := filepath.Join(dopsDir, "config.json")
	store := config.NewFileStore(fs, configPath)

	// Ensure config exists.
	cfg, err := store.EnsureDefaults()
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	fmt.Fprintf(out, "  %-40s %s\n", configPath, check(true))

	// If no catalogs configured, scaffold hello-world.
	if len(cfg.Catalogs) == 0 {
		catDir := filepath.Join(dopsDir, "catalogs", "default")
		rbDir := filepath.Join(catDir, "hello-world")

		if err := os.MkdirAll(rbDir, 0o750); err != nil {
			return fmt.Errorf("create runbook dir: %w", err)
		}

		rbPath := filepath.Join(rbDir, "runbook.yaml")
		if err := os.WriteFile(rbPath, []byte(helloWorldRunbook), 0o644); err != nil {
			return fmt.Errorf("write runbook.yaml: %w", err)
		}

		scriptPath := filepath.Join(rbDir, "script.sh")
		if err := os.WriteFile(scriptPath, []byte(helloWorldScript), 0o755); err != nil {
			return fmt.Errorf("write script.sh: %w", err)
		}
		fmt.Fprintf(out, "  %-40s %s\n", rbDir+"/", check(true))

		// Register the catalog.
		cfg.Catalogs = append(cfg.Catalogs, domain.Catalog{
			Name:   "default",
			Path:   catDir,
			Active: true,
		})
		if err := store.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Fprintf(out, "  %-40s %s\n", "default catalog registered", check(true))
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "  Run 'dops' to launch the TUI.")
	return nil
}

func check(ok bool) string {
	if ok {
		return "\u2713"
	}
	return "\u2717"
}
