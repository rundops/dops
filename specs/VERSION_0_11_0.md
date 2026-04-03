# dops v0.11.0 — Specification

## Overview

v0.11.0 focuses on MCP server completeness and Web UI polish. Three MCP
enhancements fill gaps in the tool schema and execution experience. One Web
UI fix removes duplicate status display and modernizes the execution view
header.

**Manual test plan:** [plans/2026-04-02-v0.11.0-manual-test-plan.md](../plans/2026-04-02-v0.11.0-manual-test-plan.md)

## Features

### 1. MCP Tool Schema Defaults

**Status:** DONE

Populate pre-filled parameter defaults in MCP tool schemas so AI clients
see saved values before invoking a tool.

**Current behavior:** `registerTools()` passes an empty map to
`RunbookToInputSchema()`. Tool schemas advertise parameters but never show
saved defaults.

**Target behavior:** Resolve saved vars from the vault (global → catalog →
runbook scope) and pass them to `RunbookToInputSchema()` so `default`
fields appear in the JSON Schema output.

**File:** `internal/mcp/server.go` (~line 74)

#### Acceptance Criteria

- [ ] `RunbookToInputSchema` receives resolved vars for each runbook
- [ ] JSON Schema `default` field is populated for parameters with saved values
- [ ] Parameters without saved values omit the `default` field
- [ ] Secret parameters do NOT expose their saved value in the schema
- [ ] Existing MCP tool tests updated to cover default population
- [ ] `go test ./internal/mcp/...` passes

---

### 2. MCP Execution Progress Notifications

**Status:** DONE

Wire real-time progress notifications during MCP tool execution so AI
clients receive streaming updates instead of silent execution followed by a
final result.

**Current behavior:** `HandleToolCall` receives `OnProgress: nil`. The
execution runs to completion and returns a single `ToolResult`.

**Target behavior:** Provide an `OnProgress` callback that emits MCP
progress notifications as execution output lines arrive. Notifications
include line count and a truncated tail of recent output.

**File:** `internal/mcp/server.go` (~line 106)

#### Acceptance Criteria

- [ ] `OnProgress` callback is wired in `makeToolHandler`
- [ ] Progress notifications include line count and last N lines of output
- [ ] Notifications are rate-limited (no more than 1 per second)
- [ ] Nil `OnProgress` still works (no regression for non-streaming clients)
- [ ] Progress works for both stdio and HTTP transports
- [ ] Test covers progress callback invocation during execution
- [ ] `go test ./internal/mcp/...` passes

---

### 3. MCP Create-Runbook Prompt Scaffolding

**Status:** DONE

Replace placeholder TODOs in the `create-runbook` MCP prompt templates with
dynamic parameter scaffolding based on the runbook's declared parameters.

**Current behavior:** Shell and PowerShell templates contain
`# TODO: Add parameter variables` and `# TODO: Implement` comments. Users
must manually write variable extraction and validation.

**Target behavior:** Templates generate parameter variable extraction lines
from the runbook's parameter list. Required params use strict validation
(`${VAR:?msg}` in bash, `throw` in PowerShell). Optional params use
fallback defaults. The `# TODO: Implement` block is replaced with a
descriptive placeholder that references the parameter variables.

**File:** `internal/mcp/prompts.go` (~lines 230-250)

#### Acceptance Criteria

- [ ] Bash template generates `VAR="${VAR:?required}"` for required params
- [ ] Bash template generates `VAR="${VAR:-default}"` for optional params with defaults
- [ ] PowerShell template generates equivalent `$env:VAR` extraction
- [ ] Secret parameters include a comment noting the value is masked
- [ ] Template body references generated variables in a placeholder block
- [ ] Runbooks with zero parameters produce a clean template (no empty variable section)
- [ ] Test covers template generation for mixed required/optional/secret params
- [ ] `go test ./internal/mcp/...` passes

---

### 4. Catalog Switcher (Tab Bar)

**Status:** REVERTED

Add a tab bar above the sidebar that lets users switch between catalogs,
showing only the active catalog's runbooks. Inspired by television CLI's
channel switching UX.

**Current behavior:** All catalogs shown simultaneously as a flat
collapsible tree. No concept of "active catalog."

**Target behavior:**
- Horizontal tab bar above the runbook list: `All | infra | demo | default`
- `All` tab shows the existing collapsible tree (backward compatible)
- Single catalog tabs show only that catalog's runbooks (names only, no headers)
- Active tab highlighted with primary color
- `Ctrl+H` / `Ctrl+L` cycle catalogs from anywhere in sidebar
- `←` / `→` also cycle when on a specific catalog tab
- Search filters within the active catalog only
- Tab bar hidden when only one catalog exists
- Web UI mirrors the TUI tab bar behavior

**Files:**
- `internal/tui/sidebar/model.go` — tab state, filtering, view, keyboard handling
- `internal/tui/sidebar/messages.go` — `CatalogSwitchedMsg`
- `internal/tui/app.go` — route `CatalogSwitchedMsg`, clear selection on switch
- `internal/tui/help/view.go` — add `Ctrl+H/L` shortcut documentation
- `web/src/components/Sidebar.vue` — tab bar UI

#### Acceptance Criteria

