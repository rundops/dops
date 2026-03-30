package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"dops/internal/adapters"
	"dops/internal/config"
	"dops/internal/domain"
	"dops/internal/vault"

	"github.com/spf13/cobra"
)

func newConfigCmd(dopsDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read and write dops configuration",
	}

	configPath := filepath.Join(dopsDir, "config.json")
	keysDir := filepath.Join(dopsDir, "keys")
	vaultPath := filepath.Join(dopsDir, "vault.json")
	fs := adapters.NewOSFileSystem()
	store := config.NewFileStore(fs, configPath)
	vlt := vault.New(vaultPath, keysDir)

	cmd.AddCommand(newConfigSetCmd(store, vlt))
	cmd.AddCommand(newConfigGetCmd(store, vlt))
	cmd.AddCommand(newConfigUnsetCmd(store, vlt))
	cmd.AddCommand(newConfigListCmd(store, vlt))

	return cmd
}

func newConfigSetCmd(store *config.FileConfigStore, vlt *vault.Vault) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set key=value",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.SplitN(args[0], "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("expected key=value, got %q", args[0])
			}
			key, value := parts[0], parts[1]

			cfg, err := store.Load()
			if err != nil {
				cfg, err = store.EnsureDefaults()
				if err != nil {
					return err
				}
			}

			// Load vars from vault into config for path routing.
			vars, err := vlt.Load()
			if err != nil {
				return fmt.Errorf("load vault: %w", err)
			}
			cfg.Vars = *vars

			if err := config.Set(cfg, key, value); err != nil {
				return fmt.Errorf("config set %q: %w", key, err)
			}

			// Vars go to vault, everything else to config.json.
			if strings.HasPrefix(key, "vars.") {
				return vlt.Save(&cfg.Vars)
			}
			return store.Save(cfg)
		},
	}

	return cmd
}

func newConfigGetCmd(store *config.FileConfigStore, vlt *vault.Vault) *cobra.Command {
	return &cobra.Command{
		Use:   "get key",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := store.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Load vars from vault for vars.* lookups.
			vars, err := vlt.Load()
			if err != nil {
				return fmt.Errorf("load vault: %w", err)
			}
			cfg.Vars = *vars

			val, err := config.Get(cfg, args[0])
			if err != nil {
				return fmt.Errorf("config get %q: %w", args[0], err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}
}

func newConfigUnsetCmd(store *config.FileConfigStore, vlt *vault.Vault) *cobra.Command {
	return &cobra.Command{
		Use:   "unset key",
		Short: "Remove a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := store.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Load vars from vault for unset routing.
			vars, err := vlt.Load()
			if err != nil {
				return fmt.Errorf("load vault: %w", err)
			}
			cfg.Vars = *vars

			if err := config.Unset(cfg, args[0]); err != nil {
				return fmt.Errorf("config unset %q: %w", args[0], err)
			}

			return vlt.Save(&cfg.Vars)
		},
	}
}

func newConfigListCmd(store *config.FileConfigStore, vlt *vault.Vault) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Display the full configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := store.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Vars are excluded from config.json serialization (json:"-"),
			// so build a combined view for display.
			type configView struct {
				Theme    string           `json:"theme"`
				Defaults json.RawMessage  `json:"defaults"`
				Catalogs json.RawMessage  `json:"catalogs"`
				Vars     *domain.Vars     `json:"vars,omitempty"`
			}

			vars, vaultErr := vlt.Load()
			view := configView{Theme: cfg.Theme}
			if b, e := json.Marshal(cfg.Defaults); e == nil {
				view.Defaults = b
			}
			if b, e := json.Marshal(cfg.Catalogs); e == nil {
				view.Catalogs = b
			}
			if vaultErr == nil && (len(vars.Global) > 0 || len(vars.Catalog) > 0) {
				view.Vars = vars
			}

			data, err := json.MarshalIndent(view, "", "  ")
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
}
