package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"dops/internal/adapters"
	"dops/internal/catalog"
	"dops/internal/config"
	"dops/internal/domain"
	"dops/internal/executor"
	"dops/internal/history"
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

			rb, cat, err := resolveRunbook(loader, id)
			if err != nil {
				return err
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
				if err := saveInputs(saveInputsParams{
					Cfg:      cfg,
					Vault:    vlt,
					Runbook:  rb,
					CatName:  cat.Name,
					Resolved: resolved,
				}); err != nil {
					return fmt.Errorf("save inputs: %w", err)
				}
			}

			// Execute
			catPath := adapters.ExpandHome(cat.RunbookRoot())
			scriptPath := filepath.Join(filepath.Dir(
				filepath.Join(catPath, rb.Name, "runbook.yaml"),
			), rb.Script)

			historyDir := filepath.Join(dopsDir, "history")
			historyStore := history.NewFileStore(historyDir, 500)

			return executeScript(cmd, scriptPath, resolved, rb, cat, historyStore)
		},
	}

	cmd.Flags().StringArrayVar(&params, "param", nil, "Parameter override (key=value, repeatable)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show resolved command without executing")
	cmd.Flags().BoolVar(&noSave, "no-save", false, "Execute without saving inputs to vault")

	return cmd
}

// resolveRunbook looks up a runbook by ID first, then falls back to alias.
func resolveRunbook(loader *catalog.DiskCatalogLoader, id string) (*domain.Runbook, *domain.Catalog, error) {
	var rb *domain.Runbook
	var cat *domain.Catalog
	var err error
	if domain.ValidateRunbookID(id) == nil {
		rb, cat, err = loader.FindByID(id)
	}
	if rb == nil {
		rb, cat, err = loader.FindByAlias(id)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("runbook %q not found: %w", id, err)
	}
	return rb, cat, nil
}

func printDryRun(cmd *cobra.Command, rb *domain.Runbook, resolved map[string]string) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "dops run %s", rb.ID)
	for _, param := range rb.Parameters {
		if v, ok := resolved[param.Name]; ok {
			if param.Secret {
				fmt.Fprintf(out, " --param %s=****", param.Name)
			} else {
				fmt.Fprintf(out, " --param %s=%s", param.Name, v)
			}
		}
	}
	fmt.Fprintln(out)
	return nil
}

// saveInputsParams groups the arguments for saveInputs.
type saveInputsParams struct {
	Cfg      *domain.Config
	Vault    domain.VaultStore
	Runbook  *domain.Runbook
	CatName  string
	Resolved map[string]string
}

// saveInputs persists resolved parameter values to the encrypted vault.
// Values are stored as plaintext inside the vault's encrypted blob.
func saveInputs(p saveInputsParams) error {
	for _, param := range p.Runbook.Parameters {
		val, ok := p.Resolved[param.Name]
		if !ok {
			continue
		}

		// Skip local/unscoped params — they aren't persisted.
		if param.Scope == "" || param.Scope == "local" {
			continue
		}

		keyPath := vars.VarKeyPath(param.Scope, param.Name, p.CatName, p.Runbook.Name)

		if err := config.Set(p.Cfg, keyPath, val); err != nil {
			return fmt.Errorf("save input %q: %w", keyPath, err)
		}
	}

	return p.Vault.Save(&p.Cfg.Vars)
}

func executeScript(cmd *cobra.Command, scriptPath string, env map[string]string, rb *domain.Runbook, cat *domain.Catalog, historyStore history.ExecutionStore) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Start execution record.
	rec := domain.NewExecutionRecord(rb.ID, rb.Name, cat.Name, domain.ExecCLI)
	rec.Parameters = make(map[string]string, len(env))
	for k, v := range env {
		rec.Parameters[k] = v
	}
	var secretNames []string
	for _, p := range rb.Parameters {
		if p.Secret {
			secretNames = append(secretNames, strings.ToUpper(p.Name))
		}
	}
	rec.MaskSecrets(secretNames)

	runner := executor.NewScriptRunner()
	lines, errs := runner.Run(ctx, scriptPath, env)

	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	lineCount := 0
	lastLine := ""
	for line := range lines {
		lineCount++
		if text := strings.TrimSpace(line.Text); text != "" {
			lastLine = text
		}
		if line.IsStderr {
			fmt.Fprintln(stderr, line.Text)
		} else {
			fmt.Fprintln(stdout, line.Text)
		}
	}

	exitCode := 0
	if err := <-errs; err != nil {
		exitCode = 1
	}

	rec.Complete(exitCode, lineCount, lastLine)
	if historyStore != nil {
		_ = historyStore.Record(rec)
	}

	if exitCode != 0 {
		return fmt.Errorf("script failed")
	}
	return nil
}
