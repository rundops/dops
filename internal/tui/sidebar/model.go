package sidebar

import (
	"strings"
	"time"

	"dops/internal/catalog"
	"dops/internal/domain"
	"dops/internal/theme"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

type entry struct {
	isHeader bool
	catalog  domain.Catalog
	runbook  domain.Runbook
}

type Model struct {
	entries       []entry
	catalogs      []catalog.CatalogWithRunbooks
	activeCatalog int // 0 = All, 1..N = specific catalog index+1
	collapsed     map[string]bool               // catalog name → collapsed
	cursor        int                           // index into visible()
	hoverIdx      int                           // index into visible() for mouse hover, -1 = none
	lastClickY    int                           // Y of last click (for double-click detection)
	lastClickAt   time.Time                     // time of last click
	height        int
	offset        int
	searching     bool
	searchQuery   string
	styles        *theme.Styles
}

func New(catalogs []catalog.CatalogWithRunbooks, height int, styles *theme.Styles) Model {
	var entries []entry

	for _, cwr := range catalogs {
		entries = append(entries, entry{
			isHeader: true,
			catalog:  cwr.Catalog,
		})
		for _, rb := range cwr.Runbooks {
			entries = append(entries, entry{
				runbook: rb,
				catalog: cwr.Catalog,
			})
		}
	}

	// Start cursor on first runbook (skip first header)
	firstRB := 0
	for i, e := range entries {
		if !e.isHeader {
			firstRB = i
			break
		}
	}

	return Model{
		entries:   entries,
		catalogs:  catalogs,
		collapsed: make(map[string]bool),
		cursor:    firstRB,
		hoverIdx:  -1,
		height:    height,
		styles:    styles,
	}
}

func (m *Model) SetHeight(h int) {
	m.height = h
}

func (m Model) Init() tea.Cmd {
	vis := m.visible()
	if m.cursor < 0 || m.cursor >= len(vis) {
		return nil
	}
	e := m.entries[vis[m.cursor]]
	if e.isHeader {
		return nil
	}
	return func() tea.Msg {
		return RunbookSelectedMsg{Runbook: e.runbook, Catalog: e.catalog}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		m.hoverIdx = -1 // clear hover on keyboard input
		if m.searching {
			return m.updateSearch(msg)
		}
		return m.updateNormal(msg)
	case tea.MouseClickMsg:
		return m.handleClick(msg)
	case tea.MouseMotionMsg:
		return m.handleMotion(msg), nil
	}
	return m, nil
}

func (m Model) tabBarHeight() int {
	if m.tabLabels() == nil {
		return 0
	}
	return 1
}

func (m Model) mouseToIdx(y int) int {
	// Y is content-relative (0 = first row of sidebar), translated by the app.
	// Subtract tab bar height since those rows aren't in the entry list.
	return y - m.tabBarHeight() + m.offset
}

func (m Model) handleClick(msg tea.MouseClickMsg) (Model, tea.Cmd) {
	vis := m.visible()
	if len(vis) == 0 {
		return m, nil
	}

	idx := m.mouseToIdx(msg.Y)
	if idx < 0 || idx >= len(vis) {
		return m, nil
	}

	e := m.entries[vis[idx]]
	now := time.Now()

	// Double-click detection: same Y, within 400ms
	isDoubleClick := msg.Y == m.lastClickY && now.Sub(m.lastClickAt) < 400*time.Millisecond
	m.lastClickY = msg.Y
	m.lastClickAt = now
	m.cursor = idx

	if e.isHeader {
		m.collapsed[e.catalog.Name] = !m.collapsed[e.catalog.Name]
		return m, nil
	}

	if isDoubleClick {
		// Double-click on runbook — execute
		return m, func() tea.Msg {
			return RunbookExecuteMsg{Runbook: e.runbook, Catalog: e.catalog}
		}
	}

	// Single click — select
	return m, m.selectionCmd()
}

func (m Model) handleMotion(msg tea.MouseMotionMsg) Model {
	vis := m.visible()
	idx := m.mouseToIdx(msg.Y)

	if idx < 0 || idx >= len(vis) {
		m.hoverIdx = -1
	} else {
		m.hoverIdx = idx
	}
	return m
}

func (m Model) updateNormal(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	vis := m.visible()
	if len(vis) == 0 {
		return m, nil
	}

	switch {
	case msg.Code == tea.KeyDown:
		if m.cursor < len(vis)-1 {
			m.cursor++
			m.ensureVisible()
			return m, m.selectionCmd()
		}
	case msg.Code == tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
			m.ensureVisible()
			return m, m.selectionCmd()
		}
	case msg.Code == tea.KeyEnter || msg.Text == " " || msg.String() == "space":
		return m.toggleOrSelect()
	case msg.Code == tea.KeyLeft:
		if m.activeCatalog > 0 {
			return m.switchCatalog(-1)
		}
		return m.collapseOrJumpToParent()
	case msg.Code == tea.KeyRight:
		if m.activeCatalog > 0 {
			return m.switchCatalog(1)
		}
		return m.expandHeader()
	case msg.String() == "ctrl+h":
		return m.switchCatalog(-1)
	case msg.String() == "ctrl+l":
		return m.switchCatalog(1)
	case msg.Code == tea.KeyEscape:
		if m.searchQuery != "" {
			m.searchQuery = ""
			m.cursor = 0
			return m, m.selectionCmd()
		}
	case msg.Text == "/" || msg.String() == "/":
		m.searching = true
		m.searchQuery = ""
		return m, nil
	}
	return m, nil
}

