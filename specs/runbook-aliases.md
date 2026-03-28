# Runbook ID Aliases

## Overview

Allow runbook authors to define one or more aliases in `runbook.yaml` that serve as alternative identifiers. Users can reference a runbook by its alias in `dops run <alias>`, the TUI sidebar, and MCP tools. This reduces friction when IDs are long or auto-generated from deep directory structures.

## Requirements

- Runbook YAML must support an optional `aliases` field (list of strings)
- Each alias must be globally unique across all loaded runbooks — duplicates are a load-time warning (first wins)
- Aliases must not collide with existing runbook IDs — IDs take precedence
- `dops run <alias>` must resolve to the aliased runbook
- Aliases must follow the same character rules as runbook IDs (lowercase alphanumeric, hyphens, dots)
- The sidebar must display the runbook's `name` field (unchanged) — aliases are for CLI/API lookup only
- MCP tool discovery must include aliases in tool metadata
- `dops catalog list --verbose` must show aliases

## Acceptance Criteria

- [ ] `runbook.yaml` accepts `aliases: [deploy, dp]` field
- [ ] `dops run deploy` resolves to the aliased runbook and executes it
- [ ] Duplicate alias across runbooks logs a warning and first-loaded wins
- [ ] Alias colliding with an existing ID is ignored with a warning
- [ ] `dops run <id>` still works (backward compatible)
- [ ] `dops catalog list -v` shows aliases column
- [ ] MCP tools include aliases in JSON schema metadata
- [ ] Invalid alias format (e.g., spaces, uppercase) rejected at load time with warning

## Error Cases

- Alias conflicts with existing runbook ID: warning logged, alias ignored
- Duplicate alias across runbooks: warning logged, first-loaded runbook keeps it
- Invalid alias format: warning logged, alias skipped
- Runbook not found by alias or ID: existing "runbook not found" error

## Out of Scope

- Alias management CLI commands (no `dops alias add/remove`)
- Alias search/autocomplete in the TUI sidebar filter
- Per-user alias overrides (aliases are defined by runbook authors only)
