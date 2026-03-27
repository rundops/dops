package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"dops/internal/adapters"
	"dops/internal/config"
	"dops/internal/domain"

	"github.com/spf13/cobra"
)

func newCatalogCmd(dopsDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Manage runbook catalogs",
	}

	cmd.AddCommand(newCatalogListCmd(dopsDir))
	cmd.AddCommand(newCatalogAddCmd(dopsDir))
	cmd.AddCommand(newCatalogRemoveCmd(dopsDir))
	cmd.AddCommand(newCatalogInstallCmd(dopsDir))
	cmd.AddCommand(newCatalogUpdateCmd(dopsDir))

	return cmd
}

func newCatalogListCmd(dopsDir string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured catalogs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(dopsDir)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tPATH\tURL\tACTIVE\tRISK POLICY")
			for _, c := range cfg.Catalogs {
				risk := string(c.Policy.MaxRiskLevel)
				if risk == "" {
					risk = "—"
				}
				url := c.URL
				if url == "" {
					url = "—"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n", c.Name, c.Path, url, c.Active, risk)
			}
			return w.Flush()
		},
	}
}

func newCatalogAddCmd(dopsDir string) *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Add a local catalog",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			catalogPath := args[0]
			info, err := os.Stat(catalogPath)
			if err != nil {
				return fmt.Errorf("path does not exist: %s", catalogPath)
			}
			if !info.IsDir() {
				return fmt.Errorf("path is not a directory: %s", catalogPath)
			}

			name := filepath.Base(catalogPath)
			absPath, err := filepath.Abs(catalogPath)
			if err != nil {
				return err
			}

			cfg, err := loadConfig(dopsDir)
			if err != nil {
				return err
			}

			// Check for duplicate.
			for _, c := range cfg.Catalogs {
				if c.Name == name {
					return fmt.Errorf("catalog %q already exists", name)
				}
			}

			cfg.Catalogs = append(cfg.Catalogs, domain.Catalog{
				Name:   name,
				Path:   absPath,
				Active: true,
			})

			return saveConfig(dopsDir, cfg)
		},
	}
}

func newCatalogRemoveCmd(dopsDir string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a catalog from config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := loadConfig(dopsDir)
			if err != nil {
				return err
			}

			found := false
			filtered := make([]domain.Catalog, 0, len(cfg.Catalogs))
			for _, c := range cfg.Catalogs {
				if c.Name == name {
					found = true
					continue
				}
				filtered = append(filtered, c)
			}
			if !found {
				return fmt.Errorf("catalog %q not found", name)
			}

			cfg.Catalogs = filtered
			return saveConfig(dopsDir, cfg)
		},
	}
}

func newCatalogInstallCmd(dopsDir string) *cobra.Command {
	var name string
	var ref string
	var subPath string

	cmd := &cobra.Command{
		Use:   "install <url>",
		Short: "Install a catalog from a git repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			if name == "" {
				// Derive name from URL: "https://github.com/org/repo.git" → "repo"
				base := filepath.Base(url)
				name = strings.TrimSuffix(base, ".git")
			}

			catalogsDir := filepath.Join(dopsDir, "catalogs")
			if err := os.MkdirAll(catalogsDir, 0o750); err != nil {
				return fmt.Errorf("create catalogs dir: %w", err)
			}

			targetDir := filepath.Join(catalogsDir, name)
			if _, err := os.Stat(targetDir); err == nil {
				return fmt.Errorf("directory already exists: %s", targetDir)
			}

			// Git clone.
			cloneArgs := []string{"clone", url, targetDir}
			if ref != "" {
				cloneArgs = []string{"clone", "--branch", ref, url, targetDir}
			}
			gitCmd := exec.Command("git", cloneArgs...)
			gitCmd.Stdout = os.Stdout
			gitCmd.Stderr = os.Stderr
			if err := gitCmd.Run(); err != nil {
				return fmt.Errorf("git clone failed: %w", err)
			}

			// Validate sub-path stays within the cloned repository.
			if subPath != "" {
				validated, err := validateSubPath(targetDir, subPath)
				if err != nil {
					_ = os.RemoveAll(targetDir)
					return err
				}
				subPath = validated
			}

			// Add to config.
			cfg, err := loadConfig(dopsDir)
			if err != nil {
				return err
			}

			cfg.Catalogs = append(cfg.Catalogs, domain.Catalog{
				Name:    name,
				Path:    targetDir,
				SubPath: subPath,
				URL:     url,
				Active:  true,
			})

			fmt.Printf("Installed catalog %q from %s\n", name, url)
			return saveConfig(dopsDir, cfg)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "catalog name (defaults to repo basename)")
	cmd.Flags().StringVar(&ref, "ref", "", "git ref to checkout (tag, branch, or commit)")
	cmd.Flags().StringVar(&subPath, "path", "", "subdirectory within the repo containing runbooks")
	return cmd
}

func newCatalogUpdateCmd(dopsDir string) *cobra.Command {
	return &cobra.Command{
		Use:   "update <name>",
		Short: "Update a git-installed catalog",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := loadConfig(dopsDir)
			if err != nil {
				return err
			}

			var cat *domain.Catalog
			for i := range cfg.Catalogs {
				if cfg.Catalogs[i].Name == name {
					cat = &cfg.Catalogs[i]
					break
				}
			}
			if cat == nil {
				return fmt.Errorf("catalog %q not found", name)
			}
			if cat.URL == "" {
				return fmt.Errorf("catalog %q is local-only (no URL), cannot update", name)
			}

			gitCmd := exec.Command("git", "-C", cat.Path, "pull")
			gitCmd.Stdout = os.Stdout
			gitCmd.Stderr = os.Stderr
			if err := gitCmd.Run(); err != nil {
				return fmt.Errorf("git pull failed: %w", err)
			}

			fmt.Printf("Updated catalog %q\n", name)
			return nil
		},
	}
}

// validateSubPath ensures sub stays inside root by resolving symlinks and
// checking the absolute prefix. Returns the cleaned relative path.
func validateSubPath(root, sub string) (string, error) {
	cleaned := filepath.Clean(sub)
	if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("sub-path %q must be a relative path within the repository", sub)
	}

	// Resolve to an absolute, symlink-free path and verify containment.
	absRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	candidate := filepath.Join(absRoot, cleaned)
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("sub-path %q does not exist in repository", sub)
	}
	if !strings.HasPrefix(resolved, absRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("sub-path %q escapes the repository", sub)
	}

	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("sub-path %q is not a directory", sub)
	}

	return cleaned, nil
}

func loadConfig(dopsDir string) (*domain.Config, error) {
	configPath := filepath.Join(dopsDir, "config.json")
	fs := adapters.NewOSFileSystem()
	store := config.NewFileStore(fs, configPath)
	return store.EnsureDefaults()
}

func saveConfig(dopsDir string, cfg *domain.Config) error {
	configPath := filepath.Join(dopsDir, "config.json")
	fs := adapters.NewOSFileSystem()
	store := config.NewFileStore(fs, configPath)
	return store.Save(cfg)
}
