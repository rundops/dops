# Plan: MCP Create-Runbook Prompt Scaffolding

**Spec:** v0.11.0 Feature 3
**Effort:** 2 hours
**Files:** `internal/mcp/prompts.go`, `internal/mcp/mcp_extra_test.go`

## Problem

The `create-runbook` MCP prompt templates contain `# TODO: Add parameter
variables` and `# TODO: Implement` placeholders. Templates should generate
actual parameter variable extraction from the runbook's parameter list.

## TDD Steps

### Red: Write failing tests

**File:** `internal/mcp/mcp_extra_test.go`

1. Test `TestPromptScaffold_BashRequiredParams` — create-runbook prompt
   with parameters that have `required: true`. Verify the bash template
   generates `VAR="${VAR:?var is required}"` lines.

2. Test `TestPromptScaffold_BashOptionalWithDefault` — parameter with
   `required: false` and `default: "us-east-1"`. Verify template
   generates `VAR="${VAR:-us-east-1}"`.

3. Test `TestPromptScaffold_BashSecretParam` — parameter with
   `secret: true`. Verify template includes a comment noting the value
   is masked.

4. Test `TestPromptScaffold_PowerShellParams` — same as bash tests but
   verify PowerShell equivalents (`$env:VAR`, `throw`).

5. Test `TestPromptScaffold_NoParams` — runbook with zero parameters.
   Verify template has no empty variable section (clean output).

6. Test `TestPromptScaffold_MixedParams` — mix of required, optional,
   secret. Verify all three types render correctly in one template.

### Green: Implement

**File:** `internal/mcp/prompts.go`

Extract a function:
```go
func generateParamVars(params []domain.Parameter, shell string) string
```

- `shell == "bash"`:
  - Required: `PARAM_NAME="${PARAM_NAME:?param_name is required}"`
  - Optional with default: `PARAM_NAME="${PARAM_NAME:-defaultValue}"`
  - Optional no default: `PARAM_NAME="${PARAM_NAME:-}"`
  - Secret: append `# (secret — value is masked in UI)`
- `shell == "powershell"`:
  - Required: `$ParamName = if ($env:PARAM_NAME) { $env:PARAM_NAME } else { throw 'param_name is required' }`
  - Optional: `$ParamName = if ($env:PARAM_NAME) { $env:PARAM_NAME } else { "defaultValue" }`

Replace the TODO blocks in `registerPrompts()` with calls to
`generateParamVars()`. Replace `# TODO: Implement` with a body comment
referencing the declared variables.

### Refactor

- Remove all TODO comments from template strings
- Keep `generateParamVars` as a pure function (no side effects, easy to test)

## Verification

```
go test ./internal/mcp/... -v -run TestPromptScaffold
go build ./...
go vet ./...
```
