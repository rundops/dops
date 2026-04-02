package sidebar

import (
	"dops/internal/catalog"
	"dops/internal/domain"
	"dops/internal/testutil"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func multiCatalogs() []catalog.CatalogWithRunbooks {
	return []catalog.CatalogWithRunbooks{
		{
			Catalog: domain.Catalog{Name: "infra", DisplayName: "Infrastructure"},
			Runbooks: []domain.Runbook{
				{ID: "infra.scale", Name: "scale-deployment"},
				{ID: "infra.restart", Name: "restart-pods"},
			},
		},
		{
			Catalog: domain.Catalog{Name: "demo"},
			Runbooks: []domain.Runbook{
				{ID: "demo.hello", Name: "hello-world"},
			},
		},
	}
}

func singleCatalog() []catalog.CatalogWithRunbooks {
	return []catalog.CatalogWithRunbooks{
		{
			Catalog:  domain.Catalog{Name: "only"},
			Runbooks: []domain.Runbook{{ID: "only.one", Name: "one"}},
		},
	}
}

func pressCtrl(m Model, key string) (Model, tea.Cmd) {
	msg := tea.KeyPressMsg{Code: rune(key[0]), Text: key, Mod: tea.ModCtrl}
	return m.Update(msg)
}

func TestTabs_HiddenWithSingleCatalog(t *testing.T) {
	m := New(singleCatalog(), 20, testutil.TestStyles())

	if labels := m.TabLabels(); labels != nil {
		t.Errorf("single catalog should hide tabs, got %v", labels)
	}

	view := m.View()
	if strings.Contains(view, "All") {
		t.Error("tab bar should not render with single catalog")
	}
}

func TestTabs_ShownWithMultipleCatalogs(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	labels := m.TabLabels()
	if len(labels) != 3 {
		t.Fatalf("expected 3 tabs (All + 2 catalogs), got %d", len(labels))
	}
	if labels[0] != "All" {
		t.Errorf("first tab = %q, want All", labels[0])
	}
	if labels[1] != "Infrastructure" {
		t.Errorf("second tab = %q, want Infrastructure (display name)", labels[1])
	}
	if labels[2] != "demo" {
		t.Errorf("third tab = %q, want demo", labels[2])
	}
}

func TestTabs_DefaultIsAll(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	if m.ActiveCatalog() != 0 {
		t.Errorf("default active catalog = %d, want 0 (All)", m.ActiveCatalog())
	}
	if m.ActiveCatalogName() != "" {
		t.Errorf("active name = %q, want empty (All)", m.ActiveCatalogName())
	}
}

func TestTabs_AllTabShowsHeaders(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	vis := m.Visible()
	hasHeader := false
	for _, idx := range vis {
		if m.EntryIsHeader(idx) {
			hasHeader = true
			break
		}
	}
	if !hasHeader {
		t.Error("All tab should include catalog headers")
	}
}

func TestTabs_SwitchToSpecificCatalog(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	// Switch to "infra" (index 1)
	m, cmd := m.switchCatalog(1)
	if m.ActiveCatalog() != 1 {
		t.Errorf("active = %d, want 1", m.ActiveCatalog())
	}
	if m.ActiveCatalogName() != "infra" {
		t.Errorf("name = %q, want infra", m.ActiveCatalogName())
	}

	// Should emit CatalogSwitchedMsg
	if cmd == nil {
		t.Fatal("switch should emit command")
	}

	// Visible entries should be runbooks only (no headers)
	vis := m.Visible()
	for _, idx := range vis {
		if m.EntryIsHeader(idx) {
			t.Error("specific catalog tab should not show headers")
		}
	}

	runbooks := m.VisibleRunbooks()
	if len(runbooks) != 2 {
		t.Fatalf("expected 2 infra runbooks, got %d", len(runbooks))
	}
	if runbooks[0].ID != "infra.scale" {
		t.Errorf("first runbook = %q, want infra.scale", runbooks[0].ID)
	}
}

func TestTabs_SwitchToDemoCatalog(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	// Switch forward twice: All → infra → demo
	m, _ = m.switchCatalog(1)
	m, _ = m.switchCatalog(1)

	if m.ActiveCatalogName() != "demo" {
		t.Errorf("name = %q, want demo", m.ActiveCatalogName())
	}

	runbooks := m.VisibleRunbooks()
	if len(runbooks) != 1 || runbooks[0].ID != "demo.hello" {
		t.Errorf("expected demo.hello, got %v", runbooks)
	}
}

func TestTabs_Wraparound(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	// Backward from All (0) should wrap to last catalog
	m, _ = m.switchCatalog(-1)
	if m.ActiveCatalog() != 2 {
		t.Errorf("active = %d, want 2 (demo)", m.ActiveCatalog())
	}

	// Forward should wrap back to All
	m, _ = m.switchCatalog(1)
	if m.ActiveCatalog() != 0 {
		t.Errorf("active = %d, want 0 (All)", m.ActiveCatalog())
	}
}

func TestTabs_SwitchResetsCursorAndSearch(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	// Navigate down and start a search
	m, _ = pressKey(m, "down")
	m = typeSearch(m, "scale")

	if m.Cursor() == 0 && !m.IsSearching() {
		t.Fatal("precondition: cursor moved and search active")
	}

	// Switch catalog
	m, _ = m.switchCatalog(1)

	if m.Cursor() != 0 {
		t.Errorf("cursor = %d, want 0 after switch", m.Cursor())
	}
	if m.IsSearching() {
		t.Error("searching should be cleared after switch")
	}
}

func TestTabs_SearchScopedToActiveCatalog(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	// Switch to infra
	m, _ = m.switchCatalog(1)

	// Search for "hello" — exists in demo but not infra
	m = typeSearch(m, "hello")
	runbooks := m.VisibleRunbooks()
	if len(runbooks) != 0 {
		t.Errorf("expected 0 matches in infra for 'hello', got %d", len(runbooks))
	}

	// Switch to All and search again
	m, _ = m.switchCatalog(-1) // back to All
	m = typeSearch(m, "hello")
	runbooks = m.VisibleRunbooks()
	if len(runbooks) != 1 {
		t.Errorf("expected 1 match in All for 'hello', got %d", len(runbooks))
	}
}

func TestTabs_CtrlHSwitchesLeft(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	// Start at All (0), Ctrl+H should go to demo (2, wraparound)
	m, _ = pressCtrl(m, "ctrl+h")
	if m.ActiveCatalog() != 2 {
		t.Errorf("active = %d, want 2 after ctrl+h", m.ActiveCatalog())
	}
}

func TestTabs_CtrlLSwitchesRight(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	// Start at All (0), Ctrl+L should go to infra (1)
	m, _ = pressCtrl(m, "ctrl+l")
	if m.ActiveCatalog() != 1 {
		t.Errorf("active = %d, want 1 after ctrl+l", m.ActiveCatalog())
	}
}

func TestTabs_LeftRightOnSpecificCatalog(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	// Switch to infra first
	m, _ = m.switchCatalog(1)
	if m.ActiveCatalog() != 1 {
		t.Fatal("precondition: should be on infra")
	}

	// Right arrow should cycle to next catalog (not expand header)
	m, _ = pressKey(m, "right")
	if m.ActiveCatalog() != 2 {
		t.Errorf("active = %d, want 2 after right arrow", m.ActiveCatalog())
	}

	// Left arrow should cycle back
	m, _ = pressKey(m, "left")
	if m.ActiveCatalog() != 1 {
		t.Errorf("active = %d, want 1 after left arrow", m.ActiveCatalog())
	}
}

func TestTabs_LeftRightOnAllTab_CollapseExpand(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	// On All tab, left/right should still collapse/expand (existing behavior)
	if m.ActiveCatalog() != 0 {
		t.Fatal("precondition: should be on All")
	}

	// Move to header, left collapses
	m, _ = pressKey(m, "up") // to infra/ header
	m, _ = pressKey(m, "left")
	if !m.IsCollapsed("infra") {
		t.Error("left on All tab header should collapse")
	}

	// Right expands
	m, _ = pressKey(m, "right")
	if m.IsCollapsed("infra") {
		t.Error("right on All tab header should expand")
	}
}

func TestTabs_ViewRendersTabBar(t *testing.T) {
	m := New(multiCatalogs(), 20, testutil.TestStyles())

	view := m.View()
	if !strings.Contains(view, "All") {
		t.Error("view should contain All tab")
	}
	if !strings.Contains(view, "Infrastructure") {
		t.Error("view should contain Infrastructure tab (display name)")
	}
	if !strings.Contains(view, "demo") {
		t.Error("view should contain demo tab")
	}
}

func TestTabs_NoSwitchOnSingleCatalog(t *testing.T) {
	m := New(singleCatalog(), 20, testutil.TestStyles())

	m, cmd := m.switchCatalog(1)
	if cmd != nil {
		t.Error("switch on single catalog should be no-op")
	}
	if m.ActiveCatalog() != 0 {
		t.Errorf("active = %d, want 0", m.ActiveCatalog())
	}
}
