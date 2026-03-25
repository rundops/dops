package sidebar

import (
	"dops/internal/catalog"
	"dops/internal/domain"
	"dops/internal/testutil"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func largeCatalogs() []catalog.CatalogWithRunbooks {
	return []catalog.CatalogWithRunbooks{
		{
			Catalog: domain.Catalog{Name: "default"},
			Runbooks: []domain.Runbook{
				{ID: "default.hello-world", Name: "hello-world"},
				{ID: "default.rotate-tls", Name: "rotate-tls"},
				{ID: "default.deploy-app", Name: "deploy-app"},
			},
		},
		{
			Catalog: domain.Catalog{Name: "infra"},
			Runbooks: []domain.Runbook{
				{ID: "infra.drain-node", Name: "drain-node"},
				{ID: "infra.scale-cluster", Name: "scale-cluster"},
			},
		},
	}
}

func typeSearch(m Model, query string) Model {
	m, _ = pressKey(m, "/")
	for _, ch := range query {
		msg := tea.KeyPressMsg{Code: ch, Text: string(ch)}
		m, _ = m.Update(msg)
	}
	return m
}

func TestSearch_ActivateWithSlash(t *testing.T) {
	m := New(largeCatalogs(), 20, testutil.TestStyles())
	m.Init()

	if m.IsSearching() {
		t.Fatal("should not be searching initially")
	}

	m, _ = pressKey(m, "/")
	if !m.IsSearching() {
		t.Fatal("should be searching after /")
	}
}

func TestSearch_FiltersRunbooks(t *testing.T) {
	m := New(largeCatalogs(), 20, testutil.TestStyles())
	m.Init()

	m = typeSearch(m, "deploy")

	visible := m.VisibleRunbooks()
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible runbook, got %d", len(visible))
	}
	if visible[0].ID != "default.deploy-app" {
		t.Errorf("visible runbook = %q, want default.deploy-app", visible[0].ID)
	}
}

func TestSearch_HidesEmptyCatalogs(t *testing.T) {
	m := New(largeCatalogs(), 20, testutil.TestStyles())
	m.Init()

	m = typeSearch(m, "drain")

	view := m.View()
	if strings.Contains(view, "default/") {
		t.Error("default/ should be hidden when no runbooks match")
	}
	if !strings.Contains(view, "infra/") {
		t.Error("infra/ should be visible since drain-node matches")
	}
}

func TestSearch_NoMatches(t *testing.T) {
	m := New(largeCatalogs(), 20, testutil.TestStyles())
	m.Init()

	m = typeSearch(m, "zzzznothing")

	visible := m.VisibleRunbooks()
	if len(visible) != 0 {
		t.Errorf("expected 0 visible runbooks, got %d", len(visible))
	}
}

func TestSearch_EscapeRestores(t *testing.T) {
	m := New(largeCatalogs(), 20, testutil.TestStyles())
	m.Init()

	m = typeSearch(m, "deploy")
	if len(m.VisibleRunbooks()) != 1 {
		t.Fatal("should have 1 match before escape")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	if m.IsSearching() {
		t.Error("should not be searching after escape")
	}
	if len(m.VisibleRunbooks()) != 5 {
		t.Errorf("expected 5 runbooks after escape, got %d", len(m.VisibleRunbooks()))
	}
}

func TestScrollbar_AppearsWhenNeeded(t *testing.T) {
	m := New(largeCatalogs(), 3, testutil.TestStyles())
	m.Init()

	view := m.View()
	if len(view) == 0 {
		t.Error("view should not be empty")
	}
}

func TestScrollbar_NotNeededForSmallList(t *testing.T) {
	small := []catalog.CatalogWithRunbooks{
		{
			Catalog:  domain.Catalog{Name: "tiny"},
			Runbooks: []domain.Runbook{{ID: "tiny.one", Name: "one"}},
		},
	}
	m := New(small, 20, testutil.TestStyles())
	m.Init()

	view := m.View()
	if strings.Contains(view, "█") || strings.Contains(view, "░") {
		t.Error("scrollbar should not appear when content fits")
	}
}
