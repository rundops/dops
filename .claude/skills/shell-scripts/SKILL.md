---
name: shell-scripts
description: "POSIX-compatible shell scripting guide based on Google Shell Style Guide. Use when writing, editing, or reviewing shell scripts (.sh), bash scripts, automation scripts, or any executable script files. Triggers on #!/bin/sh, #!/bin/bash, .sh file creation, shell function definitions, or shell scripting questions."
user-invocable: false
---

# Shell Script Style Guide

Based on https://google.github.io/styleguide/shellguide.html with POSIX compatibility as the default.

## Shell Selection

**Default to POSIX sh** (`#!/bin/sh`) for cross-platform compatibility (Linux + macOS).

Only use bash/zsh when you need specific features. If you do, add a comment explaining why:

```sh
#!/bin/bash
# Requires bash for: associative arrays, process substitution, [[ ]]
```

### When to use each shell

| Shell | Shebang | Use when |
|-------|---------|----------|
| **POSIX sh** | `#!/bin/sh` | Default. Simple scripts, CI/CD, Docker, automation |
| **Bash** | `#!/bin/bash` | Need arrays, `[[ ]]`, `${var//pat/rep}`, process substitution |
| **Zsh** | `#!/bin/zsh` | Need zsh-specific features (glob qualifiers, associative arrays with special syntax) |

### POSIX vs Bash quick reference

| Feature | POSIX (sh) | Bash |
|---------|-----------|------|
| Test | `[ -n "$var" ]` | `[[ -n "$var" ]]` |
| String replace | `echo "$var" \| sed 's/a/b/'` | `${var//a/b}` |
| Arrays | Not available | `arr=(a b c)` |
| Local vars | `var=value` (function scope) | `local var=value` |
| Arithmetic | `$(( x + 1 ))` | `$(( x + 1 ))` ✓ same |
| Command check | `command -v foo` | `command -v foo` ✓ same |
| Here strings | Not available | `<<< "string"` |
| Pipefail | Not available | `set -o pipefail` |
| Read -r | `read -r var` ✓ | `read -r var` ✓ same |

## File Header

```sh
#!/bin/sh
#
# Brief description of what the script does.
#
# Usage:
#   ./script.sh [options]
#
```

## Error Handling

```sh
#!/bin/sh
set -eu

# Trap for cleanup on exit
cleanup() {
  rm -f "${TMPFILE:-}"
}
trap cleanup EXIT
```

**Do NOT use `set -o pipefail`** — it's not POSIX. If you need pipefail, switch to bash and document why.

## Variables

```sh
# Use uppercase for exported/environment variables
ENDPOINT="${ENDPOINT:?endpoint is required}"
DRY_RUN="${DRY_RUN:-false}"
TIMEOUT="${TIMEOUT:-30}"

# Use lowercase for local script variables
output_dir="/tmp/results"
log_file="${output_dir}/run.log"
```

**Always quote variables**: `"${var}"` not `$var`

**Use `${var:?message}`** for required variables — fails with message if unset/empty.

**Use `${var:-default}`** for optional variables with defaults.

## Functions

```sh
# Lowercase with underscores. Short and focused.
check_dependency() {
  if ! command -v "$1" > /dev/null 2>&1; then
    echo "Error: $1 is not installed" >&2
    return 1
  fi
}

# Document complex functions
#
# Validates the input configuration file.
#
# Args:
#   $1 - Path to config file
# Returns:
#   0 on success, 1 on validation error
validate_config() {
  config_file="$1"

  if [ ! -f "${config_file}" ]; then
    echo "Error: config file not found: ${config_file}" >&2
    return 1
  fi
}

# Put main at the bottom
main() {
  check_dependency "curl"
  check_dependency "jq"
  validate_config "${CONFIG_FILE}"
  # ...
}

main "$@"
```

## Conditionals

