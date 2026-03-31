---
name: tdd-workflow
description: Test Driven Development workflow for Go. Use when implementing new features, adding new functions, or when the user asks for TDD. Triggers on "write tests first", "TDD", "test driven", new feature implementation where tests should drive the design.
user-invocable: true
---

# TDD Workflow for Go

When implementing features test-first, follow this exact cycle. Do not skip steps.

## The Cycle: Red-Green-Refactor

### 1. RED — Write a Failing Test

Write the test **before** the implementation exists. The test defines the desired behavior.

```go
func TestVarKeyPath_GlobalScope(t *testing.T) {
    got := VarKeyPath("global", "region", "default", "hello-world")
    want := "vars.global.region"
    if got != want {
        t.Errorf("VarKeyPath() = %q, want %q", got, want)
    }
}
```

At this point, the code **must not compile** or the test **must fail**. If it passes, either the test is wrong or the feature already exists.

Run: `go test ./internal/vars/... 2>&1` — expect a compilation error or test failure.

**Rules for the Red step:**
- Test one behavior per test function
- Name the test after the behavior: `TestDeploy_FailsWhenTargetUnreachable`
- Use table-driven tests when multiple inputs test the same behavior
- The compiler is part of Red — a type error IS a failing test
- Define the function signature in the test first, then make the compiler happy

### 2. GREEN — Write Minimum Code to Pass

Write the **smallest possible implementation** that makes the test pass. No more.

```go
func VarKeyPath(scope, paramName, catalogName, runbookName string) string {
    return fmt.Sprintf("vars.global.%s", paramName)
}
```

This is intentionally incomplete — it only handles the global case. That's correct. The next Red step will force the catalog case.

Run: `go test ./internal/vars/...` — expect green.

**Rules for the Green step:**
- Do not generalize. Do not add code "because we'll need it later."
- Hardcode if that's the fastest path to green. The next test will force real logic.
- Do not refactor yet — that's the next step.
- If green requires touching more than one file, the scope may be too large. Break into smaller tests.

### 3. REFACTOR — Clean Up Under Green Tests

With passing tests as a safety net, improve the code:
- Extract helpers, rename variables, simplify logic
- Remove duplication between production code and between tests
- Ensure the implementation follows codebase conventions

Run: `go test ./internal/vars/...` — must stay green after every change.

**Rules for the Refactor step:**
- Never add new behavior during refactor — only restructure
- Run tests after every change, not just at the end
- Refactor test code too — extract test helpers, improve names

### 4. Repeat

Go back to Red. Add the next test case. Drive the next behavior.

## Practical TDD in This Codebase

### When Writing a New Function

```
1. Create or open the _test.go file
2. Write the first test case (simplest happy path)
3. Run tests — RED (won't compile)
4. Write the function signature + minimal body — GREEN
5. Write the next test case (next behavior or edge case) — RED
6. Extend the implementation — GREEN
7. Refactor if needed — still GREEN
8. Repeat until all behaviors are covered
```

### When Writing Table-Driven Tests

Start with one row, get it green, then add rows:

```go
func TestParseRiskLevel(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    RiskLevel
        wantErr bool
    }{
        // Start here — one case
        {name: "low", input: "low", want: RiskLow},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseRiskLevel(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ParseRiskLevel(%q) = %v, want %v", tt.input, got, tt.want)
            }
        })
    }
}
```

Get it green, then add cases one at a time:
```go
        {name: "high", input: "high", want: RiskHigh},
        {name: "invalid", input: "extreme", wantErr: true},
        {name: "empty", input: "", wantErr: true},
```

Each new row may require extending the implementation.

### When Adding to BubbleTea Models

TDD a model's Update behavior by sending messages and asserting state:

```go
// RED: write test for new behavior
func TestSidebar_SearchFilters(t *testing.T) {
    m := New(testCatalogs(), 20, testutil.TestStyles())
    m.Init()

    // Type "/" to enter search, then "drain"
    m, _ = pressKey(m, "/")
    m, _ = pressKey(m, "d")
    m, _ = pressKey(m, "r")

    // Only drain-node should be visible
    if count := m.VisibleCount(); count != 1 {
        t.Errorf("visible = %d, want 1", count)
    }
}

// GREEN: implement VisibleCount() and search filtering
// REFACTOR: clean up
```

### When Fixing a Bug

Always write a test that reproduces the bug **before** fixing it:

```
1. Write a test that fails the same way the bug manifests — RED
2. Fix the bug — GREEN
3. The test now serves as a regression guard
```

This ensures the bug is actually fixed (not just masked) and prevents regressions.

## What NOT to Do in TDD

- **Don't write all tests first then implement** — that's not TDD, that's waterfall testing. The cycle is one test at a time.
- **Don't skip Red** — if the test passes immediately, it's not testing new behavior.
- **Don't write production code without a failing test** — every line of production code should be demanded by a test.
- **Don't test implementation details** — test behavior through the public API. If you need to reach into private fields, add a query method.
- **Don't refactor during Red or Green** — keep the steps separate.
- **Don't chase 100% coverage** — TDD naturally produces high coverage, but coverage is a side effect, not the goal.

## When TDD Is Not Worth It

- Exploratory prototyping (spike and throw away)
- One-off scripts
- Pure UI layout code (verify visually with VHS/Freeze instead)
- Generated code

For everything else — business logic, validation, state machines, parsers, CLI commands — TDD produces better design and fewer bugs.
