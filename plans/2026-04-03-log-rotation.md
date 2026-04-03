# Plan: Log Rotation

**Spec:** v0.12.0 Feature 2
**Issue:** [#64](https://github.com/rundops/dops/issues/64)
**Effort:** Half day
**Branch:** `feature/v0.12.0`

## Overview

Replace the current per-file gzip approach with a tiered log rotation
system: fresh individual logs → daily compressed archives → monthly
bundles. Configurable thresholds via `config.json`.

## Storage Layout

```
~/.dops/history/
├── logs/
│   ├── {uuid}/                    ← fresh (< 1 day)
│   │   └── 2026-04-03T14-30-00.log
│   ├── archives/
│   │   ├── 2026-04-01.tar.gz     ← daily archive
│   │   ├── 2026-04-02.tar.gz
│   │   └── 2026-03.tar.gz        ← monthly bundle
```

## TODO

### Phase 1: Config + Daily Archives (3 files)

- [ ] **1.1** Add `HistoryConfig` to `internal/domain/config.go`
  ```
  HistoryConfig {
    ArchiveAfterDays int  // default 1
    BundleAfterDays  int  // default 30
    BundleAfterRuns  int  // default 100
  }
  ```
  Add to `Config.History` field. Defaults in `EnsureDefaults()`.

- [ ] **1.2** Create `internal/history/archive.go` — daily archive logic
  - `ArchiveDailyLogs(logsDir, archivesDir, cutoff time.Time)` — scans
    UUID directories, groups logs by date, creates `{date}.tar.gz` per
    day, removes archived UUID dirs
  - Uses `archive/tar` + `compress/gzip` (stdlib, cross-platform)
  - Skip logs newer than cutoff

- [ ] **1.3** Create `internal/history/archive_test.go`
  - Test: logs older than cutoff archived into daily tar.gz
  - Test: logs newer than cutoff left alone
  - Test: multiple logs from same day grouped into one archive
  - Test: UUID directories removed after archiving
  - Test: existing archive not overwritten (append or skip)

### Phase 2: Monthly Bundles (2 files)

- [ ] **2.1** Add monthly bundling to `internal/history/archive.go`
  - `BundleMonthlyArchives(archivesDir, cutoff time.Time, maxRuns int)` —
    rolls daily archives older than cutoff into `{YYYY-MM}.tar.gz`
  - Also triggers when daily archive count exceeds `maxRuns`
  - Removes daily archives after bundling

- [ ] **2.2** Add monthly bundle tests to `internal/history/archive_test.go`
  - Test: daily archives older than cutoff bundled into monthly
  - Test: triggers on run count threshold
  - Test: monthly archive contains all daily archives for that month

### Phase 3: Integration (3 files)

- [ ] **3.1** Wire rotation into `FileExecutionStore.Record()`
  - Replace current `archiveOldLogs()` with new tiered rotation
  - Read thresholds from config (passed at construction)
  - Run daily archive pass, then monthly bundle pass

- [ ] **3.2** Add `dops history archive` subcommand — `cmd/history.go`
  - Manual trigger for rotation
  - Prints what was archived/bundled

- [ ] **3.3** Update `ReadLog` in `internal/history/store.go`
  - Search order: individual file → daily archive → monthly bundle
  - Extract single log from tar.gz by matching UUID in entry name

### Phase 4: Verification (0 new files)

- [ ] **4.1** `go test ./...` passes
- [ ] **4.2** `go vet ./...` clean
- [ ] **4.3** `GOOS=windows go build ./...` clean
- [ ] **4.4** Manual: run several executions, manually trigger archive, verify tiers

## Key Constraints

- stdlib only (`archive/tar` + `compress/gzip`) — no external deps
- Windows compatible (tar is cross-platform in Go stdlib)
- `ReadLog` must transparently search all tiers
- Non-destructive: if archiving fails mid-way, individual logs remain
- Config defaults must work without any `history` section in config.json

## Files

| File | Action |
|------|--------|
| `internal/domain/config.go` | Modify — add HistoryConfig |
| `internal/history/archive.go` | New — daily + monthly archiving |
| `internal/history/archive_test.go` | New — archive tests |
| `internal/history/store.go` | Modify — wire rotation, update ReadLog |
| `cmd/history.go` | Modify — add archive subcommand |
