---
title: MCP / AI Agents
---

# MCP / AI Agents

dops includes an [MCP](https://modelcontextprotocol.io/) server that exposes runbooks as tools for AI agents like Claude.

---

## Quick Setup with Claude Code

Add to your Claude Code MCP config:

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

Claude can now discover and execute your runbooks as tools.

---

## Docker

Run the MCP server in a container:

```sh
docker run -i --rm \
  -v ~/.dops:/data/dops \
  ghcr.io/rundops/dops:latest
```

The container uses `DOPS_HOME=/data/dops` and runs `dops mcp serve --transport stdio` by default.

For HTTP transport:

```sh
docker run -d --rm \
  -v ~/.dops:/data/dops \
  -p 8080:8080 \
  ghcr.io/rundops/dops:latest \
  dops mcp serve --transport http --port 8080
```

---

## Transports

| Transport | Flag | Use Case |
|-----------|------|----------|
| `stdio` | `--transport stdio` | Claude Code, local AI agents |
| `http` | `--transport http --port 8080` | Remote agents, web integrations |

HTTP transport includes gzip compression and supports SSE for streaming.

---

## Risk Control

Limit which runbooks are exposed by risk level:

```sh
dops mcp serve --allow-risk medium
```

This hides `high` and `critical` runbooks from AI agents.

---

## How It Works

### Tools

Each runbook becomes an MCP tool. The tool name is the runbook ID (e.g., `infra.health-check`). Parameters are mapped to the tool's JSON Schema input.

- **Sensitive parameters** are excluded from the schema and noted in the tool description.
- **High-risk** runbooks require a `_confirm_id` parameter matching the runbook ID.
- **Critical-risk** runbooks require a `_confirm_word` parameter set to `CONFIRM`.

### Resources

The MCP server exposes two schema resources:

| URI | Description |
|-----|-------------|
| `dops://schema/runbook` | Full runbook.yaml schema reference |
| `dops://schema/shell-style` | POSIX shell scripting guide |

Per-runbook resources are also available at `dops://runbook/<id>`.

### Prompts

The `create-runbook` prompt guides AI agents through creating a new runbook with the correct YAML schema and shell template.

---

## List Available Tools

```sh
dops mcp tools
dops mcp tools --allow-risk medium
```
