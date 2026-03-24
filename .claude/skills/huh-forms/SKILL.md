---
name: huh-forms
description: "Custom wizard form patterns for Go. Use when creating or editing interactive forms, field-by-field parameter input, select menus, text inputs, confirmations, multi-select, or the wizard overlay. Triggers on wizard model changes, new field types, or form flow modifications."
user-invocable: false
---

# Custom Wizard Form Patterns

This project uses a **custom wizard implementation** (not Huh library) in `internal/tui/wizard/model.go`.

## Architecture

Field-by-field progression with per-field save control:

```go
type Model struct {
    params   []domain.Parameter  // ALL params to collect
    current  int                 // current field index
    values   map[string]string   // collected values
    input    textinput.Model     // for text fields
    cursor   int                 // for select/multiselect/boolean
    checked  map[int]bool        // for multiselect toggles
    phase    wizardPhase         // phaseInput or phaseSave
    changed  bool                // whether value differs from pre-fill
    resolved map[string]string   // pre-filled values from config
    prefill  map[string]bool     // which fields had saved values
    store    config.ConfigStore  // for per-field saving
}
```

## Field Types

| Type | Rendering | Input |
|---|---|---|
| `string` | `> ` text input | Type text, Enter to submit |
| `integer` | `> ` text input | Must be numeric |
| `boolean` | `[Yes] [No]` toggle | ←→ toggle, Enter confirm |
| `select` | `> option` cursor list | ↑↓ nav, Enter select |
| `multi_select` | `[x]/[ ]` checkboxes | Space toggle, ↑↓ nav, Enter confirm |
| `file_path` | `> ` text input | Type path |
| `resource_id` | `> ` text input | Type ID |
| `secret` | `> ••••••••••` or `> ****` | Enter to keep saved, type to override |

## Persistence Flow

1. Wizard shows ALL params with pre-filled saved values
2. Sensitive fields show `••••••••••  (enter to keep, type to override)`
3. Enter on pre-fill → accept, advance (no save prompt)
4. New/changed value → "Save for future runs? [Yes/No]" (default No)
5. Yes → saves to config.json at field's scope (global/catalog/runbook)
6. No → ephemeral, advances without saving

## Overlay Style

- Left accent bar: `ThickBorder()` left only, `primary` color
- Panel background: `backgroundPanel` color
- Header: green `$` + bold command text (only shows `--param` for overrides)
- Centered: `lipgloss.Place()`, width 2/3 terminal
- Context-sensitive footer hints per field type and phase

## Adding a New Field Type

1. Add `ParamXxx` constant to `internal/domain/runbook.go`
2. Add case to `fieldMode()` in wizard model
3. Add rendering in `renderCurrentField()`
4. Add key handling in `Update()` switch
5. Add schema generation in `internal/mcp/schema.go`
