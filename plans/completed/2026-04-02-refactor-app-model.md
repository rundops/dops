# Refactor: Decompose App God Struct

**Status:** TODO
**Effort:** 1-2 days
**Risk:** Medium — touches core TUI orchestration
**Branch:** `refactor/app-model-decomposition`

## Problem

`internal/tui/app.go` is ~1350 lines with 40+ methods on a single `App` struct. It handles initialization, message routing, execution management, overlay lifecycle, layout computation, rendering, mouse handling, text selection, and clipboard operations. Any non-trivial TUI change requires understanding the full file.

## Current Structure

```
App struct (17 fields)
├── Sub-models: sidebar, output, wizard, palette, confirm
├── Domain state: selected, selCat
├── Execution state: execRunning, cancelExec
├── UI state: state, focus, copiedFlash, updateAvailable
├── Layout state: width, height
└── Dependencies: deps (AppDeps)
```

**Methods by concern:**
- Lifecycle & routing: 4 methods (~160 lines)
- Execution management: 6 methods (~130 lines)
- Overlay lifecycle: 4 methods (~70 lines)
- Layout & rendering: 9 methods (~250 lines)
- Mouse handling: 6 methods (~100 lines)
- Text selection & clipboard: 4 methods (~100 lines)
- Test getters: 8 methods (~20 lines)

## Extraction Plan

### Phase 1: Extract Execution Manager

**New file:** `internal/tui/exec.go` (same package, not a new package)

Move to an `execManager` struct:
- `startExecution(rb, cat, params)`
- `runStreaming(ctx, prog, scriptPath, env, logPath)`
- `runBlocking(ctx, scriptPath, env, logPath)`
- `emitDryRun(scriptPath, params)`
- `createLogFile(catalogName, runbookName)`
- `buildEnv(params)`

Fields to move: `cancelExec`, `execRunning`, plus references to `deps.Runner`, `deps.LogWriter`, `deps.DryRun`.

**Why first:** Self-contained concern with clear inputs/outputs. The execution methods don't read layout state or render anything.

### Phase 2: Extract Mouse & Selection Handler

**New file:** `internal/tui/mouse.go` (same package)

Move to a `mouseHandler` struct or standalone functions:
- `focusTargetFromMouse(msg)`
- `translateMouseForSidebar(msg)`
- `translateMouseForOutput(msg)`
- `sidebarBounds(mx, my)`
- `mouseCoords(msg)`
- `isMouseMsg(msg)` / `isMouseClick(msg)`
- `outputPaneBounds()`
- `handleMetadataClick(msg)`
- `handleOutputClick(msg)`

**Why second:** Pure geometry calculations that only need layout dimensions. Mostly stateless.

### Phase 3: Extract Selection Highlighter

**New file:** `internal/tui/selection.go` (same package)

Move:
- `applySelectionHighlight(content, sel, styles, bounds)`
- `highlightLine(...)`
- `extractSelectionFromView()`
- `injectBorderBadge(rendered, text, styles)`
- `selectionBounds` struct

**Why third:** Post-processing functions that operate on rendered strings. No state dependencies.

### Phase 4: Consolidate Overlay Lifecycle

**New file:** `internal/tui/overlays.go` (same package)

Move:
- `openWizard()`
- `openConfirm(rb, cat, params)`
- `openPalette()`
- `resolveVars()`
- `view*Overlay()` methods

This keeps overlay open/close/render co-located.

## What Stays in app.go

After extraction, `app.go` should contain:
- `App` struct definition with embedded/referenced sub-structs
- `NewApp` / `NewAppWithDeps` constructors
- `Init()`, `Update()`, `View()` — the BubbleTea contract
- `handleKeyPress()`, `handleAppMessage()`, `routeToComponent()` — top-level routing
- `resizeAll()`, `computeLayout()` — layout orchestration
- `viewNormal()` — main view composition
- Test getters

Target: ~500-600 lines in app.go, ~800 lines distributed across 4 new files.

## Constraints

- [ ] All new files stay in `internal/tui` package — no new packages
- [ ] BubbleTea message flow unchanged — `Update` still the single entry point
- [ ] No behavior changes — pure structural refactor
- [ ] Tests pass after each phase
- [ ] Each phase is one commit

## Verification

- `go build ./...` after each phase
- `go test ./internal/tui/...` after each phase
- Manual smoke test after Phase 4: launch TUI, navigate, execute a runbook, use wizard, test mouse selection
