# dops Architecture

## What dops Is

dops is a DevOps operations tool that makes runbooks discoverable, parameterizable, and safely executable. It provides four interfaces to the same underlying engine: a full-screen TUI, a CLI, a web UI, and an MCP server for AI agents.

Built in Go with Cobra (CLI), BubbleTea (TUI), and Vue 3 (web). Runbooks are YAML manifests paired with shell scripts, organized into catalogs. Parameters are scoped and encrypted at rest.

---

## Directory Layout

```
cmd/                    Cobra commands (one file per command)
  root.go               Entry point — launches TUI by default
  deps.go               Shared dependency bootstrap (loadDeps)
  run.go                CLI runbook execution
  catalog.go            Catalog management (install, add, remove, update)
  config.go             Config get/set/unset/list
  mcp.go                MCP server (stdio + HTTP)
  open.go               Web UI launcher
  init.go               Bootstrap ~/.dops with defaults
  version.go            Version output

internal/
  domain/               Core types — no external dependencies
    config.go           Config, Catalog, CatalogPolicy, Defaults
    runbook.go          Runbook, Parameter, ParameterType
    risk.go             RiskLevel enum (low/medium/high/critical)
    vars.go             Vars, CatalogVars (scoped parameter storage)
    vault.go            VaultStore interface

  config/               Configuration persistence
    store.go            FileConfigStore — Load/Save config.json
    path.go             Dot-notation path access (Set, Get, Unset)

  vault/                Encrypted parameter storage
    vault.go            age-encrypted vault.json, atomic writes

  catalog/              Runbook discovery
    loader.go           DiskCatalogLoader — scan dirs, parse YAML, build alias index

  vars/                 Variable resolution
    resolver.go         3-layer merge: global → catalog → runbook
    keypath.go          Scope-aware vault key generation

  executor/             Script execution
    runner.go           Runner interface, ScriptRunner implementation
    shell_unix.go       Shell detection (Unix)
    shell_windows.go    Shell detection (Windows)
    proc_unix.go        Process management (Unix)
    proc_windows.go     Process management (Windows)
    demo.go             DemoRunner for showcase mode

  tui/                  BubbleTea terminal UI
    app.go              Root model — Init/Update/View, state routing
    exec.go             Execution orchestration (start, stream, cancel)
    overlays.go         Wizard/confirm/palette overlay lifecycle
    layout.go           Dimension computation (layoutDims)
    mouse.go            Mouse coordinate translation, focus
    selection.go        Text selection + clipboard
    sidebar/            Catalog tree with search/filter
    wizard/             Parameter input form (8 field types)
    output/             Live execution output with search + scroll
    confirm/            Risk confirmation modal
    palette/            Theme selector overlay
    help/               Keyboard shortcut reference
    metadata/           Runbook detail display
    footer/             Context-sensitive status bar

  mcp/                  Model Context Protocol server
    server.go           HTTP/stdio server setup
    tools.go            Runbook → MCP tool schema conversion
    resources.go        Catalog/runbook as MCP resources
    prompts.go          MCP prompt templates
    progress.go         Streaming execution progress
    watcher.go          File watcher for catalog changes

  web/                  Web UI server
    server.go           HTTP server (port 3000 default)
    api.go              JSON API endpoints
    handler.go          Request routing + SPA fallback
    embed.go            Embedded Vue.js dist (go:embed)

  theme/                Color theme system
    loader.go           JSON theme file loading
    resolver.go         Background-aware color resolution
    styles.go           Lipgloss style construction
    themes/             20 embedded JSON themes

  crypto/               Encryption primitives
    age.go              age X25519 + ChaCha20-Poly1305
    encrypter.go        Encrypter interface
    mask.go             Secret masking for display

  adapters/             Interface adapters
    fs.go               OSFileSystem (os.ReadFile, os.WriteFile)
    log.go              LogWriter (temp dir execution logs)

  clipboard/            OSC 52 clipboard fallback
  cli/                  Error formatting, styled help
  update/               Version check
  testutil/             Shared test helpers

specs/                  Feature specifications (the contract)
plans/                  Implementation plans (active + completed/)
catalogs/               Example runbook catalogs
tapes/                  VHS terminal recordings (demos + visual tests)
web/                    Vue 3 SPA source
docs/                   VitePress documentation site
assets/                 Logos, demo GIFs, screenshots
```