- [ ] Tab bar renders above sidebar when 2+ catalogs exist
- [ ] Tab bar hidden when only 1 catalog
- [ ] `All` tab shows existing collapsible tree view (no behavior change)
- [ ] Single catalog tab shows flat runbook list (names only, no headers)
- [ ] `Ctrl+H` / `Ctrl+L` cycles catalogs
- [ ] `←` / `→` cycles catalogs when on a specific catalog tab
- [ ] Search filters within active catalog only
- [ ] Switching catalog resets cursor and clears search
- [ ] Selection clears if selected runbook not in new catalog
- [ ] Web UI tab bar mirrors TUI behavior
- [ ] `dops run <id>` CLI unaffected
- [ ] Help overlay shows new shortcuts

---

### 5. MCP Skills — Injectable Context for AI Agents

**Status:** DONE

Add skills as a new content type alongside runbooks. A skill is a markdown
file that provides domain knowledge to AI agents via MCP prompts. Where a
runbook pairs `runbook.yaml` with `script.sh`, a skill pairs `runbook.yaml`
with `skill.md`.

Inspired by [Claude Code's skill system](https://docs.anthropic.com/en/docs/claude-code/skills),
adapted for dops catalogs and MCP.

**Directory structure:**

Skills live alongside runbooks in the same catalog directory:

```
catalogs/infra/
├── scale-deployment/
│   ├── runbook.yaml        # type: runbook (default, implicit)
│   └── script.sh
├── restart-pods/
│   ├── runbook.yaml
│   └── script.sh
└── k8s-scaling-guide/
    ├── runbook.yaml        # type: skill
    └── skill.md            # markdown context for AI agents
```

**runbook.yaml for a skill:**

```yaml
name: Kubernetes Scaling Guide
type: skill
description: >
  Context for AI agents about Kubernetes scaling operations.
  Use when the user asks about scaling deployments, adjusting
  replica counts, or resizing workloads.
trigger: scale, resize, replicas, horizontal pod autoscaler
```

Fields:
- `type: skill` — distinguishes from executable runbooks (default: `runbook`)
- `description` — used by AI agents to discover when this skill is relevant.
  Should explain what the skill covers and when to load it.
- `trigger` — comma-separated keywords that help agents match user intent
  to available skills. Lightweight discovery hint.
- `name`, `id`, `aliases` — work the same as runbooks.

**skill.md:**

Free-form markdown. The full content is returned when the agent requests
the skill via MCP. Can include instructions, examples, decision trees,
reference tables, links to related runbooks, or any domain knowledge.

**MCP exposure:**

Skills are exposed as MCP prompts:
- Prompt name: skill ID (e.g., `infra.k8s-scaling-guide`)
- Prompt description: from `description` field
- `GetPrompt` returns the full `skill.md` content as a text message
- `ListPrompts` includes all skills with their trigger keywords in metadata

**Not exposed as:**
- MCP tools (skills are not executable)
- TUI sidebar entries (skills are MCP-only, hidden from TUI)

**Catalog loader changes:**

- `DiskCatalogLoader` scans for `skill.md` in addition to `script.sh`
- Entries with `type: skill` are stored separately from runbooks
- Skills are not filtered by risk level (they have no risk)
- Skills are not passed to TUI sidebar

**Files:**
- `internal/domain/runbook.go` — add `Type` field, `Skill` type constant
- `internal/domain/skill.go` — `Skill` struct (Name, ID, Description, Trigger, Content)
- `internal/catalog/loader.go` — load skills alongside runbooks
- `internal/mcp/server.go` — register skills as MCP prompts
- `internal/mcp/prompts.go` — serve skill content via `GetPrompt`

#### Acceptance Criteria

- [ ] `type: skill` in runbook.yaml identifies a skill
- [ ] Skill directory contains `skill.md` (not `script.sh`)
- [ ] `DiskCatalogLoader` loads skills and exposes them separately from runbooks
- [ ] Skills appear in `ListPrompts` with description and trigger metadata
- [ ] `GetPrompt` for a skill returns the full `skill.md` content
- [ ] Skills are NOT shown in the TUI sidebar
- [ ] Skills are NOT exposed as MCP tools
- [ ] Skills without a `skill.md` file log a warning and are skipped
- [ ] Runbooks without `type` field default to `type: runbook` (backward compatible)
- [ ] `dops mcp tools` does not list skills
- [ ] `go test ./...` passes

---

## Fixes

### Fix 1. Execution View — Remove Duplicate Status, Modernize Header

**Status:** DONE

The `ExecutionView.vue` component shows the status pill ("Completed" /
"Failed") in both the header bar and a footer bar that appears on
completion. This creates redundant information. The header bar layout also
needs a modern refresh.

**Current behavior:**
- Header: `[Execution] [runbook-name] [status pill] ... [Cancel/Done/Failed]`
- Footer (on complete): `[status pill] [duration] ... [← Back to runbook]`
- Status pill and right-side label both communicate completion state

**Target behavior:**
- Header: merge duration into the header status pill area on completion.
  Remove the separate right-side "Done" / "Execution failed" labels —
  the status pill already communicates this.
- Footer: replace the status pill with just the duration and back button.
  No duplicate status display.
- Header polish: tighten spacing, use a breadcrumb-style layout
  (`dops / catalog / runbook`), add a subtle separator between sections.

**File:** `web/src/views/ExecutionView.vue`

#### Acceptance Criteria

- [ ] Status pill appears exactly once (header only)
- [ ] Duration appears in the header pill area after completion (e.g., "Completed · 3.2s")
- [ ] Footer shows only duration and back button — no status pill
- [ ] Running state still shows animated pulse dot + Cancel button
- [ ] Error state shows "Failed" pill + error context
- [ ] Header uses breadcrumb-style runbook identification
- [ ] Layout is responsive (no overflow on narrow viewports)
- [ ] `make web` builds without errors
