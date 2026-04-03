// +build ignore

// test-mcp-schema queries the dops MCP server and prints tool schemas.
// Usage: go run scripts/test-mcp-schema.go [tool-name-filter]
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"dops/internal/adapters"
	"dops/internal/catalog"
	"dops/internal/config"
	"dops/internal/mcp"
	"dops/internal/vault"
	"dops/internal/vars"
)

func main() {
	filter := ""
	if len(os.Args) > 1 {
		filter = os.Args[1]
	}

	dopsDir := os.Getenv("DOPS_HOME")
	if dopsDir == "" {
		home, _ := os.UserHomeDir()
		dopsDir = home + "/.dops"
	}

	fs := adapters.NewOSFileSystem()
	store := config.NewFileStore(fs, dopsDir+"/config.json")
	cfg, _ := store.EnsureDefaults()

	vlt := vault.New(dopsDir+"/vault.json", dopsDir+"/keys")
	v, _ := vlt.Load()
	cfg.Vars = *v

	loader := catalog.NewDiskLoader(fs)
	catalogs, _ := loader.LoadAll(cfg.Catalogs, cfg.Defaults.MaxRiskLevel)

	resolver := vars.NewDefaultResolver()

	for _, c := range catalogs {
		for _, rb := range c.Runbooks {
			if filter != "" && rb.ID != filter {
				continue
			}
			resolved := resolver.Resolve(cfg, c.Catalog.Name, rb.Name, rb.Parameters)
			schema, err := mcp.RunbookToInputSchema(rb, resolved)
			if err != nil {
				continue
			}

			var parsed any
			json.Unmarshal(schema, &parsed)
			out, _ := json.MarshalIndent(map[string]any{
				"tool":        rb.ID,
				"description": mcp.RunbookToDescription(rb),
				"inputSchema": parsed,
			}, "", "  ")
			fmt.Println(string(out))
		}
	}

	// Also show skills
	if filter == "" || filter == "--skills" {
		fmt.Println("\n--- Skills ---")
		for _, c := range catalogs {
			for _, sk := range c.Skills {
				fmt.Printf("  %s: %s [triggers: %s]\n", sk.ID, sk.Description, sk.Trigger)
			}
		}
	}

}