func (m Model) collapseOrJumpToParent() (Model, tea.Cmd) {
	vis := m.visible()
	if m.cursor < 0 || m.cursor >= len(vis) {
		return m, nil
	}

	e := m.entries[vis[m.cursor]]
	if e.isHeader {
		// On a header — collapse it
		if !m.collapsed[e.catalog.Name] {
			m.collapsed[e.catalog.Name] = true
		}
		return m, nil
	}

	// On a runbook — jump to its parent catalog header
	for i := m.cursor - 1; i >= 0; i-- {
		if m.entries[vis[i]].isHeader {
			m.cursor = i
			m.ensureVisible()
			return m, nil
		}
	}
	return m, nil
}

func (m Model) expandHeader() (Model, tea.Cmd) {
	vis := m.visible()
	if m.cursor < 0 || m.cursor >= len(vis) {
		return m, nil
	}

	e := m.entries[vis[m.cursor]]
	if e.isHeader && m.collapsed[e.catalog.Name] {
		m.collapsed[e.catalog.Name] = false
	}
	return m, nil
}

func (m Model) toggleOrSelect() (Model, tea.Cmd) {
	vis := m.visible()
	if m.cursor < 0 || m.cursor >= len(vis) {
		return m, nil
	}

	e := m.entries[vis[m.cursor]]
	if e.isHeader {
		catName := e.catalog.Name
		m.collapsed[catName] = !m.collapsed[catName]
		return m, nil
	}

	// Runbook selected — emit for app to handle (open wizard)
	return m, func() tea.Msg {
		return RunbookExecuteMsg{Runbook: e.runbook, Catalog: e.catalog}
	}
}

func (m Model) updateSearch(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch {
	case msg.String() == "ctrl+h":
		return m.switchCatalog(-1)
	case msg.String() == "ctrl+l":
		return m.switchCatalog(1)
	case msg.Code == tea.KeyEnter:
		// Stop typing but keep filtered results visible.
		m.searching = false
		return m, m.selectionCmd()

	case msg.Code == tea.KeyEscape:
		m.searching = false
		m.searchQuery = ""
		m.cursor = 0
		return m, m.selectionCmd()

	case msg.Code == tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.cursor = 0
		}
		return m, m.selectionCmd()

	case msg.Code == tea.KeyDown:
		vis := m.visible()
		if len(vis) > 0 && m.cursor < len(vis)-1 {
			m.cursor++
			return m, m.selectionCmd()
		}
		return m, nil

	case msg.Code == tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
			return m, m.selectionCmd()
		}
		return m, nil

	default:
		if msg.Text != "" {
			m.searchQuery += msg.Text
			m.cursor = 0
			return m, m.selectionCmd()
		}
	}
	return m, nil
}

