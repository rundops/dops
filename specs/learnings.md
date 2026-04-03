# Learnings

Accumulated knowledge from developing dops-cli. Updated as discoveries are made.

## Codebase Patterns

### Runbook ID Generation
- If `id` is not set in `runbook.yaml`, the loader auto-generates it as `<catalogName>.<directoryName>`
- If `id` IS set in YAML, it takes precedence
- Validation enforces `<catalog>.<runbook>` format — exactly one dot, both segments non-empty
- Location: `internal/catalog/loader.go` lines 102-105

### Catalog Name
- `Catalog.Name` is used as the display name in sidebar and as map keys for scoped variable storage
- Currently derived from the repository basename or manually set during `catalog add`
- No concept of a separate "display name" vs "identifier" yet

### Saved Values / Vault
- Values stored in encrypted `vault.json`, NOT config.json
- Config.json `Vars` field has `json:"-"` — never serialized to config.json
- Scoping: `vars.global.<key>`, `vars.catalog.<cat>.<key>`, `vars.catalog.<cat>.runbooks.<rb>.<key>`
- Encryption: age X25519 + ChaCha20-Poly1305

### Wizard Prefill
- `prefill` map tracks which params have saved values
- `ShouldSkip()` returns true when ALL required params have values — skips wizard entirely
- Individual fields with saved values still show in the wizard (pre-filled but visible)
- `changed` detection: `!m.prefill[p.Name] || val != m.resolved[p.Name]`
- Save prompt only shown if value changed AND scope is not "local"

### Sidebar Filter
- Filter label: `"  " + filterLabel.Render("Filter: ") + filterInput.Render(m.searchQuery) + cursor`
- 2-space indent before "Filter: " label
- Rendered at bottom of sidebar View()

### Platform Build Tags
- `//go:build !windows` / `//go:build windows` for platform-specific files
- `proc_unix.go` / `proc_windows.go` for process management
- `shell_unix.go` / `shell_windows.go` for shell selection

### Theme System
- 20 embedded themes via `//go:embed`
- `rainbow` = random theme selection
- `github` = default theme
- JSON theme files in `internal/theme/themes/`

## v0.10.0 Discoveries

### Runbook Aliases
- Aliases are validated at load time via `buildAliasIndex()` — warnings logged for duplicates and ID collisions
- First-loaded runbook wins on alias conflict (deterministic based on catalog order in config)
- `dops run` resolution: try `FindByID` first (if valid format), then `FindByAlias`
- Removed the strict `ValidateRunbookID` gate from `cmd/run.go` to allow alias-only lookups

### Catalog Display Names
- `Label()` method on Catalog struct — returns DisplayName if set, else Name
- Sidebar uses `Label()` for display but all internal keys (collapsed, vars) still use `Name`
- `--display-name ""` on `catalog update` clears the display name (uses `cmd.Flags().Changed`)
- Update command logic needed restructuring: display-name-only updates skip git operations

### Skip Saved Fields
- `dops run` CLI has no wizard — it takes `--param` flags directly. `--prompt-all` flag not applicable there.
- TUI always creates the wizard (removed `ShouldSkip` bypass in v0.3.0 wizard redesign)
- When all fields are auto-applied, `Init()` returns `WizardSubmitMsg` immediately (no empty wizard shown)
- Ctrl+E clears the `skipped` map and sets `showAll = true` — works mid-wizard
- `goBack()` skips over auto-applied fields to find the previous user-editable field

### Web UI Research (Idea 3)
- **Vue 3 + Go embed**: Use `//go:embed all:dist` — the `all:` prefix is critical (without it, Go skips `_`-prefixed dirs like Vite's output)
- **SPA fallback**: `fs.Stat` check in Go handler; if file not found, serve `index.html`
- **shadcn-vue**: v1.x stable, 60+ components, backed by Reka UI v2 (headless, accessible). Copy-paste model, not npm dependency.
- **TailwindCSS v4**: CSS-first config (`@theme` directive), `@tailwindcss/vite` plugin, auto content detection. Replace `tailwindcss-animate` with `tw-animate-css`.
- **Streaming**: SSE over WebSocket — simpler, auto-reconnect, standard HTTP, `http.Flusher` in Go (~20 LOC)
- **ANSI rendering**: `ansi_up` (4KB, zero-dep) for client-side. `terminal-to-html` (Buildkite) for server-side alternative.
- **Theme mapping**: dops JSON tokens → CSS custom properties (`--primary`, `--background`, etc.) served as `<style>` block
- **Build pipeline**: Makefile (`make web`), `dist/` gitignored, `.gitkeep` placeholder for embed compile
- **Binary impact**: ~200KB-1MB added (3-15% of Go binary). Negligible.
- **Real-world precedent**: Gitea, AdGuard Home, Flipt all embed SPAs in Go binaries this way
