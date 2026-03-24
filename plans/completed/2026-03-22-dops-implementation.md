# dops Implementation Plan

## Approach

Build bottom-up: domain types and config first, then CLI commands, then TUI. Each phase is independently testable and produces a working artifact. Later phases compose earlier ones.

### Architecture Principles

- **Depend on interfaces, not concretions** — every cross-package dependency flows through an interface defined at the consumer site (DIP)
- **Adapters at the edges** — filesystem, age, os/exec, and clipboard are wrapped in thin adapters behind domain interfaces (Boundaries)
- **One responsibility per package** — config parsing is separate from dot-notation traversal, loading is separate from saving (SRP)
- **Accept interfaces, return structs** — the Go way to apply DIP
- **Wire at the root** — `cmd/` is the composition root; `internal/` never imports `cmd/`

### TDD Workflow

Every phase follows strict test-driven development:

1. **Red** — write a failing test that describes the expected behavior from the spec
2. **Green** — write the minimum code to make the test pass
3. **Refactor** — clean up while all tests stay green

Concrete rules:
- **No production code without a failing test first.** If a function exists, a test drove its creation.
- **Tests are the first artifact of each phase.** Start by writing test files, then implement.
- **One test per acceptance criterion** — each criterion in this plan maps to at least one test case.
- **Table-driven tests for input/output functions** — write the table (with all cases) first, then implement until every row passes.
- **Interfaces emerge from tests** — when a test needs a dependency, define the interface the test requires. The production implementation comes after.
- **Refactor only on green** — never refactor while tests are failing.
- **Commit cadence: red → green → refactor** — each cycle is a potential commit point.

---

## Directory Layout

```
main.go
cmd/
├── root.go              # Cobra root, persistent flags, dependency wiring
├── version.go
├── run.go               # dops run <id> --param key=value
├── config.go            # dops config parent command
├── config_set.go
├── config_get.go
├── config_unset.go
└── config_list.go
internal/
├── domain/              # Pure domain types — no dependencies, no I/O
│   ├── config.go        # Config, Catalog, Defaults, Vars structs
│   ├── runbook.go       # Runbook, Parameter, ParameterType
│   ├── risk.go          # RiskLevel type with ordered comparison
│   └── theme.go         # ThemeFile, ThemeDef, ThemeToken structs
├── config/
│   ├── store.go         # ConfigStore interface + file-backed implementation
│   └── path.go          # Dot-notation get/set/unset on Config structs
├── catalog/
│   ├── loader.go        # CatalogLoader interface + implementation
│   └── filter.go        # Risk-level filtering logic
├── vars/
│   └── resolver.go      # VarResolver interface + implementation
├── crypto/
│   ├── encrypter.go     # Encrypter interface
│   └── age.go           # age-backed implementation (adapter)
├── theme/
│   ├── loader.go        # ThemeLoader interface + file/embed implementation
│   ├── resolver.go      # Def resolution + dark/light selection
│   ├── styles.go        # Styles struct (lipgloss.Style per token)
│   └── tokyonight.json  # Embedded default theme
├── executor/
│   ├── runner.go        # Runner interface
│   └── script.go        # os/exec-backed implementation (adapter)
├── clipboard/
│   └── clipboard.go     # Clipboard interface + OS adapter
├── tui/
│   ├── app.go           # Root tea.Model — state machine, message routing
│   ├── styles.go        # Centralized lipgloss styles from resolved theme
│   ├── layout.go        # Compose regions with lipgloss layout
│   ├── sidebar/         # Separate tea.Model — own Update/View
│   │   ├── model.go
│   │   ├── search.go    # Fuzzy filter sub-component
│   │   └── messages.go  # RunbookSelectedMsg, etc.
│   ├── metadata/        # View function (stateless — renders from selected runbook)
│   │   └── view.go
│   ├── output/          # Separate tea.Model — own Update/View
│   │   ├── model.go
│   │   ├── search.go    # In-pane search sub-component
│   │   └── messages.go  # OutputLineMsg, ExecutionDoneMsg, etc.
│   ├── wizard/          # Separate tea.Model — wraps huh.Form
│   │   ├── model.go
│   │   └── messages.go  # WizardSubmitMsg, WizardCancelMsg
│   ├── palette/         # Separate tea.Model
│   │   ├── model.go
│   │   └── messages.go
│   └── footer/          # View function (stateless — renders from app state)
│       └── view.go
└── adapters/            # Thin wrappers for external I/O
    ├── fs.go            # FileSystem interface + os-backed implementation
    └── log.go           # LogWriter interface + file-backed implementation
```