---

## Data Flow

### Bootstrap Sequence

Every command shares the same `loadDeps()` path:

```
main.go → cmd.Execute()
  → loadDeps(dopsDir)
    1. FileConfigStore.Load()        → config.json
    2. Vault.Load()                  → vault.json (decrypt with age keys)
    3. Merge vault vars into config  → config.Vars populated
    4. ThemeLoader.Load()            → JSON theme file
    5. Resolve() + BuildStyles()     → lipgloss styles
    6. DiskCatalogLoader.LoadAll()   → scan catalog dirs, parse runbook.yaml
    → appDeps{} struct returned
```

### Entry Points

```
dops              → launchTUI()     → BubbleTea program
dops run <id>     → resolveRunbook → buildEnv → Runner.Run()
dops mcp serve    → MCP server     → tools expose runbooks
dops open         → web.NewServer  → HTTP + embedded Vue SPA
dops config       → config store   → get/set/unset/list
dops catalog      → git clone/pull → config update
dops init         → scaffold ~/.dops with hello-world catalog
```

### TUI Message Flow

BubbleTea's Elm architecture drives all TUI state:

```
User Input → tea.Msg
  → App.Update()
    → route to focused sub-model (sidebar, output, wizard, etc.)
    → sub-model returns (Model, tea.Cmd)
    → App processes returned messages
  → App.View()
    → render based on state (normal, wizard, confirm, help, palette)

Key flows:
  sidebar.RunbookSelectedMsg  → App stores selection, updates metadata
  sidebar.RunbookExecuteMsg   → App opens wizard overlay
  wizard.SubmitMsg            → App checks risk → confirm or execute
  wizard.SaveFieldMsg         → App persists to vault → SaveFieldResultMsg back
  confirm.ConfirmMsg          → App starts execution
  executionDoneMsg            → App updates output, clears running state
```

### Execution Pipeline

```
1. Resolve saved vars (global → catalog → runbook scope)
2. Merge with user input (wizard or --param flags)
3. Check risk level against policy ceiling
4. Build env map (uppercase param names as env vars)
5. Runner.Run(ctx, scriptPath, env) → channel of OutputLine
6. Stream lines to output pane (TUI) or stdout (CLI) or SSE (web)
7. Log to temp file (/tmp/dops-<catalog>-<runbook>-<timestamp>.log)
```

---

## Persistence Model

### config.json

Theme, defaults, catalog registry. No secrets, no parameter values.

```json
{
  "theme": "github",
  "defaults": { "max_risk_level": "medium" },
  "catalogs": [
    { "name": "infra", "path": "/path/to/catalog", "active": true,
      "policy": { "max_risk_level": "high" } }
  ]
}
```

### vault.json

All saved parameter values. Encrypted with age (X25519 + ChaCha20-Poly1305). Keys stored in `~/.dops/keys/keys.txt` (0600 permissions). Atomic writes via temp file + rename.

```json
{
  "version": 1,
  "data": "<base64-encoded age-encrypted JSON>"
}
```

Decrypted payload structure (Vars):
```json
{
  "global": { "aws_region": "us-east-1" },
  "catalog": {
    "infra": {
      "vars": { "cluster": "prod" },
      "runbooks": {
        "scale-deployment": { "replicas": "3" }
      }
    }
  }
}
```

### Variable Scoping

Parameters declare their scope in `runbook.yaml`:

| Scope | Storage Key | Applies To |
|-------|------------|------------|
| `global` | `vars.global.<param>` | All runbooks |
| `catalog` | `vars.catalog.<cat>.<param>` | All runbooks in catalog |
| `runbook` | `vars.catalog.<cat>.runbooks.<rb>.<param>` | Single runbook |
| `local` | Not stored | Never persisted |

Resolution precedence: global < catalog < runbook < user input.

---

## Risk Policy

Two enforcement gates protect against accidental execution of dangerous runbooks:

1. **Global ceiling** — `defaults.max_risk_level` in config.json
2. **Catalog ceiling** — `catalog.policy.max_risk_level` (overrides global)

