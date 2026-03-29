---
name: demo-recording
description: Record terminal demo GIFs for README and docs using asciinema + agg. Use when creating or updating demo GIFs for interactive sessions (e.g., Claude Code MCP demos). Triggers on demo GIF creation, mcp-demo recording, or any request to record a terminal session as a GIF.
user-invocable: true
---

# Demo GIF Recording with asciinema + agg

Record interactive terminal sessions as GIFs that match the project's visual style. Use this for demos that can't be scripted with VHS (e.g., Claude Code, interactive AI agents).

## Tools

| Tool | Use For |
|---|---|
| **VHS** | Scripted TUI demos — deterministic flows (`tapes/*.tape`), `make screenshots` |
| **asciinema + agg** | Interactive session recordings — Claude Code, MCP demos, anything with non-deterministic output |
| **Playwright + ffmpeg** | Web UI demos — browser recordings (`make web-demo`) |

## Prerequisites

```bash
brew install asciinema agg
```

## Recording Workflow

### Step 1: Resize terminal to landscape

```bash
printf '\e[8;30;120t'
```

This gives a 120x30 (cols x rows) layout matching the web-demo aspect ratio.

### Step 2: Record with clean prompt

```bash
asciinema rec -c "PS1='$ ' zsh --no-rcs" <name>.cast
```

- `PS1='$ '` — clean dollar-sign prompt, no username/path/git info
- `--no-rcs` — prevents zsh from loading profile, keeps prompt clean

### Step 3: Perform the demo inside the recording

Run whatever interactive session you want to capture, then `exit` to stop.

### Step 4: Trim dead time (optional)

```bash
asciinema cut --start <seconds> --end <seconds> <name>.cast -o <name>-trimmed.cast
```

### Step 5: Convert to GIF

```bash
agg --font-size 28 --cols 120 --rows 30 --theme github-dark <name>.cast assets/<name>.gif
```

## Settings Reference

These settings ensure visual consistency across all project demos:

| Setting | Value | Reason |
|---|---|---|
| **Cols** | 120 | Wide landscape ratio matching web-demo |
| **Rows** | 30 | Compact height, avoids wasted vertical space |
| **Font size** | 28 | Readable when embedded at 900px width in README |
| **Theme** | github-dark | Matches docs site dark theme |
| **Prompt** | `$ ` | Clean, no user/path/git decorations |

## Adding Demos to README and Docs

### README.md pattern

```html
<div align="center">
  <em>Description of the demo:</em> <code>command</code>
  <br /><br />
  <img src="assets/<name>-demo.gif" alt="dops <name> demo" width="900" />
</div>
```

### Docs index.md pattern

```html
<div class="demo-section">
  <h2>Section Title</h2>
  <p>Description. Run with <code>command</code>.</p>
  <img src="https://raw.githubusercontent.com/rundops/dops/main/assets/<name>-demo.gif" alt="dops <name> demo" />
</div>
```

Note: Docs uses raw GitHub URLs for GIFs, not relative paths.

## Demo order on homepage

1. Terminal UI (`demo.gif`)
2. Web UI (`web-demo.gif`)
3. MCP Server (`mcp-demo.gif`)
