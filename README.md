# dops

A terminal user interface for browsing and executing operational runbooks.

`dops` provides a browsable catalog of automation scripts that operators can select, parameterize, and execute directly from the terminal. Built for DevOps and platform engineering workflows.

## Features

### Interactive TUI

- **Sidebar** — collapsible catalog tree with fuzzy search
- **Metadata panel** — runbook details, risk level, click-to-copy path
- **Output pane** — live streaming output with scroll, search, and text selection
- **Wizard** — field-by-field parameter input with per-field save control
- **Help overlay** — context-aware keybinding display (`?` key)

### Execution

- **Live streaming** — stdout/stderr streamed in real-time
- **Log persistence** — execution output saved to timestamped log files
- **Process control** — `ctrl+x` to stop running execution
- **Risk gates** — confirmation required for high/critical risk runbooks
- **Dry-run mode** — preview resolved command without executing

### MCP Server

AI agents can discover and execute runbooks via the [Model Context Protocol](https://modelcontextprotocol.io):

- **Tools** — each runbook exposed as an MCP tool with JSON Schema
- **Resources** — catalog listing and runbook details
- **Transports** — stdio (for Claude Code) and HTTP with gzip
- **Security** — sensitive params excluded from schema, loaded from local config
- **Progress** — real-time output streaming via MCP notifications

### CLI

- `dops` — launch the TUI
- `dops run <id>` — execute a runbook by ID
- `dops config set/get/unset/list` — manage configuration
- `dops catalog list/add/remove/install/update` — manage catalogs
- `dops mcp serve` — start MCP server
- `dops mcp tools` — list available MCP tools

## Install

### Homebrew

```bash
brew tap <owner>/tap
brew install dops
```

### Go

```bash
go install github.com/<owner>/dops-cli@latest
```

### Docker (MCP server)

```bash
# Mount your local catalogs and config into the container
docker run -i -v ~/.dops:/data/dops ghcr.io/<owner>/dops-cli:latest
```

### From source

```bash
git clone https://github.com/<owner>/dops-cli.git
cd dops-cli
make build
./bin/dops
```

## Quick Start

1. **Create a catalog** with runbook scripts:

```
~/.dops/catalogs/default/
├── hello-world/
│   ├── runbook.yaml
│   └── script.sh
└── check-health/
    ├── runbook.yaml
    └── script.sh
```

2. **Define a runbook** (`runbook.yaml`):

```yaml
name: check-health
version: 1.0.0
description: Runs health checks against a service endpoint
risk_level: medium
script: script.sh
parameters:
  - name: endpoint
    type: string
    required: true
    description: The endpoint to check
    scope: global
```

3. **Launch dops**:

```bash
dops
```

4. **Navigate** with arrow keys, **run** with Enter, **scroll** output with Up/Down, **search** with `/`.

## MCP Integration

### Claude Code

Add to `.claude/settings.json`:

```json
{
  "mcpServers": {
    "dops": {
      "command": "dops",
      "args": ["mcp", "serve"]
    }
  }
}
```

### Docker

```bash
# stdio transport — mount your catalogs/config
docker run -i -v ~/.dops:/data/dops ghcr.io/<owner>/dops-cli:latest

# HTTP transport with gzip
docker run -p 8080:8080 -v ~/.dops:/data/dops ghcr.io/<owner>/dops-cli:latest --transport http --port 8080
```

> **Note:** The container uses `DOPS_HOME=/data/dops`. Mount your local `~/.dops` directory to `/data/dops` to provide catalogs, config, and themes. You can also set `DOPS_HOME` to any path when running dops outside Docker.

## Keyboard Shortcuts

### Sidebar
| Key | Action |
|-----|--------|
| `↑↓` | Navigate runbooks |
| `←→` | Collapse/expand catalog |
| `Enter` | Run selected runbook |
| `/` | Search runbooks |

### Output
| Key | Action |
|-----|--------|
| `↑↓ j/k` | Scroll one line |
| `PgUp/PgDn` | Scroll one page |
| `h/l` | Scroll left/right |
| `/` | Search output |
| `n/N` | Next/prev match |
| `y` | Copy selection |

### Global
| Key | Action |
|-----|--------|
| `Tab` | Switch pane focus |
| `?` | Help overlay |
| `ctrl+x` | Stop execution |
| `ctrl+shift+p` | Command palette |
| `q` | Quit |

## Themes

dops uses a JSON-based theme system. Built-in themes: `tokyonight`, `tokyomidnight`.

Custom themes go in `~/.dops/themes/`:

```json
{
  "defs": {
    "bg": "#1a1b26",
    "fg": "#c0caf5",
    "blue": "#7aa2f7"
  },
  "theme": {
    "background": "bg",
    "text": "fg",
    "primary": "blue"
  }
}
```

## Development

```bash
make build       # Build binary
make test        # Run tests
make vet         # Go vet
make lint        # golangci-lint
make screenshots # Generate VHS screenshots
make docker      # Build Docker image
make ci          # Run CI checks (vet + test + build)
```

## License

MIT