// visible returns indices into m.entries for items currently visible.
func (m Model) visible() []int {
	if m.searchQuery != "" {
		return m.filteredVisible()
	}

	activeName := m.activeCatalogName()

	// All tab: show collapsible tree (original behavior)
	if activeName == "" {
		var vis []int
		for i, e := range m.entries {
			if e.isHeader {
				vis = append(vis, i)
				continue
			}
			if !m.collapsed[e.catalog.Name] {
				vis = append(vis, i)
			}
		}
		return vis
	}

	// Single catalog tab: show only that catalog's runbooks (no headers)
	var vis []int
	for i, e := range m.entries {
		if e.catalog.Name == activeName && !e.isHeader {
			vis = append(vis, i)
		}
	}
	return vis
}

// filteredVisible returns visible items matching the search query.
func (m Model) filteredVisible() []int {
	q := strings.ToLower(m.searchQuery)
	activeName := m.activeCatalogName()
	matchedCatalogs := make(map[string]bool)

	// Find matching runbooks (scoped to active catalog if set)
	var vis []int
	for i, e := range m.entries {
		if e.isHeader {
			continue
		}
		if activeName != "" && e.catalog.Name != activeName {
			continue
		}
		if strings.Contains(strings.ToLower(e.runbook.Name), q) {
			matchedCatalogs[e.catalog.Name] = true
			vis = append(vis, i)
		}
	}

	// Single catalog tab: return flat list (no headers)
	if activeName != "" {
		return vis
	}

	// All tab: prepend catalog headers for matched catalogs
	var result []int
	for i, e := range m.entries {
		if e.isHeader && matchedCatalogs[e.catalog.Name] {
			result = append(result, i)
			for _, ri := range vis {
				if m.entries[ri].catalog.Name == e.catalog.Name {
					result = append(result, ri)
				}
			}
		}
	}
	return result
}

func (m Model) renderTabBar(width int) string {
	labels := m.tabLabels()
	if labels == nil {
		return ""
	}

	activeStyle := lipgloss.NewStyle().Bold(true)
	inactiveStyle := lipgloss.NewStyle()
	if m.styles != nil {
		activeStyle = m.styles.Primary.Bold(true)
		inactiveStyle = m.styles.TextMuted
	}

	var tabs []string
	for i, label := range labels {
		if i == m.activeCatalog {
			tabs = append(tabs, activeStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveStyle.Render(label))
		}
	}

	return " " + strings.Join(tabs, " │ ") + "\n"
}

func (m Model) View() string {
	if len(m.entries) == 0 {
		return "  No runbooks loaded"
	}

	vis := m.visible()
	lines := m.buildLines(vis)

	// Reserve rows for tab bar and filter bar
	filterHeight := 0
	if m.searching || m.searchQuery != "" {
		filterHeight = 2 // blank separator + filter line
	}

	// Scrolling
	visibleLines := m.height - filterHeight - m.tabBarHeight()
	if visibleLines <= 0 {
		visibleLines = 1
	}

	start := m.offset
	if start > len(lines) {
		start = len(lines)
	}
	end := start + visibleLines
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[start:end]

	var sb strings.Builder

	// Tab bar (when multiple catalogs)
	tabBar := m.renderTabBar(0)
	if tabBar != "" {
		sb.WriteString(tabBar)
	}

	for _, line := range visible {
		sb.WriteString(line + "\n")
	}

	// Pad remaining lines to push filter bar to bottom (only when filter is active)
	if m.searching || m.searchQuery != "" {
		renderedLines := len(visible)
		for renderedLines < visibleLines {
			sb.WriteString("\n")
			renderedLines++
		}
	}

	if m.searching || m.searchQuery != "" {
		// Filter bar pinned at bottom with separator
		filterLabel := lipgloss.NewStyle()
		filterInput := lipgloss.NewStyle()
		if m.styles != nil {
			filterLabel = m.styles.TextMuted
			filterInput = m.styles.Text
		}
		cursor := ""
		if m.searching {
			cursor = "█"
		}
		sb.WriteString("\n")
		sb.WriteString(filterLabel.Render("Filter: ") + filterInput.Render(m.searchQuery) + cursor)
	}

	return sb.String()
}

