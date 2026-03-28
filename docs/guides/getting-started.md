---
title: Getting Started
---

# Getting Started

## Installation

### Homebrew (macOS/Linux)

```sh
brew install rundops/tap/dops
```

### Go

```sh
go install github.com/rundops/dops@latest
```

### Docker

```sh
docker pull ghcr.io/rundops/dops:latest
```

### From Source

```sh
git clone https://github.com/rundops/dops.git
cd dops
make install
```

---

## Initialize

```sh
dops init
```

This creates `~/.dops/` with a default configuration and a sample hello-world runbook.

---

## Launch the TUI

```sh
dops
```

Navigate with arrow keys, press Enter to run a runbook, fill in parameters, and confirm.

---

## Launch the Web UI

```sh
dops open
```

Opens a browser-based interface at `http://localhost:3000` with the same catalog, forms, and execution streaming. See the [Web UI guide](web-ui) for details.

---

## Add Your First Catalog

A catalog is a directory of runbooks. Create one:

```sh
mkdir -p ~/.dops/catalogs/my-team/hello-world
```

Create the runbook definition:

```yaml
# ~/.dops/catalogs/my-team/hello-world/runbook.yaml
name: hello-world
version: 1.0.0
description: Say hello
risk_level: low
script: script.sh
parameters:
  - name: greeting
    type: string
    required: true
    description: The greeting message
    scope: runbook
```

Create the script:

```sh
#!/bin/sh
set -eu

GREETING="${GREETING:?greeting is required}"

main() {
  echo "==> Stage 1/1: Hello"
  echo "    ${GREETING}"
  echo ""
  echo "Done"
}

main "$@"
```

Make it executable:

```sh
chmod +x ~/.dops/catalogs/my-team/hello-world/script.sh
```

Register the catalog:

```sh
dops catalog add ~/.dops/catalogs/my-team
```

Launch `dops` — your runbook will appear in the sidebar.

---

## Install a Shared Catalog

Install a catalog from a git repository:

```sh
dops catalog install https://github.com/your-org/ops-runbooks.git
```

Update it later:

```sh
dops catalog update ops-runbooks
```

---

## Run a Runbook from the CLI

Execute a runbook non-interactively:

```sh
dops run my-team.hello-world --param greeting="Hello, world!"
```

---

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `DOPS_HOME` | `~/.dops` | Config and catalog directory |
| `DOPS_NO_ALT_SCREEN` | (unset) | Set to `1` to disable alternate screen |