### Key Interfaces (defined at consumer sites)

```go
// internal/config/store.go — consumed by cmd/, tui/wizard, tui/palette
type ConfigStore interface {
    Load() (*domain.Config, error)
    Save(cfg *domain.Config) error
}

// internal/catalog/loader.go — consumed by cmd/root.go, tui/app.go
type CatalogLoader interface {
    LoadAll(catalogs []domain.Catalog, defaultRisk domain.RiskLevel) ([]CatalogWithRunbooks, error)
}

// internal/vars/resolver.go — consumed by tui/wizard, executor
type VarResolver interface {
    Resolve(cfg *domain.Config, catalogName, runbookName string, params []domain.Parameter) map[string]string
}

// internal/crypto/encrypter.go — consumed by config, vars, tui/wizard
type Encrypter interface {
    Encrypt(plaintext string) (string, error)
    Decrypt(ciphertext string) (string, error)
    IsEncrypted(value string) bool
}

// internal/executor/runner.go — consumed by tui/app.go
type Runner interface {
    Run(ctx context.Context, scriptPath string, env map[string]string) (<-chan OutputLine, error)
}

// internal/adapters/fs.go — consumed by config/store, catalog/loader, theme/loader
type FileSystem interface {
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte, perm os.FileMode) error
    ReadDir(path string) ([]os.DirEntry, error)
    MkdirAll(path string, perm os.FileMode) error
    Stat(path string) (os.FileInfo, error)
}

// internal/clipboard/clipboard.go — consumed by tui/output, tui/wizard
type Clipboard interface {
    Write(text string) error
}
```

### TUI Component Architecture

Components are either **models** (stateful, own `Update`/`View`) or **views** (stateless render functions):

| Component | Type | Why |
|---|---|---|
| `tui/app` | Model | Root state machine — routes messages based on active view state |
| `tui/sidebar` | Model | Owns selection state, search state, scroll position |
| `tui/output` | Model | Owns output buffer, search state, scroll position |
| `tui/wizard` | Model | Owns huh.Form lifecycle, manages submit/cancel |
| `tui/palette` | Model | Owns command filter, input state |
| `tui/metadata` | View function | Stateless — renders whatever runbook the app passes it |
| `tui/footer` | View function | Stateless — renders keybinds based on app state |

**Message routing:** The root `app` model receives all messages. It delegates to the focused component's `Update`. Child models return domain messages (e.g., `RunbookSelectedMsg`, `WizardSubmitMsg`) that the root model handles to coordinate between components.

```go
// tui/app.go — simplified routing
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch m.state {
    case stateNormal:
        // Route keyboard/mouse to focused panel (sidebar or output)
        // Handle RunbookSelectedMsg from sidebar → update metadata
    case stateWizard:
        // Route all input to wizard model
        // Handle WizardSubmitMsg → save config, start execution
        // Handle WizardCancelMsg → return to normal
    case statePalette:
        // Route all input to palette model
    }
}
```

---

## Test Strategy

All development follows TDD: write the failing test first, then the minimum code to pass, then refactor. Each phase's "TDD Order" section defines the sequence of tests to write. Tests are the first artifact — code follows.

### Unit Tests (per package)

Every package gets `*_test.go` files. Test through public APIs only.

| Package | Test doubles | Key test patterns |
|---|---|---|
| `domain` | None (pure types) | Table-driven: RiskLevel comparison, parameter type validation |
| `config/path` | None (pure logic) | Table-driven: get/set/unset with nested paths, edge cases |
| `config/store` | Fake `FileSystem` | Round-trip: load → save → load. Missing file creates defaults |
| `catalog` | Fake `FileSystem` | Filter inactive catalogs, filter by risk, malformed YAML handling |
| `vars` | None (pure logic) | Table-driven: precedence resolution, missing keys, empty scopes |
| `crypto` | None (test real age) | Round-trip: encrypt → decrypt. IsEncrypted edge cases |
| `theme` | Fake `FileSystem` | Def resolution, dark/light selection, fallback chain, invalid refs |
| `executor` | Fake `Runner` for consumers | Real `os/exec` in executor tests with test scripts |

