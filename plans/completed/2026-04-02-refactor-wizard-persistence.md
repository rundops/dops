# Refactor: Wizard Persistence via Messages

**Status:** Complete
**Effort:** Half day
**Risk:** Medium — changes save-on-advance flow
**Branch:** `refactor/wizard-persistence-v2`

## Problem

The wizard (`internal/tui/wizard/model.go`) directly imports `config` and `vault` packages to persist parameter values in `saveCurrentField()`. This violates BubbleTea's architecture where child components should emit messages and let the parent handle side effects.

**Current flow:**
```
User confirms save → wizard.saveCurrentField()
  → config.Set(m.cfg, keyPath, val)   // direct mutation
  → m.vault.Save(&m.cfg.Vars)         // direct I/O
```

**Problems:**
1. Wizard has a direct dependency on `config.Set()` and vault I/O
2. Wizard holds mutable pointer to `*domain.Config` — shared state mutation
3. Save errors are silently swallowed (stored in `m.err`, wizard continues)
4. No tests for save behavior — the coupling makes it hard to test
5. `SetStore(cfg, vault)` is an awkward post-construction injection

## Proposed Flow

```
User confirms save → wizard emits SaveFieldMsg
  → App.handleAppMessage receives SaveFieldMsg
  → App calls config.Set + vault.Save
  → App sends SaveFieldResultMsg back to wizard
  → Wizard advances (or shows error)
```

## Implementation

### Step 1: Define New Messages

**File:** `internal/tui/wizard/messages.go`

```go
// SaveFieldMsg is emitted when the user confirms saving a parameter value.
type SaveFieldMsg struct {
    Scope       string
    ParamName   string
    CatalogName string
    RunbookName string
    Value       string
}

// SaveFieldResultMsg is sent back to the wizard after a save attempt.
type SaveFieldResultMsg struct {
    Err error
}
```

### Step 2: Replace saveCurrentField with Message Emission

**File:** `internal/tui/wizard/model.go`

In `updateSaveConfirm`, where `m.saveCurrentField()` is called today (lines 491, 497):

```go
// Before:
m.saveCurrentField()
return m.advance()

// After:
p := m.params[m.current]
return m, func() tea.Msg {
    return SaveFieldMsg{
        Scope:       p.Scope,
        ParamName:   p.Name,
        CatalogName: m.catalog.Name,
        RunbookName: m.runbook.Name,
        Value:       m.values[p.Name],
    }
}
```

Add a new `waitingForSave` phase so the wizard pauses until the result comes back:
```go
const (
    phaseInput wizardPhase = iota
    phaseSave
    phaseWaitingSave  // new: waiting for App to confirm save
)
```

Handle the result:
```go
case SaveFieldResultMsg:
    if msg.Err != nil {
        m.err = fmt.Sprintf("save failed: %v", msg.Err)
    }
    m.phase = phaseInput
    return m.advance()
```

### Step 3: Handle SaveFieldMsg in App

**File:** `internal/tui/app.go` in `handleAppMessage`:

```go
case wizard.SaveFieldMsg:
    keyPath := vars.VarKeyPath(msg.Scope, msg.ParamName, msg.CatalogName, msg.RunbookName)
    var saveErr error
    if err := config.Set(m.deps.Config, keyPath, msg.Value); err != nil {
        saveErr = err
    } else if err := m.deps.Vault.Save(&m.deps.Config.Vars); err != nil {
        saveErr = err
    }
    return m, func() tea.Msg {
        return wizard.SaveFieldResultMsg{Err: saveErr}
    }, true
```

### Step 4: Remove Config/Vault Dependencies from Wizard

- Delete `saveCurrentField()` method
- Remove `cfg *domain.Config` and `vault domain.VaultStore` fields from Model
- Remove `SetStore()` method
- Remove `config` and `vault` imports from wizard package
- Update `openWizard()` in app.go to stop calling `SetStore()`

### Step 5: Add Tests

**File:** `internal/tui/wizard/model_test.go`

Test the new flow:
- Confirm save prompt emits `SaveFieldMsg` with correct scope/name/value
- `SaveFieldResultMsg{Err: nil}` causes advance to next field
- `SaveFieldResultMsg{Err: errors.New("fail")}` sets `m.err` and still advances
- Local-scoped params skip save prompt entirely (no message emitted)

**File:** `internal/tui/app_test.go`

Test the handler:
- `SaveFieldMsg` triggers `config.Set` + `vault.Save`
- Save failure returns `SaveFieldResultMsg` with error

## Constraints

- [ ] No behavior change from user perspective — same prompts, same save timing
- [ ] Wizard package no longer imports `config` or `vault`
- [ ] Wizard no longer holds mutable `*domain.Config` pointer
- [ ] All existing wizard tests pass
- [ ] New tests cover save flow
- [ ] `go build ./...` clean
- [ ] Manual test: run wizard, save a param, verify it persists on next run

## Migration Notes

The `WizardConfig` struct loses no fields — `SetStore` was a post-construction call, not part of `WizardConfig`. The `openWizard()` method in app.go simplifies by removing the `SetStore` call.
