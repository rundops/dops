---
name: mutation-testing
description: Mutation testing with Gremlins for Go. Use when assessing test quality, finding weak tests, or when the user asks about mutation testing. Triggers on "mutation test", "gremlins", "test quality", "test effectiveness", or after completing a feature to verify test strength.
user-invocable: true
---

# Mutation Testing with Gremlins

Gremlins tests your tests. It introduces small bugs (mutants) into your code and checks whether your test suite catches them. Surviving mutants = weak tests.

## Install

```bash
brew tap go-gremlins/tap && brew install gremlins
# or
go install github.com/go-gremlins/gremlins/cmd/gremlins@latest
```

## Quick Start

```bash
# Dry run — see what would be mutated, no testing
gremlins unleash --dry-run

# Run mutation testing on the full module
gremlins unleash

# Run only on specific packages
gremlins unleash --coverpkg ./internal/vars/...

# Run only on code changed since main (fast, use in PRs)
gremlins unleash --diff main
```

## Reading Results

| Status | Meaning | Action |
|---|---|---|
| **KILLED** | Tests caught the mutation | Good. No action needed. |
| **LIVED** | Tests did NOT catch the mutation | Bad. Write a better test. |
| **NOT COVERED** | No test coverage at mutation site | Write tests for this code. |
| **TIMED OUT** | Tests hung on the mutant | Usually good (mutation broke a loop). |
| **NOT VIABLE** | Mutation caused compile error | Ignore. |

**Key metrics:**
- **Efficacy** = KILLED / (KILLED + LIVED) — target 70-80%
- **Mutant coverage** = tested mutations / total mutations

## What Gremlins Mutates

Default operators (all enabled):

| Operator | Example |
|---|---|
| `arithmetic-base` | `+` becomes `-`, `*` becomes `/` |
| `conditionals-boundary` | `>` becomes `>=`, `<` becomes `<=` |
| `conditionals-negation` | `==` becomes `!=`, `<` becomes `>=` |
| `increment-decrement` | `++` becomes `--` |
| `invert-negatives` | `-x` becomes `x` |

Additional operators (opt-in):

```bash
gremlins unleash --invert-logical      # && <-> ||
gremlins unleash --invert-loopctrl     # break <-> continue
gremlins unleash --invert-assignments  # += <-> -=
```

## Configuration

Create `.gremlins.yaml` in the project root:

```yaml
unleash:
  diff: ""
  workers: 0                    # 0 = all CPUs
  exclude-files:
    - ".*_generated\\.go$"
    - "internal/executor/demo_data\\.go$"
  threshold:
    efficacy: 70                # exit code 10 if below
    mutant-coverage: 80         # exit code 11 if below

mutants:
  arithmetic-base:
    enabled: true
  conditionals-boundary:
    enabled: true
  conditionals-negation:
    enabled: true
  increment-decrement:
    enabled: true
  invert-negatives:
    enabled: true
  invert-logical:
    enabled: false              # enable when baseline is strong
  invert-loopctrl:
    enabled: false
  invert-assignments:
    enabled: false
```

## Workflow

### After Completing a Feature

```bash
# 1. Run tests first (must be green)
go test ./...

# 2. Mutation test the packages you changed
gremlins unleash --diff main

# 3. Filter to only survived mutants
gremlins unleash --diff main --output-statuses l

# 4. For each LIVED mutant, write a test that kills it
# 5. Re-run to confirm the mutant is now KILLED
```

### Improving Test Quality on Existing Code

```bash
# 1. Pick a package
gremlins unleash --coverpkg ./internal/config/...

# 2. Focus on LIVED mutants in business-critical code
# 3. Ignore LIVED mutants in:
#    - Logging/formatting code
#    - Error message strings
#    - Generated code
#    - Demo/fixture data
```

### In CI (GitHub Actions)

```yaml
- uses: go-gremlins/gremlins-action@v1
  with:
    version: latest
    args: --diff origin/main --threshold-efficacy 70
```

Use `--diff origin/main` in PRs for fast runs. Run full module weekly on a schedule.

## Interpreting LIVED Mutants

A LIVED mutant means: "I changed this line and no test noticed."

**Common patterns and fixes:**

1. **Boundary mutation survived** (`>` became `>=`)
   - Missing edge case test. Add a test for the exact boundary value.
   ```go
   // If maxItems > 100 survived becoming >= 100:
   // Add test: {name: "exactly 100", items: 100, wantErr: false}
   ```

2. **Arithmetic mutation survived** (`+` became `-`)
   - Test doesn't verify the computed value precisely. Add exact value assertions.

3. **Condition negation survived** (`==` became `!=`)
   - Both branches produce acceptable results. Either the test is too loose or the code has an equivalent mutant (false positive).

4. **Increment survived** (`++` became `--`)
   - Loop counter or index not tested at boundaries. Add tests for first/last element.

**Equivalent mutants (false positives):**
Some mutations produce semantically identical code. These show as LIVED but aren't real weaknesses:
- Changing `<` to `<=` when the boundary value never occurs
- Arithmetic on constants used only for cosmetic purposes
- Negating conditions that return the same value in both branches

When you encounter these, exclude the file or accept the lower efficacy score.

## What Gremlins Does NOT Catch

- Logic errors in test assertions themselves
- Missing test scenarios (it only mutates covered code)
- Concurrency bugs
- Integration issues between packages
- Incorrect error messages (doesn't mutate strings)
- Missing validation for new fields

Mutation testing complements — not replaces — code review, integration tests, and property-based testing.

## Performance Tips

- **Targeted runs**: Always use `--coverpkg` or `--diff` instead of full-module runs
- **Worker control**: `--workers 4` to limit CPU usage
- **Exclude data files**: Exclude `demo_data.go`, generated files, fixtures
- **Schedule full runs**: Run full-module mutation testing weekly, not per-commit
- **Start small**: Begin with default operators; enable extras as baseline improves