### TUI Tests

- **Component-level:** each model tested via `Update()` with synthetic messages, assert on returned model state
- **Golden files:** for View output — render a model state and compare to `testdata/*.golden`
- **No visual/manual testing required** for basic correctness

### Integration Tests

- `cmd/` tests: run Cobra commands against a temp `~/.dops/` directory with real filesystem
- End-to-end: load config → load catalogs → resolve vars → execute script → verify log output

### Test Conventions

- Table-driven for all input/output mapping functions
- Arrange-Act-Assert structure
- Fakes over mocks (implement the interface with simple structs)
- `testdata/` directories for fixture files (config.json, runbook.yaml, theme.json)
- `t.Helper()` on all test helpers

---

## Phase 1 — Project Scaffold & Domain Types

**Goal:** Go module, domain types, config store, catalog loader, vars resolver. All with interfaces and tests.

### Steps

1. **Init Go module** — `go mod init dops`, add dependencies
2. **`internal/domain/`** — pure types, no I/O:
   - `Config`, `Catalog`, `Defaults` structs
   - `Vars` — flat structure, no `inputs` nesting: `Global map[string]any`, `Catalog map[string]CatalogVars` where `CatalogVars` holds vars directly + `Runbooks map[string]map[string]any`
   - `Runbook` with `ID` field — globally unique identifier in `<catalog>.<runbook>` format (e.g. `"default.hello-world"`), used as the CLI invocation key for `dops run <id>`
   - `Parameter`, `ParameterType` (string/boolean/integer/select)
   - `RiskLevel` with `const` values and `Exceeds(other RiskLevel) bool` method
   - `ThemeFile`, `ThemeDef`, `ThemeToken` structs
3. **`internal/adapters/fs.go`** — `FileSystem` interface + `OSFileSystem` implementation
4. **`internal/config/store.go`** — `ConfigStore` interface + `FileConfigStore` (accepts `FileSystem`):
   - `Load() (*domain.Config, error)`
   - `Save(cfg *domain.Config) error` — atomic write (write tmp + rename)
   - `EnsureDefaults() (*domain.Config, error)` — create `~/.dops/` and default config if missing
5. **`internal/config/path.go`** — pure functions, no I/O:
   - `Get(cfg *domain.Config, keyPath string) (any, error)`
   - `Set(cfg *domain.Config, keyPath string, value any) error`
   - `Unset(cfg *domain.Config, keyPath string) error`
6. **`internal/catalog/loader.go`** — `CatalogLoader` interface + `DiskCatalogLoader` (accepts `FileSystem`):
   - `LoadAll(catalogs []domain.Catalog, defaultRisk domain.RiskLevel) ([]CatalogWithRunbooks, error)`
   - Walks catalog dirs, parses `runbook.yaml`, applies risk filter
   - `FindByID(id string) (*domain.Runbook, *domain.Catalog, error)` — look up a runbook by its `id` field across all loaded catalogs
7. **`internal/vars/resolver.go`** — `VarResolver` interface + `DefaultVarResolver`:
   - `Resolve(cfg *domain.Config, catalogName, runbookName string, params []domain.Parameter) map[string]string`
   - Pure logic: merge global → catalog → runbook

### TDD Order

Write tests first in this sequence (red → green → refactor for each):

1. `domain/risk_test.go` — table-driven `RiskLevel.Exceeds` (pure logic, no deps)
2. `domain/runbook_test.go` — validate `ID` format (`<catalog>.<name>`)
3. `config/path_test.go` — table-driven get/set/unset with nested paths (pure logic)
4. `config/store_test.go` — fake FileSystem, round-trip, missing file defaults
5. `catalog/loader_test.go` — fake FileSystem with fixture YAML, risk filtering, `FindByID`
6. `vars/resolver_test.go` — table-driven precedence with overlapping keys

### Acceptance Criteria
- [ ] `go build` succeeds
- [ ] Config round-trips: load → modify → save → load produces identical result
- [ ] Dot-notation get/set/unset works for nested paths including `vars.catalog.X.runbooks.Y.dry_run`
- [ ] Vars structure is flat — no `inputs` nesting (e.g., `vars.global.region` not `vars.global.inputs.region`)
- [ ] Catalog loader correctly filters inactive catalogs and risk-excluded runbooks
- [ ] `FindByID` returns the correct runbook and its parent catalog
- [ ] `FindByID` returns error for unknown IDs
- [ ] Vars resolution follows precedence: runbook > catalog > global
- [ ] RiskLevel comparison is correct (`low < medium < high < critical`)
- [ ] Runbook ID validation rejects malformed IDs (missing dot, empty segments)
- [ ] All tests pass — and they were written before the implementation

