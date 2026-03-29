# Plan: Catalog Display Aliases

**Spec:** [specs/catalog-aliases.md](../../specs/catalog-aliases.md)
**Status:** DONE

## Summary

Add a `display_name` field to the Catalog struct so users can show friendly names in the TUI sidebar without changing the canonical catalog name used for IDs and vault paths.

## Implementation Steps

### Step 1: Domain Model
- [x] Add `DisplayName string` field to `Catalog` struct with `json:"display_name,omitempty"`
- [x] Add `Label() string` method: returns `DisplayName` if non-empty, else `Name`
- [x] Add `ValidateDisplayName()`: max 50 chars, printable characters only
- [x] Add tests for `Label()`, `RunbookRoot()`, and `ValidateDisplayName`

### Step 2: Sidebar Rendering
- [x] In sidebar `buildLines()`, replaced `e.catalog.Name` with `e.catalog.Label()` for header display
- [x] Verified collapsed state still keys on `Name` (not `DisplayName`) — all `m.collapsed[e.catalog.Name]` unchanged

### Step 3: CLI — Catalog Commands
- [x] `dops catalog add` — added `--display-name` flag with validation
- [x] `dops catalog install` — added `--display-name` flag with validation (before clone)
- [x] `dops catalog update` — added `--display-name` flag (empty to clear, uses `cmd.Flags().Changed`)
- [x] `dops catalog list` — added DISPLAY NAME column

### Step 4: Config Persistence
- [x] `DisplayName` uses `omitempty` — backward compatible with existing configs
- [x] Build passes, all existing tests pass

## Files Changed

| File | Change |
|------|--------|
| `internal/domain/config.go` | `DisplayName` field, `Label()`, `ValidateDisplayName()` |
| `internal/domain/config_test.go` | New: Label, RunbookRoot, ValidateDisplayName tests |
| `internal/tui/sidebar/model.go` | `e.catalog.Label()` for display |
| `cmd/catalog.go` | `--display-name` flag on add/install/update, list column |
