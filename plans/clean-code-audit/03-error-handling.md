# Clean Code Audit: Error Handling

## High Priority (bugs or data loss)

- [x] **internal/executor/demo.go:39** — `crypto/rand.Int` error discarded with `_`. If it fails, `jitter` is nil → panic on `jitter.Int64()`. Add fallback.
- [x] **cmd/run.go:62-68** — Original error from `FindByID`/`FindByAlias` is discarded and replaced with generic message. Wrap the actual error.
- [x] **internal/vars/decrypting_resolver.go:23** — Decryption failure silently passes ciphertext as a parameter value. Log warning or return error.

## Missing Error Wrapping (bare `return err`)

- [x] **cmd/config.go:65** — `config.Set` error not wrapped
- [x] **cmd/config.go:88** — `store.Load` error not wrapped
- [x] **cmd/config.go:98** — `config.Get` error not wrapped
- [x] **cmd/config.go:115** — `store.Load` error not wrapped
- [x] **cmd/config.go:127** — `config.Unset` error not wrapped
- [x] **cmd/config.go:141** — `store.Load` error not wrapped
- [x] **cmd/catalog.go:91** — `ValidateDisplayName` error not wrapped
- [x] **cmd/catalog.go:97** — `filepath.Abs` error not wrapped
- [x] **cmd/catalog.go:102** — `loadConfig` error not wrapped
- [x] **cmd/catalog.go:184** — `ValidateDisplayName` error not wrapped (2nd occurrence)
- [x] **cmd/catalog.go:287** — `ValidateDisplayName` error not wrapped (3rd occurrence)
- [ ] **cmd/run.go:165** — `config.Set` error not wrapped
- [ ] **cmd/open.go:108** — `srv.Start` error not wrapped
- [ ] **internal/mcp/server.go:216** — `srv.Connect` error not wrapped
- [ ] **internal/mcp/watcher.go:30** — `fsnotify.NewWatcher` error not wrapped
- [ ] **internal/update/check.go:126** — `client.Get` error not wrapped
- [ ] **internal/update/check.go:136** — `json.Decode` error not wrapped
- [ ] **internal/mcp/resources.go:37** — `json.MarshalIndent` error not wrapped
- [ ] **internal/mcp/resources.go:57** — `json.MarshalIndent` error not wrapped
- [ ] **internal/crypto/age.go:28** — `loadOrCreateIdentity` error not wrapped

## Inconsistent Patterns

- [ ] **cmd/config.go:155-165** — `json.Marshal` errors silently ignored in config list. Show warning or return error.
- [ ] **internal/mcp/schema.go:105** — `json.Marshal` error discarded with `_`. Handle or return `(json.RawMessage, error)`.

## Acceptable (documented, no change needed)

- internal/mcp/watcher.go:78 — `_ = err` documented (MCP stdio constraint)
- internal/executor/script.go:68-69 — pipe close errors (signaling EOF)
- internal/adapters/log.go:54-55,60-64 — best-effort logging writes
- internal/update/check.go:116-117 — cache write errors (non-critical)
- internal/web/handler.go:50 — `http.ResponseWriter.Write` error (standard Go HTTP pattern)
