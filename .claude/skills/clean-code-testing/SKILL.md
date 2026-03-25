---
name: clean-code-testing
description: Clean Code testing principles for Go. Use when writing, editing, or reviewing Go test files (*_test.go). Triggers on test function creation, table-driven test setup, test helper design, mock/stub creation, or test refactoring.
user-invocable: false
---

# Clean Code Testing for Go

Apply these principles when writing or modifying Go tests.

## Tests Derive from Specs

- Every spec acceptance criterion becomes one or more test cases
- Name tests after the behavior, not the implementation:
  ```go
  // Bad
  func TestDeployFunction(t *testing.T)

  // Good
  func TestDeploy_FailsWhenTargetUnreachable(t *testing.T)
  ```
- If you can't map a test to a spec requirement, question whether the test (or the code it tests) is needed

## FIRST Principles

### Fast

Tests must run quickly — milliseconds, not seconds. Slow tests don't get run.

- No network calls, no disk I/O in unit tests. Use interfaces and test doubles.
- No `time.Sleep`. Use channels, `sync.WaitGroup`, or fake clocks.
- If a test needs a database, it's an integration test — separate it with a build tag.

```go
// Bad — slow, flaky, depends on external service
func TestHealthCheck(t *testing.T) {
    resp, err := http.Get("https://api.example.com/health")
    // ...
}

// Good — fast, deterministic
func TestHealthCheck(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer srv.Close()
    err := CheckHealth(srv.URL)
    // ...
}
```

### Independent

Tests must not depend on each other or on execution order. No shared mutable state.

- Each test creates its own fixtures and state.
- No package-level `var` that tests mutate.
- Use `t.TempDir()` for filesystem tests — it cleans up automatically.
- Use `t.Parallel()` when possible to prove independence.

```go
// Bad — test B depends on test A's side effect
var globalStore = NewStore()

func TestStore_Add(t *testing.T) {
    globalStore.Add("key", "value") // mutates shared state
}

func TestStore_Get(t *testing.T) {
    v := globalStore.Get("key") // depends on TestStore_Add running first
}

// Good — each test is self-contained
func TestStore_Add(t *testing.T) {
    store := NewStore()
    store.Add("key", "value")
    if v := store.Get("key"); v != "value" {
        t.Errorf("Get(key) = %q, want value", v)
    }
}
```

### Repeatable

Same result every time, in any environment, in any order. No flaky tests.

- Control all sources of non-determinism: time, randomness, concurrency, environment.
- Inject `time.Now` as a dependency when time matters.
- Use deterministic seeds for random values, or inject a fake random source.
- Never depend on map iteration order.

```go
// Bad — depends on wall clock
func TestExpiry(t *testing.T) {
    token := NewToken()
    time.Sleep(2 * time.Second)
    if !token.IsExpired() { t.Error("should be expired") }
}

// Good — inject clock
func TestExpiry(t *testing.T) {
    now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
    token := NewToken(WithClock(func() time.Time { return now }))
    now = now.Add(2 * time.Hour)
    if !token.IsExpired() { t.Error("should be expired") }
}
```

### Self-Validating

Tests produce pass or fail. No manual inspection of logs or output.

- Every test must have assertions. A test with no assertions always passes — useless.
- Use `t.Errorf` with context: include input, got, and want.
- Don't `fmt.Println` results for a human to read — assert them.

```go
// Bad — requires human to read output
func TestFormat(t *testing.T) {
    result := Format(input)
    fmt.Println(result) // "looks right" is not a test
}

// Good — machine-verifiable
func TestFormat(t *testing.T) {
    result := Format(input)
    if result != expected {
        t.Errorf("Format(%q) = %q, want %q", input, result, expected)
    }
}
```

### Timely

Write tests alongside the code, not weeks later.

- Tests written after the fact rationalize the implementation instead of verifying behavior.
- If writing the test is hard, the design likely needs improvement (testability = good design).
- Use query methods and interfaces to make code testable without exposing internals.
- Don't access unexported struct fields in tests — add minimal public query methods instead.

```go
// Bad — test reaches into internals
func TestModel_Update(t *testing.T) {
    m := NewModel()
    m, _ = m.Update(someMsg)
    if m.cursor != 3 { // couples test to field name
        t.Error("wrong cursor")
    }
}

// Good — test uses public API
func TestModel_Update(t *testing.T) {
    m := NewModel()
    m, _ = m.Update(someMsg)
    if m.Cursor() != 3 { // survives field renames
        t.Error("wrong cursor")
    }
}
```

## Table-Driven Tests

- Use table-driven tests for related scenarios — this is idiomatic Go:
  ```go
  func TestValidateTarget(t *testing.T) {
      tests := []struct {
          name    string
          target  string
          wantErr bool
      }{
          {name: "valid hostname", target: "prod-01.example.com", wantErr: false},
          {name: "empty target", target: "", wantErr: true},
          {name: "invalid chars", target: "prod 01!", wantErr: true},
      }

      for _, tt := range tests {
          t.Run(tt.name, func(t *testing.T) {
              err := ValidateTarget(tt.target)
              if (err != nil) != tt.wantErr {
                  t.Errorf("ValidateTarget(%q) error = %v, wantErr %v", tt.target, err, tt.wantErr)
              }
          })
      }
  }
  ```
- Each table entry should have a descriptive `name` field
- Keep the test logic in the loop minimal — complexity belongs in the table entries

## One Concept per Test

- A test function verifies one behavioral concept
- Multiple assertions are fine if they all verify the same concept
- If a test name has "and" in it, consider splitting it

## Test Helpers

- Use `t.Helper()` in helper functions so failures report the caller's line number
- Put shared test utilities in a `testutil` package or `_test.go` files
- Test helpers should not use `t.Fatal` unless the failure makes continuing meaningless

## Test Doubles (Go Style)

- Prefer interfaces + simple stub implementations over mocking frameworks
- Fakes are better than mocks for complex behavior:
  ```go
  type fakePusher struct {
      pushed []string
      err    error
  }

  func (f *fakePusher) Push(target string) error {
      f.pushed = append(f.pushed, target)
      return f.err
  }
  ```
- Only mock at boundaries (external services, filesystem, clock) — not internal types
- If you need to mock something deep inside, the design needs refactoring (DIP violation)

## Test Readability

- Tests are documentation — a new developer should understand the feature by reading the test
- Follow Arrange-Act-Assert (Given-When-Then):
  ```go
  func TestDeploy_SucceedsWithValidTarget(t *testing.T) {
      // Arrange
      pusher := &fakePusher{}
      target := "prod-01"

      // Act
      err := Deploy(pusher, target)

      // Assert
      if err != nil {
          t.Fatalf("unexpected error: %v", err)
      }
      if len(pusher.pushed) != 1 || pusher.pushed[0] != target {
          t.Errorf("expected push to %q, got %v", target, pusher.pushed)
      }
  }
  ```
- Don't DRY tests at the cost of readability — some duplication in tests is fine
- Use `testdata/` directory for fixture files

## What NOT to Do

- Do not test private functions directly — test through the public API
- Do not write tests that test the Go standard library or third-party libraries
- Do not use `init()` in test files
- Do not skip flaky tests with `t.Skip` — fix the flakiness
- Do not use `time.Sleep` in tests — use channels, waitgroups, or fake clocks