```sh
# POSIX test — use [ ] not [[ ]]
if [ -n "${var}" ]; then
  echo "var is set"
fi

if [ "${status}" -eq 0 ]; then
  echo "success"
fi

# String comparison
if [ "${env}" = "production" ]; then
  echo "deploying to prod"
fi

# File tests
if [ -f "${path}" ]; then echo "file exists"; fi
if [ -d "${path}" ]; then echo "directory exists"; fi
if [ -x "${path}" ]; then echo "is executable"; fi

# Negation
if [ ! -f "${path}" ]; then echo "file missing"; fi

# AND / OR
if [ -n "${var}" ] && [ -f "${path}" ]; then
  echo "both conditions met"
fi
```

## Loops

```sh
# Iterate over arguments
for arg in "$@"; do
  echo "Processing: ${arg}"
done

# Iterate over command output
for file in $(find /tmp -name "*.log" -type f); do
  echo "Found: ${file}"
done

# While loop with read (POSIX safe)
while IFS= read -r line; do
  echo "Line: ${line}"
done < "${input_file}"

# Counter loop
i=0
while [ "${i}" -lt 10 ]; do
  echo "Iteration: ${i}"
  i=$((i + 1))
done
```

## Output

```sh
# Normal output to stdout
echo "Processing..."

# Errors to stderr
echo "Error: file not found" >&2

# Formatted output — use printf for portability
printf "%-20s %s\n" "Name:" "${name}"
printf "Status: %d\n" "${exit_code}"

# DO NOT use echo -e (not portable)
# Bad:  echo -e "line1\nline2"
# Good: printf "line1\nline2\n"
```

## Temporary Files

```sh
# Use mktemp for secure temp files
TMPFILE=$(mktemp)
TMPDIR=$(mktemp -d)

# Always clean up
trap 'rm -f "${TMPFILE}"; rm -rf "${TMPDIR}"' EXIT
```

## Command Substitution

```sh
# Use $() not backticks
result=$(date +%Y-%m-%d)
count=$(wc -l < "${file}")

# Nested substitution
path=$(dirname $(readlink -f "$0"))
```

## Arithmetic

```sh
# POSIX arithmetic
total=$((count + 1))
half=$((total / 2))

# Comparison in conditions
if [ "${count}" -gt 10 ]; then
  echo "many items"
fi
```

## Common Patterns

### Check if command exists
```sh
if ! command -v docker > /dev/null 2>&1; then
  echo "Error: docker is required" >&2
  exit 1
fi
```

### Parse simple flags
```sh
verbose=false
dry_run=false

while [ $# -gt 0 ]; do
  case "$1" in
    -v|--verbose) verbose=true ;;
    -n|--dry-run) dry_run=true ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
  shift
done
```

### Retry with backoff
```sh
retry() {
  max_attempts="$1"
  shift
  attempt=1
  while [ "${attempt}" -le "${max_attempts}" ]; do
    if "$@"; then
      return 0
    fi
    echo "Attempt ${attempt}/${max_attempts} failed, retrying..." >&2
    sleep "$((attempt * 2))"
    attempt=$((attempt + 1))
  done
  return 1
}

retry 3 curl -sf "https://example.com/health"
```

### JSON processing with jq
```sh
# Extract a field
name=$(echo "${json}" | jq -r '.name')

# Check if jq is available, fall back to grep
if command -v jq > /dev/null 2>&1; then
  status=$(echo "${response}" | jq -r '.status')
else
  status=$(echo "${response}" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
fi
```

## Things to Avoid

| Avoid | Use instead | Why |
|-------|------------|-----|
| `#!/bin/bash` (default) | `#!/bin/sh` | POSIX portability |
| `[[ ]]` | `[ ]` | Not POSIX |
| `echo -e` | `printf` | Not portable |
| `which` | `command -v` | Not POSIX guaranteed |
| `$var` unquoted | `"${var}"` | Word splitting, globbing |
| Backticks `` `cmd` `` | `$(cmd)` | Nesting, readability |
| `set -o pipefail` | error check per pipe | Not POSIX |
| `local -r` | `local` or plain var | `-r` not POSIX with local |
| `{a,b,c}` | explicit list | Brace expansion not POSIX |
| `source file` | `. file` | `source` not POSIX |
| `function foo()` | `foo()` | `function` keyword not POSIX |
