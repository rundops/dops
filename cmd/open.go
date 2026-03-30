package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"time"

	"dops/internal/executor"
	"dops/internal/web"

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
			return runWebUI(WebUIOptions{
				DopsDir:   dopsDir,
				Port:      port,
				NoBrowser: noBrowser,
				Demo:      demo,
			})
		},
	}

	cmd.Flags().IntVar(&port, "port", 3000, "HTTP server port")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Start server without opening browser")
	cmd.Flags().BoolVar(&demo, "demo", false, "Demo mode: disable script execution and config writes")

	return cmd
}

// WebUIOptions holds the parameters for launching the web UI.
type WebUIOptions struct {
	DopsDir   string
	Port      int
	NoBrowser bool
	Demo      bool
}

func runWebUI(opts WebUIOptions) error {
	deps, err := loadDeps(opts.DopsDir)
	if err != nil {
		return err
	}

	var runner executor.Runner
	if opts.Demo {
		runner = executor.NewDemoRunner()
	} else {
		runner = executor.NewScriptRunner()
	}

	// Start web server.
	srv := web.NewServer(web.ServerDeps{
		Config:      deps.Cfg,
		ConfigStore: deps.Store,
		Catalogs:    deps.Catalogs,
		Loader:      deps.Loader,
		Runner:      runner,
		Vault:       deps.Vault,
		Theme:       deps.Resolved,
		ThemeLoader: deps.ThemeLoader,
		IsDark:      deps.IsDark,
		Port:        opts.Port,
		Demo:        opts.Demo,
	})

	if err := srv.Start(); err != nil {
		return fmt.Errorf("start web server: %w", err)
	}

	// Open browser.
	if !opts.NoBrowser {
		url := fmt.Sprintf("http://localhost:%d", opts.Port)
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
