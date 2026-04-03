# Plan: Execution History + Audit Trail

**Spec:** v0.12.0 Feature 1
**Issue:** [#55](https://github.com/rundops/dops/issues/55)
**Effort:** 1-2 days
**Branch:** `feature/v0.12.0`

## Overview

Persist every runbook execution as a structured record. Today, logs go to
`/tmp` (ephemeral) and no metadata is tracked. After this feature, every
execution across TUI, CLI, Web, and MCP is recorded to `~/.dops/history/`
with full metadata.

## Storage Design

Records stored as individual JSON files in `~/.dops/history/`:
```
~/.dops/history/
├── 2026-04-03T14-30-00-default.hello-world.json
├── 2026-04-03T14-31-22-infra.scale-deployment.json
└── ...
```

Each file is a single `ExecutionRecord` JSON object. File-per-record
avoids locking issues and makes cleanup trivial. Retention: keep last
N records (default 500), oldest deleted on new write.

## TODO

Work through each item sequentially. Mark done and commit before moving
to the next.

### Phase 1: Domain + Store (3 files)

- [x] **1.1** Create `internal/domain/execution.go` — `ExecutionRecord` struct
  ```
  Fields: ID, RunbookID, RunbookName, CatalogName, Parameters (map, secrets masked),
  Status (running/success/failed/cancelled), ExitCode, StartTime, EndTime,
  Duration, OutputLines, OutputSummary, LogPath, Interface (tui/cli/web/mcp)
  ```

- [x] **1.2** Create `internal/domain/execution_test.go` — test `ExecutionRecord` helpers
  - Test `NewExecutionRecord()` sets ID, StartTime, Status=running
  - Test `Complete()` sets EndTime, Duration, Status
  - Test `MaskSecrets()` replaces secret param values with `"****"`

- [x] **1.3** Create `internal/history/store.go` — `FileExecutionStore`
  ```
  Interface: ExecutionStore {
    Record(record *ExecutionRecord) error
    Get(id string) (*ExecutionRecord, error)
    List(opts ListOpts) ([]*ExecutionRecord, error)
    Delete(id string) error
  }
  ListOpts: RunbookID, Status, Limit (default 50), Offset
  ```
  - Writes JSON to `~/.dops/history/{timestamp}-{runbook-id}.json`
  - List reads directory, parses filenames for sorting, loads matching records
  - Enforces retention (delete oldest when count > max)

- [x] **1.4** Create `internal/history/store_test.go` — test FileExecutionStore
  - Test Record + Get roundtrip
  - Test List returns records sorted by time (newest first)
  - Test List with RunbookID filter
  - Test List with Status filter
  - Test List with Limit/Offset
  - Test retention enforcement (record N+1 deletes oldest)
  - Test Delete removes file

### Phase 2: Integration — TUI + CLI (4 files)

- [x] **2.1** Add `ExecutionStore` to `cmd/deps.go` — `appDeps` struct
  - Create `FileExecutionStore` in `loadDeps()`
  - Pass history dir as `filepath.Join(dopsDir, "history")`

- [x] **2.2** Wire TUI execution recording — `internal/tui/exec.go`
  - In `startExecution()`: create `ExecutionRecord` (status=running)
  - In `executionDoneMsg` handler: call `Complete()`, `Record()`
  - Pass `ExecutionStore` via `AppDeps`
  - Mask secret params before recording

- [x] **2.3** Wire CLI execution recording — `cmd/run.go`
  - After `executeScript()`: create record, complete, record
  - Load `ExecutionStore` in run command setup
  - Mask secret params before recording

- [x] **2.4** Add `dops history` command — `cmd/history.go`
  - `dops history` — list last 20 executions (table format)
  - `dops history --runbook <id>` — filter by runbook
  - `dops history --status failed` — filter by status
  - `dops history --limit 50` — control result count
  - Output columns: Time, Runbook, Status, Duration, Exit Code

### Phase 3: Integration — Web + MCP (3 files)

- [x] **3.1** Wire Web API execution recording — `internal/web/api.go`
  - Pass `ExecutionStore` to web server deps
  - Record on execution completion
  - Add `GET /api/history` endpoint (JSON list)
  - Add `GET /api/history/{id}` endpoint (JSON detail)

- [x] **3.2** Wire MCP execution recording — `internal/mcp/tools.go`
  - Pass `ExecutionStore` in `ToolCallRequest` or `ServerConfig`
  - Record on execution completion in `HandleToolCall`
  - Add MCP resource `dops://history` (JSON list of recent executions)

- [x] **3.3** Add Web UI history view — `web/src/views/HistoryView.vue`
  - Route: `/history`
  - Table: time, runbook, status badge, duration, exit code
  - Click row → detail view with params and output summary
  - Filter by runbook and status
  - Link from sidebar or header

### Phase 4: Verification (0 new files)

- [x] **4.1** Run full test suite: `go test ./...` — 24 packages pass
- [x] **4.2** Run `go vet ./...` — clean
- [x] **4.3** TypeScript check: `cd web && npx vue-tsc --noEmit` — clean
- [x] **4.4** Manual test: CLI execution records to `~/.dops/history/` ✓
- [x] **4.5** Manual test: `dops history` shows executions with filters ✓
- [ ] **4.6** Manual test: web UI history view loads and filters

## Key Constraints

- Secret parameters MUST be masked before writing to disk
- Records are plain JSON (not encrypted — no secrets in values after masking)
- Retention enforced on write, not background cleanup
- File-per-record avoids locking (safe for concurrent TUI + Web)
- No breaking changes to existing execution behavior
- Log files continue to go to temp dir (unchanged)

## Files

| File | Action |
|------|--------|
| `internal/domain/execution.go` | New — ExecutionRecord type |
| `internal/domain/execution_test.go` | New — domain tests |
| `internal/history/store.go` | New — FileExecutionStore |
| `internal/history/store_test.go` | New — store tests |
| `cmd/deps.go` | Modify — add ExecutionStore to appDeps |
| `cmd/history.go` | New — dops history command |
| `cmd/run.go` | Modify — record CLI executions |
| `internal/tui/exec.go` | Modify — record TUI executions |
| `internal/tui/app.go` | Modify — AppDeps gets ExecutionStore |
| `internal/web/api.go` | Modify — record + history endpoints |
| `internal/mcp/tools.go` | Modify — record MCP executions |
| `internal/mcp/server.go` | Modify — history resource |
| `web/src/views/HistoryView.vue` | New — history UI |
| `web/src/main.ts` | Modify — add /history route |
| `web/src/lib/api.ts` | Modify — add history API calls |
