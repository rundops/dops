package confirm

import (
	"dops/internal/domain"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func testRb(risk domain.RiskLevel) domain.Runbook {
	return domain.Runbook{
		ID:        "default.check-health",
		Name:      "check-health",
		RiskLevel: risk,
	}
}

func testCat() domain.Catalog {
	return domain.Catalog{Name: "default"}
}

func TestConfirm_HighRisk_Y(t *testing.T) {
	m := New(testRb(domain.RiskHigh), testCat(), map[string]string{}, 60, nil)
	m, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd == nil {
		t.Fatal("y should produce a command")
	}
	msg := cmd()
	if _, ok := msg.(AcceptMsg); !ok {
		t.Fatalf("expected AcceptMsg, got %T", msg)
	}
}

func TestConfirm_HighRisk_N(t *testing.T) {
	m := New(testRb(domain.RiskHigh), testCat(), map[string]string{}, 60, nil)
	m, cmd := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if cmd == nil {
		t.Fatal("n should produce a command")
	}
	msg := cmd()
	if _, ok := msg.(CancelMsg); !ok {
		t.Fatalf("expected CancelMsg, got %T", msg)
	}
}

func TestConfirm_CriticalRisk_TypeID(t *testing.T) {
	rb := testRb(domain.RiskCritical)
	m := New(rb, testCat(), map[string]string{}, 60, nil)

	// Type the runbook ID.
	for _, ch := range rb.ID {
		m, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}

	// Enter should accept.
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter after typing ID should produce a command")
	}
	msg := cmd()
	if _, ok := msg.(AcceptMsg); !ok {
		t.Fatalf("expected AcceptMsg, got %T", msg)
	}
}

func TestConfirm_CriticalRisk_WrongID(t *testing.T) {
	m := New(testRb(domain.RiskCritical), testCat(), map[string]string{}, 60, nil)

	for _, ch := range "wrong-id" {
		m, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("enter with wrong ID should not produce a command")
	}
}

func TestConfirm_Escape(t *testing.T) {
	m := New(testRb(domain.RiskCritical), testCat(), map[string]string{}, 60, nil)
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("escape should produce a command")
	}
	msg := cmd()
	if _, ok := msg.(CancelMsg); !ok {
		t.Fatalf("expected CancelMsg, got %T", msg)
	}
}
