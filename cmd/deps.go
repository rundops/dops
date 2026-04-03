package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"dops/internal/adapters"
	"dops/internal/catalog"
	"dops/internal/config"
	"dops/internal/domain"
	"dops/internal/history"
	"dops/internal/theme"
	"dops/internal/vault"

	lipgloss "charm.land/lipgloss/v2"
)

// appDeps holds the shared dependencies loaded during bootstrap.
type appDeps struct {
	FS          config.FileSystem
	Store       *config.FileConfigStore
	Cfg         *domain.Config
	Vault       domain.VaultStore
	ThemeLoader *theme.FileThemeLoader
	Resolved    *theme.ResolvedTheme
	Styles      *theme.Styles
	IsDark      bool
	Loader      *catalog.DiskCatalogLoader
	Catalogs    []catalog.CatalogWithRunbooks
	History     history.ExecutionStore
}

// loadDeps performs the common bootstrap sequence: load config, vault, theme,
// and catalogs. Both the TUI and Web UI entry points call this instead of
// duplicating the logic.
func loadDeps(dopsDir string) (*appDeps, error) {
	configPath := filepath.Join(dopsDir, "config.json")
	themesDir := filepath.Join(dopsDir, "themes")
	fs := adapters.NewOSFileSystem()
	store := config.NewFileStore(fs, configPath)

	cfg, err := store.EnsureDefaults()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Vault: encrypted parameter storage.
	vaultPath := filepath.Join(dopsDir, "vault.json")
	keysDir := filepath.Join(dopsDir, "keys")
	vlt := vault.New(vaultPath, keysDir)

	// Load vars from vault into config.
	vars, err := vlt.Load()
	if err != nil {
		return nil, fmt.Errorf("load vault: %w", err)
	}
	cfg.Vars = *vars

	// Load theme.
	themeLoader := theme.NewFileLoader(fs, themesDir)
	themeFile, err := themeLoader.Load(cfg.Theme)
	if err != nil {
		return nil, fmt.Errorf("load theme: %w", err)
	}
	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	resolved, err := theme.Resolve(themeFile, isDark)
	if err != nil {
		return nil, fmt.Errorf("resolve theme: %w", err)
	}
	styles := theme.BuildStyles(resolved)

	// Load catalogs.
	loader := catalog.NewDiskLoader(fs)
	catalogs, err := loader.LoadAll(cfg.Catalogs, cfg.Defaults.MaxRiskLevel)
	if err != nil {
		return nil, fmt.Errorf("load catalogs: %w", err)
	}

	// Execution history store.
	historyDir := filepath.Join(dopsDir, "history")
	historyStore := history.NewFileStore(historyDir, 500)

	return &appDeps{
		FS:          fs,
		Store:       store,
		Cfg:         cfg,
		Vault:       vlt,
		ThemeLoader: themeLoader,
		Resolved:    resolved,
		Styles:      styles,
		IsDark:      isDark,
		Loader:      loader,
		Catalogs:    catalogs,
		History:     historyStore,
	}, nil
}
