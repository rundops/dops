---
layout: default
title: Home
nav_order: 1
---

<pre class="hero-cat">  /\_/\
 ( =^.^= )</pre>

# do(ops) cli
{: .fs-9 }

a runbook toolkit for operators and AI agents.
{: .fs-6 .fw-300 }

[Get Started](/dops-cli/guides/getting-started){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub](https://github.com/jacobhuemmer/dops-cli){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## What is dops?

**dops** is an open-source terminal toolkit for browsing, executing, and managing operational runbooks. It works two ways:

- **For operators** — a full-screen TUI with sidebar navigation, parameter wizards, live streaming output, and risk confirmation gates.
- **For AI agents** — an MCP server that exposes runbooks as tools, so Claude and other AI agents can execute automation on your behalf.

Runbooks are simple YAML + shell scripts organized in catalogs. No proprietary DSL, no cloud dependency.

---

## Features

### Interactive TUI
- Sidebar with catalog tree, search, collapse/expand
- Metadata panel with click-to-copy paths
- Output pane with live streaming, scrollback, text selection
- Field-by-field wizard with parameter validation and persistence
- Risk confirmation gates (high = y/N, critical = type runbook ID)

### MCP Server
- Expose runbooks as tools for AI agents via Model Context Protocol
- Stdio and HTTP transports with gzip
- Sensitive parameters excluded from tool schemas
- Schema and style guide resources for runbook creation

### CLI
- `dops run <id>` — execute runbooks non-interactively
- `dops catalog install <url>` — install catalogs from git repos
- `dops config set/get/list` — manage configuration
- `dops mcp serve` — start the MCP server
- Shell completion for bash, zsh, fish, powershell

---

## Quick Install

```sh
# Homebrew
brew install jacobhuemmer/tap/dops

# Go
go install github.com/jacobhuemmer/dops-cli@latest

# From source
git clone https://github.com/jacobhuemmer/dops-cli.git
cd dops-cli && make install
```

---

## Support

If you find dops useful, consider [buying me a coffee](https://buymeacoffee.com/jacobhuemmer).
