# Plan: Web UI Execution View — Remove Duplicate Status, Modernize Header

**Spec:** v0.11.0 Fix 1
**Effort:** 1-2 hours
**Files:** `web/src/views/ExecutionView.vue`

## Problem

`ExecutionView.vue` shows the status pill ("Completed" / "Failed") in both
the header bar and a footer bar. Redundant information. Header also needs
a modern refresh.

## Current Layout

```
Header:  [Execution] [runbook-name] [status pill] ... [Cancel|Done|Failed]
Output:  (scrolling log lines)
Footer:  [status pill] [duration] ... [← Back to runbook]
```

Status communicated 3 times: header pill, header right-side label, footer pill.

## Target Layout

```
Header:  [runbook-name] [status pill · duration] ... [Cancel] (running)
         [runbook-name] [Completed · 3.2s]             (success)
         [runbook-name] [Failed · 1.4s]                (error)
Output:  (scrolling log lines)
Footer:  [3.2s] ......................... [← Back to runbook]
```

## Steps

### 1. Header — merge status + duration

- Remove "Execution" label (runbook name is sufficient context)
- Keep status pill with animated dot for running state
- On completion: append ` · {duration}` inside the pill text
- Remove the separate right-side Done/Failed labels
- Keep Cancel button (running state only)

### 2. Footer — remove status pill

- Replace status pill with just the duration as plain text
- Keep back button
- Footer only visible when complete (existing `v-if="isComplete"`)

### 3. Header polish

- Use `font-mono` for runbook name (consistent with sidebar)
- Tighten padding: `px-5 py-3` instead of `px-6 py-4`
- Add subtle left border accent using primary color on the status pill

### 4. TypeScript check

```
cd web && npx vue-tsc --noEmit
```

## Verification

```
cd web && npx vue-tsc --noEmit
make web  (if available — builds Vue SPA)
```

Manual: open web UI, execute a runbook, verify:
- Status appears exactly once (header pill only)
- Duration shows in pill after completion
- Footer has duration + back button, no pill
- Running state shows pulse dot + Cancel
- Error state shows Failed pill