type entryStyles struct {
	header         lipgloss.Style
	headerSelected lipgloss.Style
	hover          lipgloss.Style
	cursor         lipgloss.Style
	selected       lipgloss.Style
	unselected     lipgloss.Style
}

// renderEntry renders a single sidebar entry (catalog header or runbook line).
func (m Model) renderEntry(e entry, s entryStyles, isCursor, isHovered, inSelectedCat, isLastRunbook bool) string {
	if e.isHeader {
		indicator := "▼"
		if m.collapsed[e.catalog.Name] {
			indicator = "▶"
		}
		label := indicator + " " + e.catalog.Label() + "/"

		switch {
		case isCursor:
			return s.cursor.Render(label)
		case isHovered:
			return s.hover.Render(label)
		case inSelectedCat:
			return s.headerSelected.Render(label)
		default:
			return s.header.Render(label)
		}
	}

	connector := "├── "
	if isLastRunbook {
		connector = "└── "
	}

	switch {
	case isCursor:
		return s.selected.Render(connector + e.runbook.Name)
	case isHovered:
		return s.hover.Render(connector + e.runbook.Name)
	default:
		return s.unselected.Render(connector + e.runbook.Name)
	}
}

func (m Model) buildLines(vis []int) []string {
	// Determine selected entry
	selectedIdx := -1
	selectedCat := ""
	if m.cursor >= 0 && m.cursor < len(vis) {
		selectedIdx = vis[m.cursor]
		selectedCat = m.entries[selectedIdx].catalog.Name
	}

	// Determine hovered entry index
	hoveredIdx := -1
	if m.hoverIdx >= 0 && m.hoverIdx < len(vis) {
		hoveredIdx = vis[m.hoverIdx]
	}

	// Styles
	headerStyle := lipgloss.NewStyle()
	headerSelectedStyle := lipgloss.NewStyle()
	hoverStyle := lipgloss.NewStyle().Underline(true)
	cursorStyle := lipgloss.NewStyle().Bold(true)
	selectedStyle := lipgloss.NewStyle().Bold(true)
	unselectedStyle := lipgloss.NewStyle()

	if m.styles != nil {
		headerStyle = m.styles.TextMuted
		headerSelectedStyle = m.styles.Primary
		hoverStyle = m.styles.Text.Underline(true)
		cursorStyle = m.styles.Primary.Bold(true)
		selectedStyle = m.styles.Text.Bold(true)
		unselectedStyle = m.styles.TextMuted
	}

	// Count runbooks per catalog to determine tree connectors
	catRunbookCount := make(map[string]int)
	catRunbookSeen := make(map[string]int)
	for _, idx := range vis {
		e := m.entries[idx]
		if !e.isHeader {
			catRunbookCount[e.catalog.Name]++
		}
	}

	styles := entryStyles{
		header:         headerStyle,
		headerSelected: headerSelectedStyle,
		hover:          hoverStyle,
		cursor:         cursorStyle,
		selected:       selectedStyle,
		unselected:     unselectedStyle,
	}

	var lines []string
	for _, idx := range vis {
		e := m.entries[idx]
		isHovered := idx == hoveredIdx && idx != selectedIdx

		if !e.isHeader {
			catRunbookSeen[e.catalog.Name]++
		}
		isLast := catRunbookSeen[e.catalog.Name] == catRunbookCount[e.catalog.Name]

		lines = append(lines, m.renderEntry(e, styles, idx == selectedIdx, isHovered, e.catalog.Name == selectedCat, isLast))
	}

	return lines
}