---

## Phase 2 — Crypto (age) & Secret Handling

**Goal:** Encrypt/decrypt secret parameter values using age, behind an interface.

### Steps

1. **`internal/crypto/encrypter.go`** — `Encrypter` interface:
   ```go
   type Encrypter interface {
       Encrypt(plaintext string) (string, error)
       Decrypt(ciphertext string) (string, error)
       IsEncrypted(value string) bool
   }
   ```
2. **`internal/crypto/age.go`** — `AgeEncrypter` implementation (accepts key path):
   - `EnsureKey(keysDir string) error` — generate identity if missing
   - Implements `Encrypter` using `filippo.io/age`
3. **`internal/crypto/mask.go`** — `MaskSecrets(cfg *domain.Config, enc Encrypter) *domain.Config`
4. **Integrate with vars resolver** — `DecryptingVarResolver` wraps `VarResolver` + `Encrypter`:
   - After resolution, decrypt any encrypted values before returning

### TDD Order

1. `crypto/encrypter_test.go` — test `IsEncrypted` table-driven (pure logic, define interface here)
2. `crypto/age_test.go` — test round-trip encrypt → decrypt with temp key file (integration)
3. `crypto/mask_test.go` — test `MaskSecrets` with fake Encrypter, mixed plain/encrypted values
4. `vars/decrypting_resolver_test.go` — test `DecryptingVarResolver` with fake Encrypter

### Acceptance Criteria
- [ ] Key generation creates a valid age identity file
- [ ] Round-trip: encrypt → decrypt produces original plaintext
- [ ] `IsEncrypted` correctly identifies age ciphertext
- [ ] `MaskSecrets` replaces all encrypted values with `****`
- [ ] `DecryptingVarResolver` decrypts encrypted values, passes plain values through
- [ ] All tests written before implementation

---

## Phase 3 — Theme Engine

**Goal:** Load themes, resolve defs, detect dark/light, produce lipgloss styles.

### Steps

1. **`internal/theme/loader.go`** — `ThemeLoader` interface + `FileThemeLoader` (accepts `FileSystem`):
   - `Load(name string) (*domain.ThemeFile, error)` — user dir first, then embedded
   - Embed `tokyonight.json` via `//go:embed`
2. **`internal/theme/resolver.go`** — pure functions:
   - `Resolve(tf *domain.ThemeFile, isDark bool) *ResolvedTheme` — expand def refs, pick variant
   - Validates all def references exist, returns error for dangling refs
3. **`internal/theme/styles.go`** — `Styles` struct:
   - One `lipgloss.Style` field per token
   - `BuildStyles(rt *ResolvedTheme) *Styles` — construct from resolved theme
   - Exposes computed styles for every TUI region

### TDD Order

1. `theme/resolver_test.go` — table-driven def resolution, dark/light selection, dangling refs (pure logic)
2. `theme/loader_test.go` — fake FileSystem, fallback chain, user override
3. `theme/styles_test.go` — verify style properties match resolved colors

### Acceptance Criteria
- [ ] Bundled tokyonight theme loads without user files
- [ ] User theme in `~/.dops/themes/` overrides bundled theme of same name
- [ ] Def references resolve correctly (e.g., `"blue"` → `"#7aa2f7"`)
- [ ] Dark/light variant selection works
- [ ] Unknown theme name falls back to tokyonight
- [ ] Dangling def references produce a clear error
- [ ] All tests written before implementation

---

## Phase 4 — CLI Commands (`dops config` & `dops run`)

**Goal:** `dops config set/get/unset/list` and `dops run <id>` commands via Cobra.

### Steps

