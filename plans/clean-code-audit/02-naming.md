# Clean Code Audit: Naming Conventions

## Cryptic Abbreviations

- [x] **cmd/*.go** — `vlt` → `vault` — SKIPPED: `vault` is the package name, `vlt` avoids shadowing (idiomatic Go)
- [x] **cmd/root.go, cmd/open.go, internal/theme/loader.go** — `tf` → `themeFile`
- [ ] **Multiple files** — `rb` → `runbook` in functions >10 lines (keep `rb` in short methods)
- [ ] **Multiple files** — `cat` → `catalog` in functions >10 lines
- [x] **internal/tui/app.go:75** — `sw` → `sidebarW` (consistent with `rightW`, `contentW`)
- [x] **internal/tui/app.go, internal/mcp/tools.go** — `lw` → `logWriter`
- [x] **internal/vars/decrypting_resolver.go, internal/vault/vault.go** — `enc` → `encrypter`
- [x] **internal/tui/*.go** — `b` → `sb` for `strings.Builder` (standardize across codebase)
- [x] **internal/tui/output/model.go** — `cw` → `contentWidth`
- [ ] **cmd/run.go** — `p` → `param` in range loops spanning 10+ lines
- [x] **internal/mcp/server.go:274** — `mkerr` → `toolError`

## Inconsistent Naming

- [ ] **4 packages** — `expandHome`/`expandTilde`/`expandPath` — 3 names for the same operation. Remove all private wrappers, call `adapters.ExpandHome` directly.
- [ ] **cmd/*.go** — `catpkg` import alias used inconsistently. Use unaliased `catalog` where no conflict.
- [ ] **cmd/mcp.go** — `mcppkg` awkward alias. Use `mcp` unaliased or `dopmcp`.

## Stuttering Type Names

- [ ] **internal/tui/wizard/messages.go** — `WizardSubmitMsg` → `SubmitMsg` (package already says wizard)
- [ ] **internal/tui/wizard/messages.go** — `WizardCancelMsg` → `CancelMsg`
- [ ] **internal/tui/confirm/messages.go** — `ConfirmAcceptMsg` → `AcceptMsg`
- [ ] **internal/tui/confirm/messages.go** — `ConfirmCancelMsg` → `CancelMsg`
- [ ] **internal/tui/palette/model.go:10** — `PaletteCommand` → `Command`

## Generic Names

- [ ] **internal/update/check.go:29** — `Result` → `CheckResult`
- [ ] **cmd/init.go:148** — `check()` → `checkmark()` (returns a glyph, not a boolean)
- [ ] **internal/vault/vault.go:13** — `currentVersion` → `vaultFormatVersion`
- [ ] **internal/mcp/tools.go:18** — `maxOutputLines` → `maxToolOutputLines`
- [ ] **internal/mcp/progress.go:10** — `defaultBatchSize` → `defaultProgressBatchSize`

## Const Grouping

- [ ] **internal/tui/app.go:52-67** — Mixed `layout*`, `overlay*`, `sidebar*`, `copyFlash*` constants in one block. Group by concern.
