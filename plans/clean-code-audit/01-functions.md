# Clean Code Audit: Functions & Complexity

## Long Functions (>30 lines)

- [ ] **cmd/catalog.go:160** — `newCatalogInstallCmd` RunE (~75 lines). Extract `cloneRepo()`, `registerCatalog()`.
- [ ] **cmd/catalog.go:257** — `newCatalogUpdateCmd` RunE (~90 lines). Extract `updateDisplayName()`, `pullOrCheckout()`, `updateRiskPolicy()`.
- [ ] **cmd/run.go:22** — `newRunCmd` RunE (~80 lines). Extract `loadRunDeps()`, `resolveRunbook()`.
- [ ] **cmd/root.go:52** — `launchTUI` (~73 lines). Extract shared `loadDeps(dopsDir)`.
- [ ] **cmd/open.go:45** — `runWebUI` (~84 lines). Reuse shared `loadDeps()` with `launchTUI`.
- [ ] **internal/mcp/tools.go:32** — `HandleToolCall` (~86 lines). Extract `prepareExecution()`, `collectResult()`.
- [x] **internal/mcp/schema.go:12** — `RunbookToInputSchema` (~95 lines). Extract `paramToSchemaProperty()`.
- [ ] **internal/tui/app.go:695** — `viewNormal` (~102 lines). Extract `renderSidebar()`, `renderRightPanel()`.
- [ ] **internal/tui/app.go:1006** — `applySelectionHighlight` (~62 lines). Extract `highlightLine()`.
- [ ] **internal/tui/output/model.go:556** — `renderLogSection` (~71 lines). Extract `renderSearchBar()`.
- [ ] **internal/tui/sidebar/model.go:413** — `buildLines` (~86 lines). Extract `renderEntry()`.
- [ ] **internal/tui/wizard/model.go:148** — `initField` (~81 lines). Extract per-type init helpers.
- [x] **internal/theme/loader.go:125** — `loadBundled` (~45 lines). Replace switch with `map[string][]byte`.
- [x] **internal/executor/demo.go** — `runbookOutputs` map (~370 lines). Move to `demo_data.go`.
- [ ] **internal/cli/help.go:55** — `renderHelp` (~83 lines). Extract `renderHelpHeader()`, `renderHelpUsage()`.

## Too Many Parameters (>3-4)

- [x] **internal/mcp/tools.go:32** — `HandleToolCall` has 7 params. Introduce `ToolCallRequest` struct.
- [x] **internal/tui/app.go:1006** — `applySelectionHighlight` has 7 params. Collapse 4 bounds ints into `Bounds` struct.
- [x] **internal/tui/confirm/model.go:25** — `confirm.New` has 5 params. Group into `confirm.Params`.
- [ ] **internal/metadata/view.go:33** — `metadata.Render` has 5 params. Group into `RenderParams`.
- [x] **internal/tui/output/model.go:645** — `renderScrollbar` has 5+ params. Use `scrollbarParams` struct.
- [ ] **cmd/run.go:140** — `saveInputs` has 5 params. Group `cfg`+`vlt` into a struct.

## Boolean Flag Parameters

- [x] **cmd/open.go:45** — `runWebUI(dopsDir, port, noBrowser, demo)`. Replace with `WebUIOptions` struct.
- [ ] **internal/tui/wizard/model.go:60** — `NewWithOptions(..., promptAll bool)`. Use `WizardConfig` struct.
- [ ] **internal/theme/resolver.go:31** — `resolveToken(..., isDark bool)`. Use `ThemeMode` enum.

## Deep Nesting (>2-3 levels)

- [ ] **internal/catalog/loader.go:90** — `buildAliasIndex` — 4 levels. Extract `registerAlias()`.
- [ ] **internal/tui/wizard/model.go:313** — `updateTextInput` — 3+ levels. Extract `validateTextValue()`.
- [ ] **cmd/catalog.go:302** — `newCatalogUpdateCmd` git operations — 4 levels. Extract `updateGitRef()`, `pullLatest()`.