Runbooks exceeding the ceiling are filtered out at load time — invisible in sidebar, CLI, and MCP tools.

For runbooks within the ceiling:
- `low` / `medium` — execute directly after wizard
- `high` / `critical` — confirmation modal required (TUI) or `_confirm_*` fields (MCP)

---

## MCP Integration

Runbooks are exposed as first-class MCP tools:

```
Tool name:     runbook ID (e.g., "infra.scale-deployment")
Description:   from runbook.yaml
Input schema:  JSON Schema built from parameters
```

Transports: stdio (AI agent integration) and HTTP (programmatic access).

Risk confirmation for high/critical runbooks uses synthetic `_confirm_id` and `_confirm_word` input fields. Output is truncated to the last 50 lines. Structured `ToolResult` includes exit code, duration, output, and log path.

Resources exposed:
- `dops://catalog` — all runbooks (JSON)
- `dops://catalog/{id}` — single runbook detail
- `dops://schema/runbook` — YAML schema guide
- `dops://schema/shell-style` — shell style guide

---

## Runbook Format

Each runbook is a directory containing `runbook.yaml` + `script.sh`:

```yaml
# runbook.yaml
name: Scale Deployment
description: Scale a Kubernetes deployment to N replicas
risk_level: medium
aliases: [scale, scale-deploy]
script: script.sh
parameters:
  - name: namespace
    type: select
    required: true
    scope: catalog
    options: [production, staging, development]
  - name: deployment
    type: string
    required: true
    scope: runbook
  - name: replicas
    type: integer
    required: true
    scope: local
    default: 3
```

Parameters become uppercase environment variables in the script: `$NAMESPACE`, `$DEPLOYMENT`, `$REPLICAS`.

Supported parameter types: `string`, `integer`, `number`, `float`, `boolean`, `select`, `multi_select`, `file_path`, `resource_id`.

---

## Development Workflow

### Spec-Driven

Non-trivial features start as specs in `specs/`. The spec defines requirements, acceptance criteria, error cases, and out-of-scope items. Implementation plans in `plans/` break specs into phases (max 5 files per phase). Tests derive directly from acceptance criteria.

### Clean Code Skills

The `.claude/skills/` directory encodes project conventions as reusable skill definitions:

| Skill | Governs |
|-------|---------|
| `clean-code-naming` | Go naming conventions |
| `clean-code-functions` | Function size (<30 LOC), SLA, command-query separation |
| `clean-code-design` | YAGNI, KISS, DRY, composition, Law of Demeter |
| `clean-code-solid` | SRP, OCP, LSP, ISP, DIP applied to Go |
| `clean-code-errors` | Error wrapping, sentinels, custom types |
| `clean-code-boundaries` | Wrapping externals, interfaces at consumer |
| `clean-code-testing` | FIRST principles, table-driven, one concept per test |

### Testing

- Table-driven tests with `t.Helper()` and `t.TempDir()`
- Interface stubs over mocking frameworks
- No global mutable state in tests
- Mutation testing with Gremlins for test quality assessment
- VHS tape scripts for visual TUI testing
- ~39 test files covering all packages

### Verification

Work is not complete until:
1. `go vet ./...` passes
2. `go test ./...` passes
3. Linters pass
4. Manual smoke test confirms behavior

---

## Key Design Decisions

1. **Message-based persistence** — Wizard emits `SaveFieldMsg` to the parent App instead of directly accessing config/vault. Enables testing without mocking I/O.

2. **Encrypted vault separate from config** — Secrets never touch config.json. Vault uses age encryption with per-machine keys.

3. **Risk as a first-class concept** — Every runbook declares risk level. Policy enforcement happens at catalog load time, not execution time.

4. **Four interfaces, one engine** — TUI, CLI, Web, and MCP all share `loadDeps()`, the same catalog loader, executor, and var resolver. No duplicated business logic.

5. **Custom wizard over form libraries** — Field-by-field input with per-field save control, auto-skip for saved values, and scope-aware persistence. No dependency on Huh or similar.

6. **Embedded assets** — Themes (JSON) and web UI (Vue dist) are embedded via `go:embed`. Single binary distribution.

7. **Platform-aware execution** — Build tags split Unix/Windows shell detection and process management. Scripts get appropriate shell invocation per platform.
