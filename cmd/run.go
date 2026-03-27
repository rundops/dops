package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"dops/internal/adapters"
	"dops/internal/catalog"
	"dops/internal/config"
	"dops/internal/domain"
	"dops/internal/vars"
	"dops/internal/vault"

	"github.com/spf13/cobra"
)

func newRunCmd(dopsDir string) *cobra.Command {
	var params []string
	var dryRun bool
	var noSave bool

	cmd := &cobra.Command{
		Use:   "run <id>",
		Short: "Execute a runbook by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			if err := domain.ValidateRunbookID(id); err != nil {
				return err
			}

			configPath := filepath.Join(dopsDir, "config.json")
			keysDir := filepath.Join(dopsDir, "keys")
			fs := adapters.NewOSFileSystem()
			store := config.NewFileStore(fs, configPath)

			cfg, err := store.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Load vars from vault.
			vaultPath := filepath.Join(dopsDir, "vault.json")
			vlt := vault.New(vaultPath, keysDir)
			vaultVars, err := vlt.Load()
			if err != nil {
				return fmt.Errorf("load vault: %w", err)
			}
			cfg.Vars = *vaultVars

			loader := catalog.NewDiskLoader(fs)
			_, err = loader.LoadAll(cfg.Catalogs, cfg.Defaults.MaxRiskLevel)
			if err != nil {
				return fmt.Errorf("load catalogs: %w", err)
			}

			rb, cat, err := loader.FindByID(id)
			if err != nil {
				return fmt.Errorf("runbook %q not found", id)
			}

			// Check risk policy
			ceiling := cat.Policy.MaxRiskLevel
			if ceiling == "" {
				ceiling = cfg.Defaults.MaxRiskLevel
			}
			if rb.RiskLevel.Exceeds(ceiling) {
				return fmt.Errorf("runbook %q blocked by risk policy (%s > %s)", id, rb.RiskLevel, ceiling)
			}

			// Resolve saved vars
			resolver := vars.NewDefaultResolver()
			resolved := resolver.Resolve(cfg, cat.Name, rb.Name, rb.Parameters)

			// Apply --param overrides
			for _, p := range params {
				parts := strings.SplitN(p, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid param format: %q (expected key=value)", p)
				}
				resolved[parts[0]] = parts[1]
			}

			if dryRun {
				return printDryRun(cmd, rb, resolved)
			}

			// Save inputs to config (unless --no-save)
			if !noSave {
				if err := saveInputs(cfg, vlt, rb, cat.Name, resolved); err != nil {
					return fmt.Errorf("save inputs: %w", err)
				}
			}

			// Execute
			catPath := expandHome(cat.RunbookRoot())
			scriptPath := filepath.Join(filepath.Dir(
				filepath.Join(catPath, rb.Name, "runbook.yaml"),
			), rb.Script)

			return executeScript(cmd, scriptPath, resolved, cat.Name, rb.Name)
		},
	}

	cmd.Flags().StringArrayVar(&params, "param", nil, "Parameter override (key=value, repeatable)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show resolved command without executing")
	cmd.Flags().BoolVar(&noSave, "no-save", false, "Execute without saving inputs to vault")

	return cmd
}

func printDryRun(cmd *cobra.Command, rb *domain.Runbook, resolved map[string]string) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "dops run %s", rb.ID)
	for _, p := range rb.Parameters {
		if v, ok := resolved[p.Name]; ok {
			if p.Secret {
				fmt.Fprintf(out, " --param %s=****", p.Name)
			} else {
				fmt.Fprintf(out, " --param %s=%s", p.Name, v)
			}
		}
	}
	fmt.Fprintln(out)
	return nil
}

// saveInputs persists resolved parameter values to the encrypted vault.
// Values are stored as plaintext inside the vault's encrypted blob.
func saveInputs(cfg *domain.Config, vlt *vault.Vault, rb *domain.Runbook, catName string, resolved map[string]string) error {
	for _, p := range rb.Parameters {
		val, ok := resolved[p.Name]
		if !ok {
			continue
		}

		// Skip local/unscoped params — they aren't persisted.
		if p.Scope == "" || p.Scope == "local" {
			continue
		}

		var keyPath string
		switch p.Scope {
		case "global":
			keyPath = fmt.Sprintf("vars.global.%s", p.Name)
		case "catalog":
			keyPath = fmt.Sprintf("vars.catalog.%s.%s", catName, p.Name)
		case "runbook":
			keyPath = fmt.Sprintf("vars.catalog.%s.runbooks.%s.%s", catName, rb.Name, p.Name)
		default:
			keyPath = fmt.Sprintf("vars.global.%s", p.Name)
		}

		if err := config.Set(cfg, keyPath, val); err != nil {
			return err
		}
	}

	return vlt.Save(&cfg.Vars)
}

func executeScript(cmd *cobra.Command, scriptPath string, env map[string]string, catName, rbName string) error {
	c := exec.Command("sh", scriptPath)
	c.Env = os.Environ()
	for k, v := range env {
		c.Env = append(c.Env, fmt.Sprintf("%s=%s", strings.ToUpper(k), v))
	}
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()

	logPath := fmt.Sprintf("/tmp/%s-%s-%s.log",
		time.Now().Format("2006.01.02-150405"),
		catName, rbName,
	)

	logFile, err := os.Create(logPath)
	if err == nil {
		defer logFile.Close()
	}

	if err := c.Run(); err != nil {
		return fmt.Errorf("script failed: %w", err)
	}

	return nil
}

func expandHome(path string) string {
	return adapters.ExpandHome(path)
}
