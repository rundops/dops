# dops v0.12.0 — Specification

## Overview

v0.12.0 introduces execution history — a persistent audit trail of every
runbook execution across all interfaces (TUI, CLI, Web, MCP).

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
