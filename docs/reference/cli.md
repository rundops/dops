---
layout: default
title: CLI Commands
nav_order: 1
parent: Reference
---

# CLI Commands

## dops

Launch the interactive TUI. No arguments required.

```sh
dops
```

---

## dops run

Execute a runbook non-interactively.

```sh
dops run <catalog.runbook> [--param key=value ...]
```

| Flag | Description |
|------|-------------|
| `--param key=value` | Set a parameter value (repeatable) |
| `--dry-run` | Show what would execute without running |

---

## dops catalog

Manage runbook catalogs.

| Subcommand | Description |
|------------|-------------|
| `catalog list` | List configured catalogs |
| `catalog add <path>` | Add a local catalog directory |
| `catalog remove <name>` | Remove a catalog from config |
| `catalog install <url>` | Clone a catalog from a git repository |
| `catalog update <name>` | Pull latest changes for a git catalog |

```sh
# Install from git with a specific branch/tag
dops catalog install https://github.com/org/runbooks.git --ref v2.0

# Update a git-installed catalog
dops catalog update my-runbooks
```

---

## dops config

Read and write configuration.

| Subcommand | Description |
|------------|-------------|
| `config set key=value` | Set a configuration value |
| `config get key` | Get a configuration value |
| `config unset key` | Remove a saved value |
| `config list` | Display the full configuration (secrets masked) |

```sh
# Set theme
dops config set theme=tokyonight

# Save a global parameter
dops config set vars.global.region=us-east-1

# View config
dops config list
```

---

## dops mcp

MCP server for AI agent integration.

| Subcommand | Description |
|------------|-------------|
| `mcp serve` | Start the MCP server |
| `mcp tools` | List available MCP tools |

```sh
# Start stdio server (for Claude Code)
dops mcp serve

# Start HTTP server
dops mcp serve --transport http --port 8080

# Limit exposed risk level
dops mcp serve --allow-risk medium
```

---

## dops version

Print the version.

```sh
dops version
dops --version
```
