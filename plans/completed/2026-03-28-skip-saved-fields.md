# Plan: Skip Saved Fields in Wizard

**Spec:** [specs/skip-saved-fields.md](../../specs/skip-saved-fields.md)
**Status:** DONE

## Summary

Auto-apply saved field values and skip them in the wizard form, reducing repetitive input for frequently-used runbooks.

## Implementation Steps

### Step 1: Wizard — Skip Logic
- [x] Added `showAll bool` and `skipped map[int]bool` to Model
- [x] `NewWithOptions(rb, cat, resolved, promptAll)` constructor with skip control
- [x] `shouldSkipField(idx)` returns true when field is prefilled and `showAll` is false
- [x] `nextUnskipped(start)` finds next non-skippable field
- [x] `advance()` skips prefilled fields, auto-applying their values
- [x] `goBack()` skips backward over auto-applied fields
- [x] When all fields are skippable, `Init()` returns `WizardSubmitMsg` immediately
- [x] Tests: partial prefill, all prefilled, promptAll mode

### Step 2: Wizard — Summary Header
- [x] View shows "Applied N saved values. Ctrl+E to edit all." when fields are skipped
- [x] Styled with `TextMuted`
- [x] Completed fields list excludes skipped fields
- [x] Test: header rendering with skipped summary

### Step 3: Wizard — Ctrl+E Reveal
- [x] `ctrl+e` key handler sets `showAll = true` and clears `skipped` map
- [x] Cursor stays on current field
- [x] Footer hints include "ctrl+e edit all" when fields are skipped

### Step 4: CLI — `--prompt-all` Flag (deferred)
- `dops run` uses `--param` flags directly (no wizard). Flag not applicable.
- `NewWithOptions` exists for any future caller that needs `promptAll`.

### Step 5: TUI — Edit Key Binding (deferred)
- Existing wizard flow always shows for runbooks with parameters
- When all fields are saved, wizard auto-submits via `Init()`
- `e` key to force wizard open can be added later if needed

### Step 6: Validation — Stale Saved Values
- Handled implicitly: if a select value is no longer in options, the field shows normally
  (prefill sets cursor but doesn't skip — value won't match any option on re-display)

## Files Changed

| File | Change |
|------|--------|
| `internal/tui/wizard/model.go` | `showAll`, `skipped`, `NewWithOptions`, skip logic, Ctrl+E, header |
| `internal/tui/wizard/model_test.go` | Tests for skip, all-prefilled, promptAll, Ctrl+E hint |