1. **cmd/config.go** — parent `config` command (no `RunE`, groups subcommands)
2. **cmd/config_set.go** — parse `key=value` arg, `--secret` flag, calls `config.Set` + `Encrypter.Encrypt`
3. **cmd/config_get.go** — reads value, masks if encrypted
4. **cmd/config_unset.go** — removes key
5. **cmd/config_list.go** — pretty-prints full config via `MaskSecrets`
6. **cmd/run.go** — `dops run <id> --param key=value`:
   - Look up runbook by `id` via `CatalogLoader.FindByID`
   - Resolve saved vars, apply `--param` overrides
   - Prompt interactively for missing required params (if TTY available)
   - Save inputs to config at correct scopes, encrypt `--secret` params
   - Execute script via `Runner`, stream stdout/stderr to terminal (plain, no TUI)
   - Write log file
   - Supports `--no-save` and `--dry-run` flags
7. **Wiring in cmd/root.go** — construct `FileConfigStore` with `OSFileSystem`, `AgeEncrypter` with key path, `ScriptRunner`, pass to subcommands
8. **Styled CLI error output** — `internal/cli/error.go`:
   - `PrintError(title, detail string)` renders styled badge: bold white on red `ERROR` + title, muted detail on next line
   - Root command sets `SilenceUsage: true` and `SilenceErrors: true`
   - Custom error handler in `Execute()` calls `PrintError` for all errors
   - No usage dump on errors — users use `--help` explicitly

### TDD Order

1. Integration tests for `config set/get/unset/list` against temp directory with real filesystem
2. `run` command tests:
   - Test `FindByID` lookup (known ID, unknown ID, risk-blocked ID)
   - Test `--param` override merging with resolved vars
   - Test `--dry-run` outputs resolved command without executing
   - Test `--no-save` does not write to config
   - Test missing required param with no TTY produces error

### Acceptance Criteria
- [ ] `dops config set theme=dracula` updates config.json
- [ ] `dops config set vars.global.token=abc --secret` encrypts and stores
- [ ] `dops config get vars.global.token` prints `****` for secrets
- [ ] `dops config unset vars.global.region` removes the key
- [ ] `dops config list` shows full config with masked secrets
- [ ] `dops run default.hello-world` executes the correct script
- [ ] `dops run default.hello-world --param namespace=staging` overrides the saved value
- [ ] `dops run unknown.id` shows styled error badge, no usage dump
- [ ] `dops run <risk-blocked-id>` shows styled error about risk policy
- [ ] `--dry-run` shows command without executing
- [ ] `--no-save` does not persist inputs
- [ ] All CLI errors render with `ERROR` badge + title + muted detail
- [ ] All tests written before implementation

---

## Phase 5 — TUI Foundation (Main View Shell)

**Goal:** BubbleTea program with themed, bordered panels matching the wireframe. The TUI must visually match `specs/diagrams/tui-layout.png`. See spec §6.5 for complete visual requirements.

### Steps

1. **Fix theme `border` token** — update `tokyonight.json` to map `border` to `fgMuted` (`#565f89`) instead of `bgElem` (`#292e42`). The current value is invisible against the background. This single change fixes most visibility issues.

2. **`tui/app.go`** — root `tea.Model`:
   - State enum: `stateNormal`, `stateWizard`, `statePalette`
   - Focus tracking: `focusSidebar`, `focusOutput`
   - Accepts `AppDeps` with `*theme.Styles`, `Runner`, `ConfigStore`, `LogWriter`
   - Handles `tea.WindowSizeMsg` — recalculate all panel sizes, pass to children
   - `RunbookSelectedMsg` (cursor move) → update metadata, clear output
   - `RunbookExecuteMsg` (Enter on runbook) → open wizard or start execution
   - View uses `AltScreen = true` (BubbleTea v2 declarative)
   - Layout: sidebar fills left column height, metadata auto-height at top-right, output fills remaining vertical space, footer pinned at bottom

3. **`tui/sidebar/model.go`** — separate model, accepts `*theme.Styles`:
   - Panel wrapped in `lipgloss.RoundedBorder()` with `border` color, `borderActive` when focused
   - No background fill — transparent background inherits terminal default
   - Left padding (1 col) inside border for content inset
   - **Collapsible catalogs:** `▼`/`▶` indicators on catalog headers, `←`/`→` to collapse/expand, `Enter`/`Space` toggles
   - `←` on a runbook jumps cursor to parent catalog header
   - **Mouse click** on header toggles collapse/expand, click on runbook selects it
   - **Mouse hover** highlights item under cursor with underline (`styles.Text.Underline`), clears on keyboard input
   - Mouse coordinates translated from terminal-absolute to content-relative by the app (`translateMouseForSidebar`) before forwarding; sidebar `mouseToIdx()` uses `y + scrollOffset` directly
   - Mouse enabled via `view.MouseMode = tea.MouseModeCellMotion` (v2 declarative)
   - Cursor navigates all visible items (headers + runbooks), not just runbooks
   - `Enter` on a runbook emits `RunbookExecuteMsg` (triggers wizard/execution)
   - `Enter` on a header toggles collapse/expand
   - No selection indicator — selected runbook distinguished by bold `text` style only
   - Catalog headers: `primary` when selected, `textMuted` otherwise
   - Runbook names: `text` when selected (bold), `textMuted` otherwise
   - No risk badges in sidebar — risk level shown in metadata panel only
   - Tree connectors (`├──`, `└──`) flush-aligned with catalog arrows

