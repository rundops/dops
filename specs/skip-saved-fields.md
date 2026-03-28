# Skip Saved Fields in Wizard

## Overview

When a wizard field has a saved value from a previous run (stored in the vault), automatically apply that value and skip displaying the field in the wizard form. This reduces repetitive form-filling for frequently-used runbooks. Users can force all fields to show via a flag or key binding.

## Requirements

- Fields with saved (prefilled) values must be auto-applied without showing in the wizard
- Secret fields with saved values must also be skipped (they already show dot placeholders)
- If ALL fields have saved values, the wizard must be skipped entirely (existing `ShouldSkip` behavior, unchanged)
- A `--prompt-all` flag on `dops run` must force all fields to show (overrides skip behavior)
- In the TUI, pressing `e` (edit) on a runbook with all saved fields must open the wizard with all fields visible
- The wizard header must show a summary of auto-applied values: "Applied N saved values. Press Ctrl+E to edit all."
- If a user Ctrl+E during wizard, remaining skipped fields must be inserted into the form
- Auto-applied values must still be passed to the script as environment variables

## Acceptance Criteria

- [ ] Field with saved value is skipped in wizard (not rendered)
- [ ] Auto-applied values passed correctly to script execution
- [ ] `dops run <id> --prompt-all` shows all fields including saved ones
- [ ] Wizard header shows count of auto-applied fields
- [ ] Ctrl+E during wizard reveals remaining skipped fields
- [ ] If only some fields have saved values, unsaved fields still show normally
- [ ] Secret fields with saved values are skipped (not shown with dots)
- [ ] Existing `ShouldSkip()` behavior preserved (all fields saved = no wizard)

## Error Cases

- Saved value no longer valid (e.g., select option removed from runbook.yaml): field shows normally with warning
- Vault decryption failure: all fields show normally (graceful degradation)

## Out of Scope

- Editing previously saved values from a summary screen (separate UX)
- Batch clearing saved values per runbook
- Visual indicator on sidebar for runbooks with saved values
