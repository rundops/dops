# Catalog Display Aliases

## Overview

Allow users to assign a custom display name to catalogs in `config.json`. By default, catalog names are derived from the repository basename (e.g., `dops-runbooks`). Display aliases let users show friendlier names in the TUI sidebar (e.g., `Production Ops`) without changing the underlying catalog identifier used for variable scoping and ID generation.

## Requirements

- Config must support an optional `display_name` field on each catalog entry
- When `display_name` is set, the TUI sidebar must show it instead of `name`
- The catalog's `name` field remains the canonical identifier for IDs, vault paths, and CLI commands
- `dops catalog list` must show both `name` and `display_name` (when set)
- `dops catalog update --display-name "Production Ops"` must set the display name
- `dops catalog add` must accept an optional `--display-name` flag
- Setting display name to empty string removes the alias

## Acceptance Criteria

- [ ] `Catalog` struct gains `DisplayName string` field with `json:"display_name,omitempty"`
- [ ] Sidebar renders `DisplayName` when non-empty, falls back to `Name`
- [ ] `dops catalog list` shows display name column
- [ ] `dops catalog add --display-name "My Ops" <url>` sets display name
- [ ] `dops catalog update <name> --display-name "New Name"` updates display name
- [ ] `dops catalog update <name> --display-name ""` clears display name
- [ ] Runbook IDs still use `Name` (not `DisplayName`) — `default.hello-world` unchanged
- [ ] Vault variable paths still use `Name` — no migration needed

## Error Cases

- Display name too long (>50 chars): rejected with error message
- Non-printable characters in display name: rejected with error message

## Out of Scope

- Renaming the canonical `name` field (breaking change, requires vault migration)
- Display names for individual runbooks (separate feature)
- Display name uniqueness enforcement (cosmetic only, not an identifier)
