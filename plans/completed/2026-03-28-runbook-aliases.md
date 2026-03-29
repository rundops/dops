# Plan: Runbook ID Aliases

**Spec:** [specs/runbook-aliases.md](../../specs/runbook-aliases.md)
**Status:** DONE

## Summary

Add an `aliases` field to `runbook.yaml` so users can reference runbooks by short names.

## Implementation Steps

### Step 1: Domain Model
- [x] Added `Aliases []string` field to `Runbook` struct with `yaml:"aliases,omitempty" json:"aliases,omitempty"`
- [x] Added `ValidateAlias(s string) error` — lowercase alphanumeric, hyphens, dots
- [x] Added tests for alias validation (valid, empty, uppercase, spaces, leading hyphen)

### Step 2: Loader — Alias Index
- [x] In `LoadAll()`, `buildAliasIndex()` builds alias-to-runbook map after loading
- [x] Duplicate alias: warning logged, first-loaded wins
- [x] Alias colliding with existing ID: warning logged, alias skipped
- [x] `FindByAlias(alias string)` resolves alias to runbook
- [x] Added `FindByAlias` to `CatalogLoader` interface
- [x] Tests: duplicate alias, alias-ID collision, valid resolution, not found

### Step 3: CLI `dops run` Resolution
- [x] In `cmd/run.go`, tries `FindByID` first (if valid ID format), then falls back to `FindByAlias`
- [x] Error message: "not found (tried ID and alias)"
- [x] Removed strict `ValidateRunbookID` gate — allows alias-only lookups

### Step 4: MCP Tool Metadata
- [x] `RunbookToDescription` includes `[aliases: deploy, dp]` when aliases present

### Step 5: Catalog List Verbose (deferred)
- Aliases are per-runbook, not per-catalog. `catalog list` shows catalogs.
- Aliases visible via MCP tool descriptions and runbook YAML.

## Files Changed

| File | Change |
|------|--------|
| `internal/domain/runbook.go` | `Aliases` field, `ValidateAlias()` |
| `internal/domain/runbook_test.go` | Alias validation tests |
| `internal/catalog/loader.go` | `aliasEntry`, `buildAliasIndex()`, `FindByAlias()`, interface update |
| `internal/catalog/loader_test.go` | Alias resolution, collision, and ID-collision tests |
| `cmd/run.go` | ID-then-alias fallback resolution |
| `internal/mcp/schema.go` | Alias metadata in tool description |
