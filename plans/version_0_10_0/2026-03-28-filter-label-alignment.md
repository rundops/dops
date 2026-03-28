# Plan: Sidebar Filter Label Alignment

**Spec:** N/A (bug fix)
**Status:** DONE

## Problem

The "Filter: " label in the sidebar is too centered/indented. Currently rendered with a 2-space indent (`"  " + filterLabel.Render("Filter: ")`), but visually it appears too far from the left edge, inconsistent with the sidebar content alignment.

## Current Code

`internal/tui/sidebar/model.go`, View() function:

```go
b.WriteString("\n")
b.WriteString("  " + filterLabel.Render("Filter: ") + filterInput.Render(m.searchQuery) + cursor)
```

## Fix

- [x] Removed leading 2-space indent — "Filter: " now starts at column 0, aligned with catalog headers
- [x] Verified against sidebar item rendering — catalog headers (`▼ default/`) start at column 0, now matches
- [x] Build passes, all sidebar tests pass
- [ ] Verify visually on WSL (deferred to manual testing)

## Files Changed

| File | Change |
|------|--------|
| `internal/tui/sidebar/model.go` | Adjust filter label indent |

## Risks

- Minimal. Single-line cosmetic change.
- Need to verify the separator line (`─`) alignment matches the new indent.
