# dops-cli — Feature TODO

## v0.11.0 — MCP Completeness + Web UI Polish

**Spec:** [specs/VERSION_0_11_0.md](../specs/VERSION_0_11_0.md)
**Branch:** `feature/v0.11.0`

- [x] MCP tool schema defaults (pass resolved vars to `RunbookToInputSchema`)
- [ ] MCP execution progress notifications (wire `OnProgress` callback)
- [ ] MCP create-runbook prompt scaffolding (dynamic parameter templates)
- [x] Catalog switcher tab bar (TUI + Web UI)
- [ ] MCP skills — injectable context for AI agents (skill.md + MCP prompts)
- [ ] Web UI: execution view — remove duplicate status, modernize header

## Completed

### 2026-03-23 — Output Pane Refinement
- [x] ANSI handling with charmbracelet/x/ansi
- [x] Scrollbar with proportional thumb (primary color)
- [x] Smart auto-scroll (atBottom flag)
- [x] Horizontal scrolling (h/l keys)
- [x] Live streaming via tea.Program.Send()
- [x] Buffered log file persistence
- [x] Three-section layout (header/log/footer) inside persistent outer border
- [x] Focus management (Tab, hover-to-focus, click-to-focus)
- [x] Terminal background from theme
- [x] Layout spacing and padding

### 2026-03-24 — Features & Selection
- [x] Risk confirmation gates (High=y/N, Critical=type ID)
- [x] Process management (ctrl+x stop with SIGKILL)
- [x] Help overlay (? key, context-aware)
- [x] Catalog management CLI (list/add/remove/install/update with --ref)
- [x] Dry-run mode
- [x] Text selection with highlight (terminal-absolute coordinates, view-relative rendering)
- [x] Clipboard copy (auto on release, y key, OSC 52 fallback)
- [x] "Copied to Clipboard!" floating border badge
- [x] Click-to-copy green flash on header/footer
- [x] Search UI ("Search:" prompt, navigation hints)

### 2026-03-24 — Wizard Redesign
- [x] Custom wizard replacing Huh form (field-by-field progression)
- [x] Left accent bar + panel background overlay style
- [x] Per-type rendering (select, multi_select, boolean, text, password)
- [x] Context-sensitive footer hints
- [x] Parameter persistence: "Save for future runs?" per-field (default No)
- [x] Pre-fill saved values, sensitive fields show bullet dots
- [x] New parameter types (multi_select, file_path, resource_id)
- [x] Command header only shows overridden params (not config defaults)

### 2026-03-24 — MCP Server
- [x] MCP server with stdio and HTTP transport
- [x] Each runbook as an MCP tool with JSON Schema input
- [x] Catalog listing and detail as MCP resources
- [x] Smart output truncation (last 50 lines + metadata)
- [x] HTTP gzip compression middleware
- [x] Risk confirmation via synthetic params
- [x] CLI: dops mcp serve / dops mcp tools

### 2026-03-25 — Vault Encryption
- [x] Encrypted parameter storage using age (X25519/ChaCha20-Poly1305)
- [x] Migration from plaintext config.json to encrypted vault.json
- [x] Tamper detection and 0600 file permissions

### 2026-03-28 — v0.10.0 Features
- [x] Runbook ID aliases (`aliases` field in runbook.yaml, CLI fallback resolution)
- [x] Catalog display names (`--display-name` flag, `Label()` method in sidebar)
- [x] Skip saved fields in wizard (auto-apply, Ctrl+E reveal, auto-submit when all saved)
- [x] Sidebar filter label alignment fix
- [x] Web UI (`dops open`): Vue 3 + Tailwind v4 SPA embedded in Go binary
  - [x] Sidebar with catalog tree, search, risk badges
  - [x] Parameter form (text, select, multi-select, boolean, secret)
  - [x] SSE execution streaming with ANSI rendering
  - [x] Risk confirmation dialogs
  - [x] Live theme switching (20 built-in themes)
  - [x] Responsive layout with mobile drawer
  - [x] Keyboard navigation (/, arrows, Enter, Escape)
  - [x] Error boundary and fallback UI
  - [x] Toast notifications
