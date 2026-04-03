# dops v0.12.0 — Specification

## Overview

v0.12.0 introduces execution history — a persistent audit trail of every
runbook execution across all interfaces (TUI, CLI, Web, MCP) with
size-based eviction to keep disk usage manageable.

## Features

### 1. Execution History + Audit Trail

**Status:** DONE
**Issue:** [#55](https://github.com/rundops/dops/issues/55)
**Plan:** [plans/2026-04-03-execution-history.md](../plans/2026-04-03-execution-history.md)

Persist every execution with structured metadata: runbook ID, catalog,
parameters, exit code, duration, output summary, and which interface
initiated it. Queryable via CLI (`dops history`), visible in web UI,
and exposed as an MCP resource.

**Storage lifecycle:**
- **Fresh** (< 7 days): `~/.dops/history/logs/{uuid}/datetime.log` — plain text
- **Compressed** (7–90 days): `.log.gz` — gzipped, still queryable via history
- **Expired** (> 90 days): record + log deleted entirely
- **Size cap** (50MB): oldest deleted regardless of age if directory exceeds limit
- UUID v4 execution IDs, temp log files deleted after persistent copy

#### Acceptance Criteria

- [x] `ExecutionRecord` domain type with UUID v4 ID, runbook, catalog, params, exit code, duration, status, interface
- [x] `ExecutionStore` interface with Record, Get, List, Delete
- [x] File-based store at `~/.dops/history/` (JSON records)
- [x] TUI records execution on completion
- [x] CLI (`dops run`) records execution on completion
- [x] Web API records execution on completion
- [x] MCP records execution on completion
- [x] `dops history` command lists recent executions
- [x] `dops history --runbook <id>` filters by runbook
- [x] `dops history --status failed` filters by status
- [x] Web UI history view (list + detail with log replay)
- [ ] MCP resource `dops://history` exposes execution records
- [x] Secret parameters masked in stored records
- [x] Size-based eviction (50MB default, oldest deleted first)
- [x] Logs persisted from /tmp to ~/.dops/history/logs/
- [x] Temp log files cleaned up after copy
- [x] `go test ./...` passes
- [x] Cross-platform: Windows + Linux build clean

---

### 2. Release Polish (required for every version)

**Status:** DONE

Every release must include documentation updates, demo data, and
release verification before tagging.

#### Acceptance Criteria

- [x] README.md updated with new features
- [x] docs/guides/ pages updated (getting-started with history section)
- [x] Demo mode has sample history data for the history view
- [ ] Screenshots/GIFs updated if UI changed
- [x] CLAUDE.md architecture reference updated if architecture changed
- [x] Changelog in GitHub release notes covers all changes
- [x] All open issues reviewed and closed if implemented
- [x] OG image and social meta tags for link previews
- [x] Favicon fixed for Safari
