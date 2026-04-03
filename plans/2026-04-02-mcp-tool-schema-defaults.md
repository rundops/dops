# Plan: MCP Tool Schema Defaults

**Spec:** v0.11.0 Feature 1
**Effort:** 2-3 hours
**Files:** `internal/mcp/server.go`, `internal/mcp/server_test.go`

## Problem

`registerTools()` passes an empty map to `RunbookToInputSchema()`. Tool
schemas advertise parameters but never show saved defaults from the vault.

## TDD Steps

### Red: Write failing tests

**File:** `internal/mcp/server_test.go`

1. Test `TestRegisterTools_PassesResolvedVars` — create a ServerConfig with
   a Config that has saved vars (global + catalog-scoped). Verify that
   `RunbookToInputSchema` receives a non-empty resolved map and the
   resulting tool schema contains `default` fields for parameters with
   saved values.

2. Test `TestRegisterTools_SecretParamsOmitDefault` — parameter with
   `secret: true` has a saved value. Verify the schema does NOT include
   a `default` field for that parameter (avoids leaking secrets).

3. Test `TestRegisterTools_NoSavedVars_NoDefaults` — empty vault. Verify
   schema has no `default` fields (same as current behavior).

### Green: Implement

**File:** `internal/mcp/server.go` — `registerTools()`

Replace:
```go
resolved := make(map[string]string) // TODO: pass resolved vars
```

With:
```go
resolver := vars.NewDefaultResolver()
resolved := resolver.Resolve(s.cfg, cat.Name, rb.Name, rb.Parameters)
```

**File:** `internal/mcp/tools.go` or `internal/mcp/schema.go`

In `RunbookToInputSchema`, when building the JSON Schema property for a
parameter:
- If `resolved[param.Name]` is non-empty AND `!param.Secret`, set
  `"default": resolved[param.Name]` on the schema property.

### Refactor

- Remove the TODO comment
- Verify `RunbookToInputSchema` signature already accepts resolved map
  (it does: `func RunbookToInputSchema(rb domain.Runbook, resolved map[string]string)`)

## Verification

```
go test ./internal/mcp/... -v -run TestRegisterTools
go build ./...
go vet ./...
```