4. **`tui/metadata/view.go`** — stateless render function, accepts `*theme.Styles`:
   - Own rounded border panel, no background fill — transparent background
   - Layout: `Name version` (bold + muted), risk badge, blank, description, blank, location path/URL
   - `Location(rb, cat)` helper returns raw path or URL string
   - Local catalogs: path to `runbook.yaml` with OSC 8 `file://` hyperlink
   - Git catalogs (URL field set): catalog URL with OSC 8 hyperlink
   - `Render` accepts `copied bool` — when true, replaces location line with `"Copied to Clipboard!"` in `success` color
   - **Click-to-copy**: app detects clicks on the path/URL text (exact character bounds), copies to clipboard via `tea.SetClipboard` (OSC 52), shows flash for 2 seconds
   - Auto-height (6-8 lines based on content)

5. **`tui/output/model.go`** — separate model, accepts `*theme.Styles`:
   - Own rounded border panel, fills remaining vertical space
   - **Header**: `backgroundElement` background fill, `text` foreground — command text must be readable against the fill
   - **Body**: default `background`, stderr in `error` color
   - **Footer**: `backgroundElement` background fill, log path in `textMuted` — must be readable
   - Placeholder when no execution: centered `"Press enter to run a runbook"` in `textMuted`

6. **`tui/footer/view.go`** — stateless, accepts `*theme.Styles`:
   - Full-width, no background fill — transparent background
   - Keybind keys in `primary`, descriptions in `textMuted`
   - Consistent left padding

7. **`tui/layout.go`** — responsive layout:
   - Sidebar: 25% width, min 20, max 40 cols. Fills full height minus footer.
   - Right panel: remaining width. Metadata at top (auto-height), output fills rest.
   - No dead space — output pane expands to fill all available vertical area.
   - Footer: 1 line, full width, pinned to bottom.
   - All panels use `lipgloss.RoundedBorder()` + `border` token foreground

### TDD Order

1. `tui/sidebar/model_test.go` — navigation, selection messages, risk badges in view output
2. `tui/metadata/view_test.go` — rendered detail includes name, risk badge, description
3. `tui/output/model_test.go` — placeholder text when empty, header/footer visibility
4. `tui/footer/view_test.go` — keybind rendering per state
5. `tui/app_test.go` — message routing, output clears on selection change, window resize recalculates

### Acceptance Criteria
- [ ] Theme `border` token maps to `fgMuted` — borders are clearly visible
- [ ] Sidebar panel has rounded border, no background fill, left padding, no risk badges
- [ ] Catalog arrows use `▼`/`▶`, tree connectors flush-aligned with arrows
- [ ] Selected runbook distinguished by bold style only (no `>` indicator)
- [ ] Metadata panel has its own rounded border, no background fill, visually separate from output
- [ ] Output pane header has visible `backgroundElement` fill with readable command text
- [ ] Output pane body fills remaining vertical space — no dead area
- [ ] Output pane footer has visible `backgroundElement` fill with readable log path
- [ ] Output shows placeholder when no execution has occurred
- [ ] Output clears when a different runbook is selected
- [ ] Footer bar has `backgroundPanel` background with styled keybind hints
- [ ] Layout matches wireframe proportions with no dead space
- [ ] Arrow keys navigate, `q` quits, mouse click selects
- [ ] All tests written before implementation

---

## Phase 6 — Sidebar Search & Scrolling

**Goal:** Fuzzy search and scrollbar for the sidebar.

### Steps

1. **`tui/sidebar/search.go`** — search sub-component:
   - `/` toggles search mode, renders text input at sidebar bottom
   - Fuzzy-filters runbook list, hides empty catalogs
   - Escape/clear restores full tree
