---
layout: default
title: Configuration
nav_order: 3
parent: Reference
---

# Configuration

dops stores its configuration in `~/.dops/config.json`. Override the base directory with the `DOPS_HOME` environment variable.

---

## Directory Layout

```
~/.dops/
├── config.json         # Main configuration
├── keys/               # Encryption keys (age)
│   └── dops.key
├── themes/             # Custom theme overrides
│   └── mytheme.json
└── catalogs/           # Runbook catalogs
    ├── default/
    └── infra/
```

---

## Config Keys

| Key | Type | Description |
|-----|------|-------------|
| `theme` | string | Active theme name (default: `tokyonight`) |
| `defaults.max_risk_level` | string | Maximum risk level to load (`low`, `medium`, `high`, `critical`) |
| `catalogs` | array | List of catalog entries with `name`, `path`, `active` |
| `vars.global.<name>` | string | Global saved parameter values |
| `vars.catalog.<cat>.<name>` | string | Catalog-scoped saved values |
| `vars.catalog.<cat>.runbooks.<rb>.<name>` | string | Runbook-scoped saved values |

---

## Themes

dops ships with the **Tokyo Night** theme. To customize, create a theme file in `~/.dops/themes/`:

```json
{
  "name": "my-theme",
  "dark": {
    "defs": {
      "bg": "#1a1b26",
      "fg": "#c0caf5",
      "blue": "#7aa2f7",
      "green": "#9ece6a",
      "orange": "#ff9e64",
      "red": "#f7768e"
    },
    "tokens": {
      "background": "bg",
      "backgroundPanel": "#1f2335",
      "backgroundElement": "#292e42",
      "text": "fg",
      "textMuted": "#565f89",
      "primary": "blue",
      "border": "#3b4261",
      "borderActive": "blue",
      "success": "green",
      "warning": "orange",
      "error": "red",
      "risk.low": "green",
      "risk.medium": "orange",
      "risk.high": "red",
      "risk.critical": "#db4b4b"
    }
  }
}
```

Activate it:

```sh
dops config set theme=my-theme
```

dops auto-detects dark/light terminal background and selects the appropriate variant.

---

## Encryption

Secret parameters are encrypted at rest using [age](https://age-encryption.org/) (X25519). The key is auto-generated at `~/.dops/keys/dops.key` on first use.

Encrypted values in `config.json` are prefixed with `age1-`. They are automatically decrypted when passed to scripts and masked with `****` in all display contexts (TUI, MCP, `config list`).

---

## Shell Completion

```sh
# Bash
dops completion bash > /etc/bash_completion.d/dops

# Zsh
dops completion zsh > "${fpath[1]}/_dops"

# Fish
dops completion fish > ~/.config/fish/completions/dops.fish

# PowerShell
dops completion powershell > dops.ps1
```
