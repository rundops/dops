# Web UI (`dops open`)

## Overview

Add a `dops open` command that launches a local web server and opens a browser-based UI for browsing and executing runbooks. Built with VueJS 3, TailwindCSS v4, and shadcn-vue. The web UI provides an alternative to the TUI for users who prefer a graphical interface or need to share runbook access with team members on a local network.

## Requirements

- `dops open` must start an HTTP server on `localhost:3000` (configurable via `--port`)
- The server must auto-open the default browser (overridable with `--no-browser`)
- The web UI must display the same catalog/runbook tree as the TUI sidebar
- Users must be able to execute runbooks with parameter forms equivalent to the TUI wizard
- Execution output must stream to the browser in real-time (WebSocket or SSE)
- The web UI must respect the same risk confirmation gates as the TUI
- `Ctrl+C` in the terminal must gracefully shut down the server
- The web UI frontend must be embedded in the Go binary (no external files needed at runtime)
- The web UI must use the existing theme system for color theming

## Acceptance Criteria

- [ ] `dops open` starts HTTP server and opens browser
- [ ] `dops open --port 8080` uses custom port
- [ ] `dops open --no-browser` starts server without opening browser
- [ ] Catalog tree renders with collapsible sections
- [ ] Runbook detail view shows name, description, risk level, parameters
- [ ] Parameter form renders all field types (text, select, multi_select, boolean, secret, file_path)
- [ ] Execution streams output in real-time to browser
- [ ] Risk confirmation shown before execution (matching TUI behavior)
- [ ] Ctrl+C in terminal stops the server gracefully
- [ ] Frontend assets embedded in Go binary via `//go:embed`
- [ ] Works without Node.js installed at runtime

## Error Cases

- Port already in use: error message with suggestion to use `--port`
- Browser fails to open: log warning, server still runs
- WebSocket disconnect during execution: execution continues, reconnect shows latest output
- Runbook execution fails: error displayed in UI with exit code

## Out of Scope

- Authentication/authorization (localhost only for v0.10.0)
- Remote access / TLS
- Editing runbook YAML from the web UI
- Creating new runbooks from the web UI
- Persistent execution history / logs in the web UI

## Research Required

Before implementation, research and document:
1. **VueJS 3 + Go embed**: Best practices for embedding a Vue SPA in a Go binary
2. **shadcn-vue**: Component library maturity, available components, theming approach
3. **TailwindCSS v4**: Build pipeline for embedded assets (pre-built vs runtime)
4. **SSE vs WebSocket**: Trade-offs for streaming execution output
5. **Build pipeline**: How to build the Vue app and embed it in `go build` without requiring Node.js for contributors who only touch Go code
