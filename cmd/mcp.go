package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"dops/internal/adapters"
	"dops/internal/catalog"
	"dops/internal/config"
	"dops/internal/domain"
	"dops/internal/executor"
	"dops/internal/mcp"
	"dops/internal/vault"

	"github.com/spf13/cobra"
)

func newMCPCmd(dopsDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server for AI agent integration",
	}

	cmd.AddCommand(newMCPServeCmd(dopsDir))
	cmd.AddCommand(newMCPToolsCmd(dopsDir))

	return cmd
}

func newMCPServeCmd(dopsDir string) *cobra.Command {
	var transport string
	var port int
	var allowRisk string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		Long:  "Start an MCP server that exposes runbooks as tools for AI agents.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, catalogs, err := loadMCPDeps(dopsDir, domain.RiskLevel(allowRisk))
			if err != nil {
				return err
			}

			runner := executor.NewScriptRunner()
			srv := mcp.NewServer(mcp.ServerConfig{
				Version:  version,
				DopsHome: dopsDir,
				Catalogs: catalogs,
				Runner:   runner,
				Config:   cfg,
				MaxRisk:  domain.RiskLevel(allowRisk),
			})

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()

			switch transport {
			case "stdio":
				return srv.ServeStdio(ctx)
			case "http":
				addr := fmt.Sprintf(":%d", port)
				fmt.Fprintf(os.Stderr, "MCP server listening on %s\n", addr)
				return srv.ServeHTTP(ctx, addr)
			default:
				return fmt.Errorf("unknown transport: %s (use stdio or http)", transport)
			}
		},
	}

	cmd.Flags().StringVar(&transport, "transport", "stdio", "transport type (stdio or http)")
	cmd.Flags().IntVar(&port, "port", 8080, "HTTP port (only for http transport)")
	cmd.Flags().StringVar(&allowRisk, "allow-risk", "critical", "maximum risk level to expose (low, medium, high, critical)")

	return cmd
}

func newMCPToolsCmd(dopsDir string) *cobra.Command {
	var allowRisk string

	cmd := &cobra.Command{
		Use:   "tools",
		Short: "List available MCP tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, catalogs, err := loadMCPDeps(dopsDir, domain.RiskLevel(allowRisk))
			if err != nil {
				return err
			}

			for _, c := range catalogs {
				for _, rb := range c.Runbooks {
					if domain.RiskLevel(allowRisk) != "" && rb.RiskLevel.Exceeds(domain.RiskLevel(allowRisk)) {
						continue
					}
					fmt.Printf("%-30s %s\n", rb.ID, mcp.RunbookToDescription(rb))
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&allowRisk, "allow-risk", "critical", "maximum risk level to show")
	return cmd
}

func loadMCPDeps(dopsDir string, maxRisk domain.RiskLevel) (*domain.Config, []catalog.CatalogWithRunbooks, error) {
	configPath := filepath.Join(dopsDir, "config.json")
	keysDir := filepath.Join(dopsDir, "keys")
	fs := adapters.NewOSFileSystem()
	store := config.NewFileStore(fs, configPath)

	cfg, err := store.EnsureDefaults()
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}

	// Load vars from vault.
	vaultPath := filepath.Join(dopsDir, "vault.json")
	vlt := vault.New(vaultPath, keysDir)
	vars, err := vlt.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("load vault: %w", err)
	}
	cfg.Vars = *vars

	loader := catalog.NewDiskLoader(fs)
	catalogs, err := loader.LoadAll(cfg.Catalogs, cfg.Defaults.MaxRiskLevel)
	if err != nil {
		return nil, nil, fmt.Errorf("load catalogs: %w", err)
	}

	return cfg, catalogs, nil
}
