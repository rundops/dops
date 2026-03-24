# MCP Server Implementation Plan

## Date: 2026-03-24

## Context

dops needs an MCP server so AI agents (Claude Code, etc.) can discover and execute runbooks programmatically. The legacy has a production-ready MCP implementation using `github.com/modelcontextprotocol/go-sdk`.

## Design

### Transport
- **stdio** (default) — for Claude Code integration via `dops mcp serve`
- **HTTP** — via `dops mcp serve --transport http --port 8080` with SSE streaming

### Tools
Each runbook becomes an MCP tool:
- **Tool name**: runbook ID (e.g. `default.check-health`)
- **Tool description**: `Name: Description [risk: level]`
- **Input schema**: JSON Schema generated from runbook parameters
- **Tool handler**: resolves inputs, executes script, returns output

### Resources
- `dops://catalog` — JSON array of all runbook summaries
- `dops://catalog/{id}` — full runbook detail by ID

### Output Format (Token Efficiency)
Tool results use smart truncation instead of full output:
```json
{
  "exit_code": 0,
  "duration": "2.3s",
  "log_path": "/tmp/2026.03.24-...-default-check-health.log",
  "output_lines": 45,
  "output": "[last 50 lines of stdout/stderr]",
  "summary": "All health checks passed."
}
```
- Last 50 lines included in response (configurable)
- Full output available via `log_path`
- HTTP transport uses standard gzip via `Content-Encoding` header

### Risk Confirmation
- High risk: requires `_confirm_id` parameter matching runbook ID
- Critical risk: requires `_confirm_word` parameter = "CONFIRM"
- Synthetic confirmation fields added to tool input schema

### Progress Streaming
- Lines batched (every 5 lines) and sent as MCP progress notifications
- Allows agents to see execution progress in real-time

### Catalog Hot-Reload
- `fsnotify` watches catalog directories for `.yaml` changes
- Debounced reload (500ms) swaps tools/resources without restart

## Files to Create

| File | Purpose |
|---|---|
| `internal/mcp/server.go` | MCP server setup, tool/resource registration, reload |
| `internal/mcp/tools.go` | Tool handler: input resolution, execution, result formatting |
| `internal/mcp/resources.go` | Catalog listing and detail resources |
| `internal/mcp/schema.go` | JSON Schema generation from runbook parameters |
| `internal/mcp/progress.go` | Progress writer: batches lines, sends notifications |
| `internal/mcp/watcher.go` | fsnotify catalog watcher with debounced reload |
| `cmd/mcp.go` | Cobra command: `dops mcp serve` and `dops mcp tools` |

## Files to Modify

| File | Changes |
|---|---|
| `cmd/root.go` | Register `newMCPCmd(dopsDir)` |
| `go.mod` | Add `github.com/modelcontextprotocol/go-sdk`, `github.com/fsnotify/fsnotify` |

## Step 1: Add Dependencies

```bash
go get github.com/modelcontextprotocol/go-sdk
go get github.com/fsnotify/fsnotify
```

## Step 2: Schema Generation (`internal/mcp/schema.go`)

Convert `domain.Parameter` to JSON Schema:
- `string` → `{"type": "string"}`
- `integer` → `{"type": "integer"}`
- `boolean` → `{"type": "boolean"}`
- `select` → `{"type": "string", "enum": [options]}`
- `multi_select` → `{"type": "array", "items": {"type": "string", "enum": [options]}}`
- `file_path` → `{"type": "string", "description": "file path"}`
- `resource_id` → `{"type": "string", "description": "resource identifier"}`

Add synthetic `_confirm_id` / `_confirm_word` for high/critical risk.

Required params marked in schema `required` array. Saved params marked optional with description annotation.

## Step 3: Tools (`internal/mcp/tools.go`)

`HandleToolCall(ctx, catalog, runner, varsResolver, store, cfg, req)`:
1. Find runbook by tool name (= runbook ID)
2. Parse JSON arguments from MCP request
3. Validate risk confirmation params
4. Resolve saved vars, merge with provided args
5. Execute script via `Runner.Run(ctx, scriptPath, env)`
6. Collect output with progress streaming
7. Return truncated result with metadata

## Step 4: Resources (`internal/mcp/resources.go`)

`HandleCatalogList(catalogs)` → JSON array of summaries
`HandleCatalogItem(catalogs, id)` → full runbook JSON

## Step 5: Progress (`internal/mcp/progress.go`)

`progressWriter` wraps `io.Writer`, batches complete lines (default 5), calls notification callback.

## Step 6: Server (`internal/mcp/server.go`)

```go
func NewMCPServer(cfg ServerConfig) *Server {
    srv := mcp.NewServer(&mcp.Implementation{Name: "dops", Version: version})
    registerTools(srv, catalogs, runner, ...)
    registerResources(srv, catalogs)
    return srv
}

func (s *Server) Serve(transport string, port int) error {
    switch transport {
    case "stdio":
        return s.srv.Run(ctx, mcp.StdioTransport{})
    case "http":
        handler := mcp.NewStreamableHTTPHandler(s.srv)
        // Add gzip middleware
        return http.ListenAndServe(":"+port, gzipMiddleware(handler))
    }
}
```

## Step 7: Watcher (`internal/mcp/watcher.go`)

Watch catalog dirs, debounce 500ms, call `server.Reload()`.

## Step 8: CLI (`cmd/mcp.go`)

```
dops mcp serve [--transport stdio|http] [--port 8080] [--allow-risk critical]
dops mcp tools
```

## Step 9: HTTP Gzip Middleware

Standard `net/http` middleware that checks `Accept-Encoding: gzip` and wraps `ResponseWriter` with `gzip.NewWriter`.

## Verification

1. `go build ./...` — compiles
2. `go test ./...` — all tests pass
3. Manual: `echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | ./dops mcp serve` → lists tools
4. Manual: `echo '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"default.echo","arguments":{}},"id":2}' | ./dops mcp serve` → executes
5. Claude Code: add to `.claude/settings.json` as MCP server, verify tool discovery
