# dops-cli — Remaining Feature TODO

All parity features and text selection have been implemented.

## Completed

### 2026-03-23 — Output Pane Refinement
- [x] ANSI handling with charmbracelet/x/ansi
- [x] Scrollbar with proportional thumb (primary color)
- [x] Smart auto-scroll (atBottom flag)
- [x] Horizontal scrolling (h/l keys)
- [x] Live streaming via tea.Program.Send()
- [x] Buffered log file persistence
- [x] Three-section layout (header/log/footer) inside persistent outer border
- [x] Focus management (Tab, hover-to-focus, click-to-focus)
- [x] Terminal background from theme
- [x] Layout spacing and padding

### 2026-03-24 — Feature Parity & Selection
- [x] Risk confirmation gates (High=y/N, Critical=type ID)
- [x] Process management (ctrl+x stop with SIGKILL)
- [x] Help overlay (? key, context-aware)
- [x] Catalog management CLI (list/add/remove/install/update with --ref)
- [x] Dry-run mode
- [x] Text selection with highlight (terminal-absolute coordinates, view-relative rendering)
- [x] Clipboard copy (auto on release, y key, OSC 52 fallback)
- [x] "Copied to Clipboard!" floating border badge
- [x] Click-to-copy green flash on header/footer
- [x] Search UI ("Search:" prompt, navigation hints)

## Future Enhancements

- [ ] Selection highlighting during drag without terminal native overlay (requires terminal-level support)
- [ ] MCP server integration
- [ ] Additional input types (multi_select, file_path, resource_id) in wizard
- [ ] Sidebar folder compaction (single-child chains → "parent / child")
- [ ] Spinner during execution
- [ ] Update check banner