func (m *Model) ensureVisible() {
	if m.height <= 0 {
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.height {
		m.offset = m.cursor - m.height + 1
	}
}

// tabLabels returns the labels for the catalog tab bar.
// Returns nil when there is only one catalog (tab bar hidden).
func (m Model) tabLabels() []string {
	if len(m.catalogs) <= 1 {
		return nil
	}
	labels := make([]string, 0, len(m.catalogs)+1)
	labels = append(labels, "All")
	for _, cwr := range m.catalogs {
		labels = append(labels, cwr.Catalog.Label())
	}
	return labels
}

// activeCatalogName returns the name of the active catalog, or "" for All.
func (m Model) activeCatalogName() string {
	if m.activeCatalog == 0 || m.activeCatalog > len(m.catalogs) {
		return ""
	}
	return m.catalogs[m.activeCatalog-1].Catalog.Name
}

// switchCatalog cycles the active catalog by delta (+1 or -1) with wraparound.
func (m Model) switchCatalog(delta int) (Model, tea.Cmd) {
	if len(m.catalogs) <= 1 {
		return m, nil
	}
	count := len(m.catalogs) + 1 // All + each catalog
	m.activeCatalog = (m.activeCatalog + delta + count) % count
	m.cursor = 0
	m.offset = 0
	m.searchQuery = ""
	m.searching = false

	name := m.activeCatalogName()
	selCmd := m.selectionCmd()

	return m, tea.Batch(
		func() tea.Msg { return CatalogSwitchedMsg{CatalogName: name} },
		selCmd,
	)
}

func (m Model) Selected() *domain.Runbook {
	vis := m.visible()
	if m.cursor < 0 || m.cursor >= len(vis) {
		return nil
	}
	e := m.entries[vis[m.cursor]]
	if e.isHeader {
		return nil
	}
	rb := e.runbook
	return &rb
}

func (m Model) SelectedCatalog() *domain.Catalog {
	vis := m.visible()
	if m.cursor < 0 || m.cursor >= len(vis) {
		return nil
	}
	cat := m.entries[vis[m.cursor]].catalog
	return &cat
}

// selectionCmd emits a RunbookSelectedMsg if cursor is on a runbook.
func (m Model) selectionCmd() tea.Cmd {
	sel := m.Selected()
	if sel == nil {
		return nil
	}
	cat := m.SelectedCatalog()
	if cat == nil {
		return nil
	}
	rb := *sel
	c := *cat
	return func() tea.Msg {
		return RunbookSelectedMsg{Runbook: rb, Catalog: c}
	}
}


// Test query methods — expose internal state for assertions.
func (m Model) Cursor() int                      { return m.cursor }
func (m Model) HoverIdx() int                    { return m.hoverIdx }
func (m Model) IsCollapsed(catalog string) bool   { return m.collapsed[catalog] }
func (m Model) IsSearching() bool                 { return m.searching }
func (m Model) Visible() []int                    { return m.visible() }
func (m Model) EntryIsHeader(idx int) bool        { return m.entries[idx].isHeader }
func (m Model) VisibleRunbooks() []domain.Runbook { return m.visibleRunbooks() }
func (m Model) ActiveCatalog() int                { return m.activeCatalog }
func (m Model) TabLabels() []string               { return m.tabLabels() }
func (m Model) ActiveCatalogName() string         { return m.activeCatalogName() }

func (m Model) visibleRunbooks() []domain.Runbook {
	vis := m.visible()
	var result []domain.Runbook
	for _, idx := range vis {
		e := m.entries[idx]
		if !e.isHeader {
			result = append(result, e.runbook)
		}
	}
	return result
}
