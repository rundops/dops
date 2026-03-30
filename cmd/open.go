package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"dops/internal/adapters"
	"dops/internal/catalog"
	"dops/internal/config"
	"dops/internal/executor"
	"dops/internal/theme"
	"dops/internal/vault"
	"dops/internal/web"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

func newOpenCmd(dopsDir string) *cobra.Command {
	var port int
	var noBrowser bool
	var demo bool

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Launch the web UI in a browser",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWebUI(dopsDir, port, noBrowser, demo)
		},
	}

	cmd.Flags().IntVar(&port, "port", 3000, "HTTP server port")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Start server without opening browser")
	cmd.Flags().BoolVar(&demo, "demo", false, "Demo mode: disable script execution and config writes")

	return cmd
}

func runWebUI(dopsDir string, port int, noBrowser, demo bool) error {
	// Load config.
	configPath := filepath.Join(dopsDir, "config.json")
	fs := adapters.NewOSFileSystem()
	store := config.NewFileStore(fs, configPath)

	cfg, err := store.EnsureDefaults()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Load vault.
	vaultPath := filepath.Join(dopsDir, "vault.json")
	keysDir := filepath.Join(dopsDir, "keys")
	vlt := vault.New(vaultPath, keysDir)
	vars, err := vlt.Load()
	if err != nil {
		return fmt.Errorf("load vault: %w", err)
	}
	cfg.Vars = *vars

	// Load theme.
	themesDir := filepath.Join(dopsDir, "themes")
	themeLoader := theme.NewFileLoader(fs, themesDir)
	themeFile, err := themeLoader.Load(cfg.Theme)
	if err != nil {
		return fmt.Errorf("load theme: %w", err)
	}
	isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	resolved, err := theme.Resolve(themeFile, isDark)
	if err != nil {
		return fmt.Errorf("resolve theme: %w", err)
	}

	// Load catalogs.
	loader := catalog.NewDiskLoader(fs)
	catalogs, err := loader.LoadAll(cfg.Catalogs, cfg.Defaults.MaxRiskLevel)
	if err != nil {
		return fmt.Errorf("load catalogs: %w", err)
	}

	var runner executor.Runner
	if demo {
		runner = executor.NewDemoRunner()
	} else {
		runner = executor.NewScriptRunner()
	}

	// Start web server.
	srv := web.NewServer(web.ServerDeps{
		Config:      cfg,
		ConfigStore: store,
		Catalogs:    catalogs,
		Loader:      loader,
		Runner:      runner,
		Vault:       vlt,
		Theme:       resolved,
		ThemeLoader: themeLoader,
		IsDark:      isDark,
		Port:        port,
		Demo:        demo,
	})

	if err := srv.Start(); err != nil {
		return fmt.Errorf("start web server: %w", err)
	}

	// Open browser.
	if !noBrowser {
		url := fmt.Sprintf("http://localhost:%d", port)
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "Could not open browser: %v\nOpen %s manually.\n", err, url)
		}
	}

	// Wait for interrupt.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

// openBrowser opens the given URL in the user's default browser.
// The URL is always constructed internally (http://localhost:<port>),
// never from external input.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url) // #nosec G204 -- URL is constructed internally
	case "linux":
		cmd = exec.Command("xdg-open", url) // #nosec G204 -- URL is constructed internally
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url) // #nosec G204 -- URL is constructed internally
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
