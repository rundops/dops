---
layout: default
title: Getting Started
nav_order: 2
parent: Guides
---

# Getting Started

## Installation

### Homebrew (macOS/Linux)

```sh
brew install jacobhuemmer/tap/dops
```

### Go

```sh
go install github.com/jacobhuemmer/dops-cli@latest
```

### Docker

```sh
docker pull ghcr.io/jacobhuemmer/dops-cli:latest
```

### From Source

```sh
git clone https://github.com/jacobhuemmer/dops-cli.git
cd dops-cli
make install
```

---

## First Run

Launch the TUI:

```sh
dops
```

On first run, dops creates `~/.dops/` with a default configuration. If no catalogs are configured, the sidebar will be empty.

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

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `DOPS_HOME` | `~/.dops` | Config and catalog directory |
| `DOPS_NO_ALT_SCREEN` | (unset) | Set to `1` to disable alternate screen |
