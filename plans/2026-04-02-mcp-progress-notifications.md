# Plan: MCP Execution Progress Notifications

**Spec:** v0.11.0 Feature 2
**Effort:** 2-3 hours
**Files:** `internal/mcp/server.go`, `internal/mcp/tools.go`, `internal/mcp/server_test.go`

## Problem

`makeToolHandler` passes `OnProgress: nil`. Execution runs silently — the
AI agent gets no feedback until the final ToolResult. The `ProgressWriter`
and `ProgressCallback` type already exist in `progress.go` but aren't wired
to MCP notifications.

## TDD Steps

### Red: Write failing tests

**File:** `internal/mcp/server_test.go`

1. Test `TestToolHandler_EmitsProgress` — execute a tool via the handler.
   Capture progress notifications. Verify at least one progress
   notification is emitted during execution with line count > 0.

2. Test `TestToolHandler_ProgressRateLimited` — execute a tool that
   produces many lines quickly. Verify that progress notifications are
   rate-limited (no more than 1 per second between notifications).

3. Test `TestToolHandler_NilProgress_StillWorks` — verify that a nil
   OnProgress callback doesn't panic (existing behavior preserved).

**File:** `internal/mcp/tools_test.go` (or existing test file)

4. Test `TestHandleToolCall_InvokesOnProgress` — call HandleToolCall with
   a non-nil OnProgress. Verify callback was invoked with line count and
   output chunk.

### Green: Implement

**File:** `internal/mcp/server.go` — `makeToolHandler()`

Wire an `OnProgress` callback that uses the MCP SDK's notification
mechanism:

```go
OnProgress: func(chunk string, linesSoFar int) {
    // Rate-limit: skip if <1s since last notification
    // Send MCP progress notification with linesSoFar as progress token
},
```

Rate-limiting approach: track `lastProgressAt time.Time` in the closure.
Skip notification if `time.Since(lastProgressAt) < time.Second`.

**File:** `internal/mcp/tools.go` — `HandleToolCall()`

The OnProgress callback is already threaded through to ProgressWriter.
Verify it flows: `HandleToolCall → ProgressWriter.Write → callback`.

### Refactor

- Remove the `// TODO: wire progress notifications` comment
- Extract rate-limiting into a helper if the closure gets complex

## Verification

```
go test ./internal/mcp/... -v -run TestToolHandler
go test ./internal/mcp/... -v -run TestHandleToolCall
go build ./...
go vet ./...
```
