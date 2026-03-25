---
layout: default
title: Creating Runbooks
nav_order: 3
parent: Guides
---

# Creating Runbooks

A runbook is a YAML definition paired with a POSIX shell script, organized in a catalog directory.

## Directory Structure

```
~/.dops/catalogs/<catalog>/<runbook-name>/
├── runbook.yaml    # Runbook definition
└── script.sh       # Automation script (POSIX sh)
```

---

## runbook.yaml Schema

```yaml
name: <runbook-name>          # Must match directory name
version: 1.0.0
description: Short description of what this runbook does
risk_level: low               # low | medium | high | critical
script: script.sh
parameters:
  - name: endpoint
    type: string              # See parameter types below
    required: true
    description: What this parameter does
    scope: global             # global | catalog | runbook
    default: ""               # Optional default value
    secret: false             # If true, masked in UI, excluded from MCP
    options: []               # Required for select and multi_select
```

---

## Parameter Types

| Type | Description | Validation |
|------|-------------|------------|
| `string` | Free text | — |
| `boolean` | Yes/No toggle | — |
| `integer` | Whole number (negative ok) | `strconv.Atoi` |
| `number` | Non-negative whole number (0+) | Must be >= 0 |
| `float` | Decimal number | `strconv.ParseFloat` |
| `select` | Single choice from options | Requires `options` list |
| `multi_select` | Multiple choices from options | Requires `options` list |
| `file_path` | File system path | — |
| `resource_id` | Resource identifier (ARN, URI) | — |

---

## Risk Levels

| Level | TUI Confirmation | MCP Confirmation |
|-------|-----------------|------------------|
| `low` | None | None |
| `medium` | None | None |
| `high` | y/N prompt | `_confirm_id` must match runbook ID |
| `critical` | Type runbook ID | `_confirm_word` must be "CONFIRM" |

---

## Parameter Scopes

Scopes control where parameter values are saved when the user chooses "Save for future runs."

| Scope | Saved To | Use When |
|-------|----------|----------|
| `global` | `vars.global.<name>` | Shared across all runbooks (API tokens, regions) |
| `catalog` | `vars.catalog.<cat>.<name>` | Shared within a catalog |
| `runbook` | `vars.catalog.<cat>.runbooks.<rb>.<name>` | Specific to one runbook |

---

## Script Template

Scripts should follow POSIX sh conventions for cross-platform compatibility (Linux + macOS).

```sh
#!/bin/sh
set -eu

# dops passes parameters as UPPERCASE environment variables.
# Parameter "endpoint" becomes $ENDPOINT
# Parameter "dry_run" becomes $DRY_RUN
ENDPOINT="${ENDPOINT:?endpoint is required}"
DRY_RUN="${DRY_RUN:-false}"

main() {
  echo "==> Stage 1/2: Validate"
  echo "    Checking ${ENDPOINT}..."

  echo ""
  echo "==> Stage 2/2: Execute"
  echo "    Running operation..."

  echo ""
  echo "Done"
}

main "$@"
```

### Shell Rules

1. Use `#!/bin/sh` — not `#!/bin/bash` (POSIX compatibility)
2. Use `set -eu` — not `set -euo pipefail` (pipefail is not POSIX)
3. Quote all variables: `"${var}"` not `$var`
4. Use `[ ]` not `[[ ]]` — POSIX test
5. Use `$(command)` not backticks
6. Stderr for errors: `echo "error" >&2`
7. Indent with 2 spaces, no tabs
8. Put `main()` at the bottom of the script
9. Use `command -v` not `which`
10. Use `printf` over `echo -e` for portability

### Output Conventions

```sh
# Stage headers
echo "==> Stage 1/3: Build"

# Indented details
echo "    Compiling source..."

# Success
echo "Done"

# Failure
echo "Build failed" >&2
```
