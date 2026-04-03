# dops v0.12.0 — Specification

## Overview

v0.12.0 introduces execution history — a persistent audit trail of every
runbook execution across all interfaces (TUI, CLI, Web, MCP) — plus log
rotation to keep disk usage manageable over time.

## Features

### 1. Execution History + Audit Trail

**Status:** TODO
**Issue:** [#55](https://github.com/rundops/dops/issues/55)
**Plan:** [plans/2026-04-03-execution-history.md](../plans/2026-04-03-execution-history.md)

Persist every execution with structured metadata: runbook ID, catalog,
parameters, exit code, duration, output summary, and which interface
initiated it. Queryable via CLI (`dops history`), visible in web UI,
and exposed as an MCP resource.

#### Acceptance Criteria

- [ ] `ExecutionRecord` domain type with ID, runbook, catalog, params, exit code, duration, status, interface
- [ ] `ExecutionStore` interface with Record, Get, List, Delete
- [ ] File-based store at `~/.dops/history/` (JSON records)
- [ ] TUI records execution on completion
- [ ] CLI (`dops run`) records execution on completion
- [ ] Web API records execution on completion
- [ ] MCP records execution on completion
- [ ] `dops history` command lists recent executions
- [ ] `dops history --runbook <id>` filters by runbook
- [ ] `dops history --status failed` filters by status
- [ ] Web UI history view (list + detail)
- [ ] MCP resource `dops://history` exposes execution records
- [ ] Secret parameters masked in stored records
- [ ] Retention policy (configurable max records, default 500)
- [ ] `go test ./...` passes

---

### 2. Log Rotation

**Status:** TODO
**Issue:** [#64](https://github.com/rundops/dops/issues/64)
**Plan:** [plans/2026-04-03-log-rotation.md](../plans/2026-04-03-log-rotation.md)

Bundle old execution logs into compressed daily archives. Configurable
thresholds for when to archive and when to roll daily archives into
monthly bundles.

**Storage tiers:**
- **Fresh** (< `archive_after_days`): individual plain `.log` files
- **Daily archive** (≥ `archive_after_days`): logs bundled into `2026-04-03.tar.gz`
- **Monthly bundle** (≥ `bundle_after_days` or `bundle_after_runs`): daily archives rolled into `2026-04.tar.gz`

**Configuration** in `config.json`:
```json
{
  "history": {
    "archive_after_days": 1,
    "bundle_after_days": 30,
    "bundle_after_runs": 100
  }
}
```

#### Acceptance Criteria

- [ ] Logs older than `archive_after_days` compressed into daily `.tar.gz`
- [ ] Daily archives older than `bundle_after_days` rolled into monthly `.tar.gz`
- [ ] Bundling also triggers when run count exceeds `bundle_after_runs`
- [ ] `dops history archive` command for manual rotation
- [ ] `ReadLog` reads from individual files, daily archives, and monthly bundles
- [ ] Config defaults: archive after 1 day, bundle after 30 days / 100 runs
- [ ] Cross-archive search via `dops history` still works
- [ ] Windows compatible (no Unix-specific tar operations)
- [ ] `go test ./...` passes