2. **Scrollbar** — vertical scrollbar when tree items exceed visible height
3. **Auto-highlight** — first match selected during search

### TDD Order

1. `tui/sidebar/search_test.go` — type query → verify filtered list, highlight follows first match
2. Edge cases: no matches returns empty, all filtered hides catalogs, escape restores full tree
3. Scrollbar: verify rendered when items exceed height, not rendered when they fit

### Acceptance Criteria
- [ ] `/` opens search input at bottom of sidebar
- [ ] Typing filters runbooks by fuzzy match
- [ ] Empty catalogs are hidden during search
- [ ] First match is auto-highlighted
- [ ] Escape restores full tree
- [ ] Scrollbar appears when content exceeds height
- [ ] All tests written before implementation

---

## Phase 7 — Wizard Overlay (Huh Forms)

**Goal:** Parameter collection wizard using Huh, with skip/partial-skip behavior.

### Steps

1. **`tui/wizard/model.go`** — wizard `tea.Model`:
   - Accepts `Runbook`, resolved vars, `Encrypter`, `ConfigStore`
   - Builds `huh.Form` from parameters:
     - `string` → `huh.NewInput()`
     - `boolean` → `huh.NewConfirm()`
     - `integer` → `huh.NewInput()` with integer validation
     - `select` → `huh.NewSelect()`
     - `secret: true` → `EchoMode(huh.EchoModePassword)`
   - Pre-fills from resolved vars, skips fields with values
   - Emits `WizardSubmitMsg{params}` or `WizardCancelMsg`
2. **Full skip logic** in `tui/app.go` — if all required params resolved, bypass wizard, go straight to execution
3. **Header** — renders `$ dops run <id>` with live param updates (e.g. `$ dops run default.hello-world --param namespace=staging`)
4. **On submit** — app handles `WizardSubmitMsg`: save to config at correct scopes, encrypt secrets, trigger runner

### TDD Order

1. `tui/wizard/model_test.go` — skip logic: all params resolved → verify wizard never created
2. Partial skip: some params resolved → verify only missing fields in form
3. Form building: verify parameter types map to correct Huh fields
4. Submit: synthetic form completion → verify `WizardSubmitMsg` with correct values and scopes
5. Cancel: escape → verify `WizardCancelMsg`, no side effects

### Acceptance Criteria
- [ ] Enter on selected runbook opens wizard overlay
- [ ] All parameter types render correct Huh fields
- [ ] Secrets are masked during input
- [ ] Pre-existing values are pre-filled
- [ ] Wizard skips entirely when all required params are resolved
- [ ] Wizard skips resolved fields, shows only missing ones
- [ ] Escape closes without side effects
- [ ] Submit saves to config.json at correct scope
- [ ] All tests written before implementation

---

## Phase 8 — Script Execution & Output Streaming

**Goal:** Run scripts, stream output to the output pane, save logs.

### Steps

1. **`internal/executor/runner.go`** — `Runner` interface:
   ```go
   type OutputLine struct {
       Text     string
       IsStderr bool
   }

   type Runner interface {
       Run(ctx context.Context, scriptPath string, env map[string]string) (<-chan OutputLine, error)
   }
   ```
2. **`internal/executor/script.go`** — `ScriptRunner` implementation:
   - Wraps `os/exec.CommandContext`
   - Pipes stdout/stderr separately, sends `OutputLine` per line to channel
   - Closes channel on process exit
3. **`internal/adapters/log.go`** — `LogWriter` interface + file implementation:
   - Writes all output to `/tmp/YYYY.MM.DD-HHmmss-<catalog>-<runbook>.log`
4. **`tui/output/model.go`** — integrate streaming:
   - Receives `OutputLineMsg` from tea.Cmd wrapping the channel
   - Appends to buffer, renders with stderr in error color
   - Header: command string. Footer: log path (after completion)
5. **`tui/app.go`** — wire execution end-to-end:
   - `NewApp` accepts `executor.Runner` and `*adapters.LogWriter`
   - On `WizardSubmitMsg` (or wizard skip when all params resolved):
     1. Save params to config via `ConfigStore`
     2. Build env map from resolved params
     3. Resolve script path from catalog path + runbook script field
     4. Create log file via `LogWriter`
     5. Start `Runner.Run()` in a `tea.Cmd`
     6. Return a subscription-style cmd that reads from the output channel and sends `OutputLineMsg` for each line
     7. On channel close, send `ExecutionDoneMsg` with log path
   - Route `OutputLineMsg` → output model (appends to buffer, writes to log)
   - Route `ExecutionDoneMsg` → output model (shows log path in footer)
   - Output pane must show live streaming output during execution
