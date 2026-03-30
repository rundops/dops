# Clean Code Audit: Design & SOLID Principles

## High Priority

- [ ] **cmd/*.go** — Duplicated bootstrap pattern (config/vault/theme/catalog loading) in 6 places. Create `internal/bootstrap` package with `LoadAppDeps(dopsDir)`.
- [ ] **cmd/*.go, internal/tui, internal/web** — `vault.Vault` used as concrete type everywhere. Define `VaultStore` interface in `domain` package.
- [ ] **internal/tui/app.go** — God struct (1318 lines). Extract execution engine, layout computation, and mouse hit-testing into separate packages.
- [ ] **cmd/run.go:172** — `executeScript` duplicates `executor.ScriptRunner` logic with unused log file bug. Reuse `ScriptRunner.Run()` and drain channels synchronously.
- [ ] **internal/web/server.go:25** — `ServerDeps.Loader` uses concrete `*DiskCatalogLoader`. Change to `CatalogLoader` interface.
- [ ] **cmd/config.go** — Config subcommands use concrete `*FileConfigStore`. Change to `ConfigStore` interface.
- [ ] **cmd/mcp.go:49** — MCP version hardcoded as `"0.1.0"` while binary is `0.10.0`. Pass actual version through.

## Medium Priority

- [ ] **cmd/run.go + internal/tui/wizard/model.go** — Duplicated `saveInputs`/`saveCurrentField` scope-to-keypath logic. Extract shared `SaveParameter()` in `vars` package.
- [ ] **internal/vars/decrypting_resolver.go** — `DecryptingVarResolver` is dead code (never instantiated). Remove or document.
- [ ] **internal/web/api.go:353** — `executionStore` leaks memory (no cleanup/TTL for completed executions). Add eviction.
- [ ] **internal/tui/wizard/model.go** — Wizard directly imports `config` and `vault` for persistence. Should emit a message and let `App` handle persistence.
- [ ] **internal/theme/loader.go:125** — `loadBundled` switch statement. Replace with `map[string][]byte` registry.

## Low Priority

- [ ] **4 packages** — Four `expandHome`/`expandTilde` wrapper functions. Remove wrappers, call `adapters.ExpandHome` directly.
- [ ] **4 packages** — Four `FileSystem` interface definitions. Consider renaming narrow interfaces for clarity (`FileReader`, `DirReader`).
- [ ] **internal/crypto/mask.go** — `MaskSecrets` takes full `*domain.Config` but only needs `domain.Vars`. Narrow the signature.
- [ ] **cmd/init.go:108-129** — `runInit` bypasses `FileSystem` abstraction, uses `os.*` directly. Use injected FS.
- [ ] **internal/adapters/** — Grab-bag package with unrelated types (`FileSystem` + `LogWriter` + `ExpandHome`). Consider splitting or renaming.
- [ ] **internal/domain/config.go** — `Catalog` struct serves dual duty (config entry + runtime data). Consider separating if it grows.
