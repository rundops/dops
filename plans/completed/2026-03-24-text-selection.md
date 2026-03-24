# Text Selection Plan

## Date: 2026-03-24

## Context

Implemented remaining feature items and refined the text selection/clipboard system through iterative testing.

## Features Implemented

### Risk Confirmation Gates
- Low/Medium risk: execute immediately, no confirmation
- High risk: y/N confirmation prompt
- Critical risk: must type the runbook ID to confirm
- Implemented as `stateConfirm` overlay with `confirm/model.go`
- Matches standard UX patterns (AWS, Terraform, GitHub)

### Process Management (ctrl+x)
- Cancellable context with `context.WithCancel` in `startExecution`
- ctrl+x calls cancel function when execution is running
- Executor uses `SysProcAttr{Setpgid: true}` for process group kill
- `cmd.Cancel` sends SIGKILL to entire process group
- `WaitDelay = 2s` for graceful shutdown
- `execRunning` flag tracks state, cleared on `executionDoneMsg`

### Help Overlay (? key)
- Context-aware keybinding display based on focused pane
- Sidebar keys: navigate, collapse/expand, run, search
- Output keys: scroll, horizontal scroll, search, match navigation
- Global keys: tab, palette, ctrl+x stop, quit
- Centered overlay with bordered box, press ? or esc to close

### Catalog Management CLI
- `dops catalog list` — shows name, path, URL, active, risk policy
- `dops catalog add <path>` — add local directory as catalog
- `dops catalog remove <name>` — remove catalog from config
- `dops catalog install <url>` — git clone + add to config
  - `--name` flag for custom name (defaults to repo basename)
  - `--ref` flag for git tag/branch/commit versioning
- `dops catalog update <name>` — git pull for git-installed catalogs

### Dry-Run Mode
- `AppDeps.DryRun` flag skips execution
- Shows resolved script path and environment variables in output pane
- "[DRY RUN]" prefix in output lines

### Text Selection & Clipboard
- Click/drag in output pane to select text
- Terminal-absolute coordinates stored directly from mouse events
- Selection highlight applied as post-render pass on full terminal view
- `ansi.Cut` used to split styled lines without breaking ANSI codes
- `ansi.Strip` extracts plain text for clipboard
- Highlight confined to output pane bounds (excludes sidebar, scrollbar, border, padding)
- Clipboard extraction also confined to same bounds
- `y` key copies selection to clipboard
- Mouse release auto-copies to clipboard
- "Copied to Clipboard!" badge floats in output border top-right for 1.5s
- Selection highlight clears when badge expires
- OSC 52 clipboard fallback for SSH/remote terminals

### Copy-to-Clipboard UX
- Unified floating border badge for all copy actions (header click, footer click, text selection)
- Header/footer click-to-copy: text briefly flashes green (success color) for 1.5s
- Command and log path always stay visible (no inline text replacement)
- Badge injected into top border row: `╭─── Copied to Clipboard! ──╮`

### Search UI Improvements
- Search prompt shows "Search: <query>▎" instead of "/<query>"
- Navigation mode shows: `<query> [x/y]  n/N next/prev  esc clear`
- Removed "»" prefix from matching lines

## Architecture Decisions

### Selection Coordinate System
- **Terminal-absolute coordinates** stored in selection (no translation)
- **Rendered view is source of truth** for character positions
- Highlight and extraction both use `viewNormal().Content` split into lines
- `outputPaneBounds()` computes exact text area in terminal-absolute space
- Same coordinate space as `handleOutputClick` for behavioral parity

### Mouse Event Routing
- `MouseModeCellMotion` for all views
- Hover over output pane auto-focuses it (motion events)
- Click in sidebar steals focus back (click events only)
- Mouse wheel events bypass coordinate translation
- Terminal's native selection overlay during drag is expected/unavoidable

## Files Modified

| File | Changes |
|---|---|
| `internal/tui/confirm/model.go` | New: confirmation overlay for high/critical risk |
| `internal/tui/confirm/messages.go` | New: ConfirmAcceptMsg, ConfirmCancelMsg |
| `internal/tui/help/view.go` | New: context-aware keybinding help overlay |
| `internal/clipboard/clipboard.go` | New: OSC 52 clipboard fallback |
| `internal/tui/output/selection.go` | New: TextSelection struct with Bounds, ExtractText |
| `cmd/catalog.go` | New: catalog list/add/remove/install/update CLI |
| `internal/tui/app.go` | Risk gates, ctrl+x, help, dry-run, selection highlight, copy extraction, output pane bounds |
| `internal/tui/output/model.go` | Mouse selection handling, copy flash, search UI |
| `internal/tui/output/messages.go` | SelectionCompleteMsg, CopyFlashExpiredMsg, CopiedRegionFlashMsg |
| `internal/tui/footer/view.go` | StateConfirm, StateHelp footer bindings |
| `internal/executor/script.go` | Process group kill, WaitDelay |
| `cmd/root.go` | Catalog subcommand registration |