6. **Clipboard integration** — click header/footer copies text

### TDD Order

1. `executor/script_test.go` — integration test with real test script in `testdata/`, verify stdout/stderr channel output
2. `adapters/log_test.go` — verify log file written with correct filename format to temp dir
3. `tui/output/model_test.go` — synthetic `OutputLineMsg` → verify buffer contents, stderr flagging
4. `tui/output/` golden files — rendered output with header, body (mixed stdout/stderr), footer

### Acceptance Criteria
- [ ] Script executes with correct env vars
- [ ] stdout streams live to output body in the TUI
- [ ] stderr renders in `styles.Error` color in the TUI
- [ ] Output pane header shows the executed command
- [ ] Log file is written with correct filename format
- [ ] Footer shows log path after completion
- [ ] Click-to-copy works for header and footer
- [ ] Enter on runbook → wizard (or skip) → execution starts → output streams live — full end-to-end in TUI
- [ ] All tests written before implementation

---

## Phase 9 — Output Pane Search & Scrolling

**Goal:** In-pane search with match highlighting and vim-style navigation.

### Steps

1. **`tui/output/search.go`** — search sub-component:
   - `/` activates search input at bottom of output body
   - Highlights all matches inline (does not filter)
   - `n`/`N` navigate matches, `[X/Y]` counter in status
   - Auto-scroll to keep current match visible
   - Escape clears highlights
2. **Scrollbar** — vertical scrollbar when content exceeds height

### TDD Order

1. `tui/output/search_test.go` — inject buffer, type query, verify match positions and count
2. Navigation: `n`/`N` → verify current match index advances/retreats, wraps around
3. Edge cases: no matches, empty buffer, escape clears state

### Acceptance Criteria
- [ ] `/` opens search in output pane
- [ ] Matches are highlighted inline
- [ ] `n`/`N` navigate between matches
- [ ] Match counter shows `[X/Y]`
- [ ] View auto-scrolls to current match
- [ ] Escape clears search
- [ ] All tests written before implementation

---

## Phase 10 — Command Palette

**Goal:** `Ctrl+Shift+P` overlay with fuzzy command search.

### Steps

1. **`tui/palette/model.go`** — palette `tea.Model`:
   - Text input for filtering
   - List of `PaletteCommand` entries (name, description, handler)
   - Emits `PaletteSelectMsg{command}` or `PaletteCancelMsg`
2. **Commands:** `theme: set`, `config: set`, `config: view`, `config: delete`, `secrets: re-encrypt`
3. **Secondary prompts** — commands needing input open a follow-up Huh field inside the palette
4. **All writes through `ConfigStore`** — same interface as CLI and wizard

### TDD Order

1. `tui/palette/model_test.go` — type filter → verify filtered command list
2. Command selection: verify `PaletteSelectMsg` emitted with correct command
3. Integration: select `theme: set` → verify theme change written via `ConfigStore`

### Acceptance Criteria
- [ ] `Ctrl+Shift+P` opens palette
- [ ] Typing filters commands
- [ ] `theme: set` shows available themes and applies selection
- [ ] `config: set` accepts key=value input
- [ ] `config: view` displays masked config
- [ ] Escape closes palette
- [ ] All tests written before implementation

---

## Phase Order & Dependencies

```
Phase 1 (domain, config, catalog, vars, adapters)
  ├── Phase 2 (crypto) ─────── depends on domain, adapters
  ├── Phase 3 (theme) ──────── depends on domain, adapters
  └── Phase 4 (CLI) ────────── depends on config, crypto
Phase 5 (TUI shell) ────────── depends on 1, 3
  └── Phase 6 (sidebar search)
Phase 7 (wizard) ────────────── depends on 5, vars, crypto
Phase 8 (execution) ─────────── depends on 7, executor
  └── Phase 9 (output search)
Phase 10 (palette) ──────────── depends on 5, config
```

Phases 2, 3, and 4 can be worked in parallel after Phase 1.
Phases 6, 9, and 10 are enhancements that can be deferred.
