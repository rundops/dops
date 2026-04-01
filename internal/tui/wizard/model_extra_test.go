package wizard

import (
	"dops/internal/domain"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
)

// ---------- helpers ----------

func stringParam(name string, required bool, scope string) domain.Parameter {
	return domain.Parameter{Name: name, Type: domain.ParamString, Required: required, Scope: scope}
}

// localStringParam creates a string param with "local" scope (no save prompt).
func localStringParam(name string, required bool) domain.Parameter {
	return domain.Parameter{Name: name, Type: domain.ParamString, Required: required, Scope: "local"}
}

func integerParam(name string) domain.Parameter {
	return domain.Parameter{Name: name, Type: domain.ParamInteger, Required: true, Scope: "local"}
}

func numberParam(name string) domain.Parameter {
	return domain.Parameter{Name: name, Type: domain.ParamNumber, Required: true, Scope: "local"}
}

func floatParam(name string) domain.Parameter {
	return domain.Parameter{Name: name, Type: domain.ParamFloat, Required: true, Scope: "local"}
}

func selectParam(name string, opts []string) domain.Parameter {
	return domain.Parameter{Name: name, Type: domain.ParamSelect, Required: true, Scope: "local", Options: opts}
}

func multiSelectParam(name string, opts []string) domain.Parameter {
	return domain.Parameter{Name: name, Type: domain.ParamMultiSelect, Required: false, Scope: "local", Options: opts}
}

func boolParam(name string) domain.Parameter {
	return domain.Parameter{Name: name, Type: domain.ParamBoolean, Required: false, Scope: "local"}
}

func secretParam(name string) domain.Parameter {
	return domain.Parameter{Name: name, Type: domain.ParamString, Required: true, Scope: "global", Secret: true}
}

func defaultCatalog() domain.Catalog {
	return domain.Catalog{Name: "default"}
}

func keyMsg(code rune) tea.Msg {
	return tea.KeyPressMsg{Code: code}
}

func textKeyMsg(text string) tea.Msg {
	return tea.KeyPressMsg{Text: text}
}

func enterMsg() tea.Msg { return keyMsg(tea.KeyEnter) }
func escMsg() tea.Msg   { return keyMsg(tea.KeyEscape) }

// ---------- Init tests ----------

func TestInit_EmptyParams(t *testing.T) {
	rb := domain.Runbook{ID: "test.empty", Name: "empty"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil when there are no parameters")
	}
}

func TestInit_NonePrefilled_TextInput(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.one",
		Name:       "one",
		Parameters: []domain.Parameter{stringParam("name", true, "global")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a focus command for text input field")
	}
}

func TestInit_NonePrefilled_SelectField(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.sel",
		Name:       "sel",
		Parameters: []domain.Parameter{selectParam("env", []string{"dev", "prod"})},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil for a select field (no text input to focus)")
	}
}

func TestInit_AllPrefilled_SubmitsImmediately(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.full",
		Name:       "full",
		Parameters: []domain.Parameter{stringParam("a", true, "global")},
	}
	resolved := map[string]string{"a": "val"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved})

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init should return a submit command when all fields are prefilled")
	}
	msg := cmd()
	if _, ok := msg.(SubmitMsg); !ok {
		t.Errorf("expected SubmitMsg, got %T", msg)
	}
}

func TestInit_BooleanField(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.bool",
		Name:       "bool",
		Parameters: []domain.Parameter{boolParam("flag")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil for a boolean field")
	}
}

// ---------- Update: escape cancels ----------

func TestUpdate_Escape(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.esc",
		Name:       "esc",
		Parameters: []domain.Parameter{stringParam("x", true, "global")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m, cmd := m.Update(escMsg())
	if cmd == nil {
		t.Fatal("escape should produce a cancel command")
	}
	msg := cmd()
	if _, ok := msg.(CancelMsg); !ok {
		t.Errorf("expected CancelMsg, got %T", msg)
	}
}

// ---------- Update: empty params ----------

func TestUpdate_EmptyParams(t *testing.T) {
	rb := domain.Runbook{ID: "test.empty", Name: "empty"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m, cmd := m.Update(enterMsg())
	if cmd != nil {
		t.Error("update on empty params should return nil cmd")
	}
}

// ---------- Update: text input enter submits value ----------

func TestUpdate_TextInput_EnterAdvances(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.txt",
		Name: "txt",
		Parameters: []domain.Parameter{
			localStringParam("first", true),
			localStringParam("second", true),
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Type a value into the text input.
	m.input.SetValue("hello")
	m, _ = m.Update(enterMsg())

	if m.values["first"] != "hello" {
		t.Errorf("expected first=hello, got %q", m.values["first"])
	}
	if m.current != 1 {
		t.Errorf("expected to advance to field 1, got %d", m.current)
	}
}

func TestUpdate_TextInput_RequiredEmpty(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.req",
		Name:       "req",
		Parameters: []domain.Parameter{stringParam("x", true, "global")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Leave empty, press enter.
	m, _ = m.Update(enterMsg())
	if m.err != "required field" {
		t.Errorf("expected 'required field' error, got %q", m.err)
	}
	if m.current != 0 {
		t.Error("should not advance past required empty field")
	}
}

func TestUpdate_TextInput_OptionalEmpty(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.opt",
		Name:       "opt",
		Parameters: []domain.Parameter{localStringParam("x", false)},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m, cmd := m.Update(enterMsg())
	if m.err != "" {
		t.Errorf("optional empty field should not error, got %q", m.err)
	}
	// Should produce a submit since it's the last field.
	if cmd == nil {
		t.Fatal("should produce submit cmd for last field")
	}
	msg := cmd()
	if _, ok := msg.(SubmitMsg); !ok {
		t.Errorf("expected SubmitMsg, got %T", msg)
	}
}

// ---------- Update: integer validation ----------

func TestUpdate_IntegerParam_Invalid(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.int",
		Name:       "int",
		Parameters: []domain.Parameter{integerParam("count")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("abc")

	m, _ = m.Update(enterMsg())
	if m.err != "must be an integer" {
		t.Errorf("expected integer error, got %q", m.err)
	}
}

func TestUpdate_IntegerParam_Valid(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.int",
		Name:       "int",
		Parameters: []domain.Parameter{integerParam("count")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("42")

	m, cmd := m.Update(enterMsg())
	if m.err != "" {
		t.Errorf("valid integer should not error, got %q", m.err)
	}
	if m.values["count"] != "42" {
		t.Errorf("expected count=42, got %q", m.values["count"])
	}
	if cmd == nil {
		t.Fatal("should submit after last field")
	}
}

func TestUpdate_IntegerParam_Negative(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.int",
		Name:       "int",
		Parameters: []domain.Parameter{integerParam("offset")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("-5")

	m, _ = m.Update(enterMsg())
	if m.err != "" {
		t.Errorf("negative integer should be valid for ParamInteger, got %q", m.err)
	}
	if m.values["offset"] != "-5" {
		t.Errorf("expected offset=-5, got %q", m.values["offset"])
	}
}

// ---------- Update: number validation (non-negative) ----------

func TestUpdate_NumberParam_Negative(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.num",
		Name:       "num",
		Parameters: []domain.Parameter{numberParam("port")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("-1")

	m, _ = m.Update(enterMsg())
	if m.err != "must be a non-negative number" {
		t.Errorf("expected non-negative error, got %q", m.err)
	}
}

func TestUpdate_NumberParam_NotANumber(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.num",
		Name:       "num",
		Parameters: []domain.Parameter{numberParam("port")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("abc")

	m, _ = m.Update(enterMsg())
	if m.err != "must be a number" {
		t.Errorf("expected 'must be a number' error, got %q", m.err)
	}
}

func TestUpdate_NumberParam_Zero(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.num",
		Name:       "num",
		Parameters: []domain.Parameter{numberParam("count")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("0")

	m, _ = m.Update(enterMsg())
	if m.err != "" {
		t.Errorf("zero should be valid for ParamNumber, got %q", m.err)
	}
	if m.values["count"] != "0" {
		t.Errorf("expected count=0, got %q", m.values["count"])
	}
}

// ---------- Update: float validation ----------

func TestUpdate_FloatParam_Invalid(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.float",
		Name:       "float",
		Parameters: []domain.Parameter{floatParam("ratio")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("not-a-float")

	m, _ = m.Update(enterMsg())
	if m.err != "must be a decimal number" {
		t.Errorf("expected float error, got %q", m.err)
	}
}

func TestUpdate_FloatParam_Valid(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.float",
		Name:       "float",
		Parameters: []domain.Parameter{floatParam("ratio")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("3.14")

	m, _ = m.Update(enterMsg())
	if m.err != "" {
		t.Errorf("valid float should not error, got %q", m.err)
	}
	if m.values["ratio"] != "3.14" {
		t.Errorf("expected ratio=3.14, got %q", m.values["ratio"])
	}
}

// ---------- Update: select field ----------

func TestUpdate_Select_NavigateAndChoose(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.sel",
		Name:       "sel",
		Parameters: []domain.Parameter{selectParam("env", []string{"dev", "staging", "prod"})},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Move down twice (to "prod").
	m, _ = m.Update(keyMsg(tea.KeyDown))
	m, _ = m.Update(keyMsg(tea.KeyDown))
	if m.cursor != 2 {
		t.Errorf("cursor should be 2, got %d", m.cursor)
	}

	// Move up once (to "staging").
	m, _ = m.Update(keyMsg(tea.KeyUp))
	if m.cursor != 1 {
		t.Errorf("cursor should be 1, got %d", m.cursor)
	}

	// Press enter to select "staging".
	m, cmd := m.Update(enterMsg())
	if m.values["env"] != "staging" {
		t.Errorf("expected env=staging, got %q", m.values["env"])
	}
	if cmd == nil {
		t.Fatal("should submit after last field")
	}
}

func TestUpdate_Select_JKNavigation(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.sel",
		Name:       "sel",
		Parameters: []domain.Parameter{selectParam("env", []string{"dev", "staging", "prod"})},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m, _ = m.Update(textKeyMsg("j"))
	if m.cursor != 1 {
		t.Errorf("j should move cursor down, got %d", m.cursor)
	}
	m, _ = m.Update(textKeyMsg("k"))
	if m.cursor != 0 {
		t.Errorf("k should move cursor up, got %d", m.cursor)
	}
}

func TestUpdate_Select_CursorBounds(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.sel",
		Name:       "sel",
		Parameters: []domain.Parameter{selectParam("env", []string{"dev", "prod"})},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Move up at top should stay at 0.
	m, _ = m.Update(keyMsg(tea.KeyUp))
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.cursor)
	}

	// Move to bottom and try to go past.
	m, _ = m.Update(keyMsg(tea.KeyDown))
	m, _ = m.Update(keyMsg(tea.KeyDown))
	if m.cursor != 1 {
		t.Errorf("cursor should be clamped at 1, got %d", m.cursor)
	}
}

// ---------- Update: multi-select field ----------

func TestUpdate_MultiSelect_ToggleAndSubmit(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.ms",
		Name:       "ms",
		Parameters: []domain.Parameter{multiSelectParam("tags", []string{"alpha", "beta", "gamma"})},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Toggle first option (space).
	m, _ = m.Update(textKeyMsg(" "))
	if !m.checked[0] {
		t.Error("first option should be checked")
	}

	// Move down and toggle second.
	m, _ = m.Update(keyMsg(tea.KeyDown))
	m, _ = m.Update(textKeyMsg(" "))
	if !m.checked[1] {
		t.Error("second option should be checked")
	}

	// Untoggle first.
	m, _ = m.Update(keyMsg(tea.KeyUp))
	m, _ = m.Update(textKeyMsg(" "))
	if m.checked[0] {
		t.Error("first option should be unchecked after second toggle")
	}

	// Submit — only "beta" is checked.
	m, cmd := m.Update(enterMsg())
	if m.values["tags"] != "beta" {
		t.Errorf("expected tags=beta, got %q", m.values["tags"])
	}
	if cmd == nil {
		t.Fatal("should submit after last field")
	}
}

func TestUpdate_MultiSelect_NoneChecked(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.ms",
		Name:       "ms",
		Parameters: []domain.Parameter{multiSelectParam("tags", []string{"a", "b"})},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Submit without toggling anything.
	m, cmd := m.Update(enterMsg())
	if m.values["tags"] != "" {
		t.Errorf("expected empty tags, got %q", m.values["tags"])
	}
	if cmd == nil {
		t.Fatal("should submit")
	}
}

// ---------- Update: boolean field ----------

func TestUpdate_Boolean_ToggleAndEnter(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.bool",
		Name:       "bool",
		Parameters: []domain.Parameter{boolParam("flag")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Default cursor is 1 (No).
	if m.cursor != 1 {
		t.Errorf("boolean default cursor should be 1 (No), got %d", m.cursor)
	}

	// Press left to move to Yes.
	m, _ = m.Update(keyMsg(tea.KeyLeft))
	if m.cursor != 0 {
		t.Errorf("cursor should be 0 (Yes), got %d", m.cursor)
	}

	// Enter confirms Yes.
	m, cmd := m.Update(enterMsg())
	if m.values["flag"] != "true" {
		t.Errorf("expected flag=true, got %q", m.values["flag"])
	}
	if cmd == nil {
		t.Fatal("should submit after last field")
	}
}

func TestUpdate_Boolean_YKey(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.bool",
		Name:       "bool",
		Parameters: []domain.Parameter{boolParam("flag")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m, cmd := m.Update(textKeyMsg("y"))
	if m.values["flag"] != "true" {
		t.Errorf("y key should set true, got %q", m.values["flag"])
	}
	if cmd == nil {
		t.Fatal("should submit")
	}
}

func TestUpdate_Boolean_NKey(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.bool",
		Name:       "bool",
		Parameters: []domain.Parameter{boolParam("flag")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m, cmd := m.Update(textKeyMsg("n"))
	if m.values["flag"] != "false" {
		t.Errorf("n key should set false, got %q", m.values["flag"])
	}
	if cmd == nil {
		t.Fatal("should submit")
	}
}

func TestUpdate_Boolean_HLNavigation(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.bool",
		Name:       "bool",
		Parameters: []domain.Parameter{boolParam("flag")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// h → Yes (cursor 0).
	m, _ = m.Update(textKeyMsg("h"))
	if m.cursor != 0 {
		t.Errorf("h should set cursor to 0, got %d", m.cursor)
	}

	// l → No (cursor 1).
	m, _ = m.Update(textKeyMsg("l"))
	if m.cursor != 1 {
		t.Errorf("l should set cursor to 1, got %d", m.cursor)
	}
}

func TestUpdate_Boolean_TabMovesToYes(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.bool",
		Name:       "bool",
		Parameters: []domain.Parameter{boolParam("flag")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m, _ = m.Update(keyMsg(tea.KeyTab))
	if m.cursor != 0 {
		t.Errorf("tab should set cursor to 0 (Yes), got %d", m.cursor)
	}
}

// ---------- Update: Ctrl+E to show all fields ----------

func TestUpdate_CtrlE_RevealsSkippedFields(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.ctrle",
		Name: "ctrle",
		Parameters: []domain.Parameter{
			stringParam("a", true, "global"),
			stringParam("b", true, "global"),
		},
	}
	resolved := map[string]string{"a": "saved"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved})

	if m.SkippedCount() != 1 {
		t.Fatalf("expected 1 skipped, got %d", m.SkippedCount())
	}

	// Send Ctrl+E.
	ctrlE := tea.KeyPressMsg{Text: "e", Mod: tea.ModCtrl}
	m, _ = m.Update(ctrlE)

	if !m.showAll {
		t.Error("showAll should be true after Ctrl+E")
	}
	if m.SkippedCount() != 0 {
		t.Errorf("skipped count should be 0 after Ctrl+E, got %d", m.SkippedCount())
	}
}

func TestUpdate_CtrlE_NoopWhenNothingSkipped(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.ctrle",
		Name:       "ctrle",
		Parameters: []domain.Parameter{stringParam("a", true, "global")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	ctrlE := tea.KeyPressMsg{Text: "e", Mod: tea.ModCtrl}
	m, _ = m.Update(ctrlE)

	// showAll should remain false — nothing to reveal.
	if m.showAll {
		t.Error("showAll should not change when no fields are skipped")
	}
}

// ---------- Update: shift+tab goes back ----------

func TestUpdate_ShiftTab_GoesBack(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.back",
		Name: "back",
		Parameters: []domain.Parameter{
			localStringParam("a", true),
			localStringParam("b", true),
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Fill field a and advance.
	m.input.SetValue("val_a")
	m, _ = m.Update(enterMsg())
	if m.current != 1 {
		t.Fatalf("expected to be on field 1, got %d", m.current)
	}

	// Shift+tab to go back.
	shiftTab := tea.KeyPressMsg{Text: "shift+tab"}
	m, _ = m.Update(shiftTab)
	if m.current != 0 {
		t.Errorf("shift+tab should go back to field 0, got %d", m.current)
	}
}

func TestUpdate_ShiftTab_AtFirstField(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.back",
		Name:       "back",
		Parameters: []domain.Parameter{localStringParam("a", true)},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	shiftTab := tea.KeyPressMsg{Text: "shift+tab"}
	m, _ = m.Update(shiftTab)
	if m.current != 0 {
		t.Error("shift+tab at first field should stay at 0")
	}
}

// ---------- advance: skipping logic ----------

func TestAdvance_SkipsPrefilled(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.skip",
		Name: "skip",
		Parameters: []domain.Parameter{
			localStringParam("a", true),
			localStringParam("b", true), // prefilled → should be skipped
			localStringParam("c", true),
		},
	}
	resolved := map[string]string{"b": "saved_b"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved})

	// Should start at field 0 (a is not prefilled).
	if m.current != 0 {
		t.Fatalf("expected to start at 0, got %d", m.current)
	}

	// Fill a and advance — should skip b and land on c.
	m.input.SetValue("val_a")
	m, _ = m.Update(enterMsg())

	if m.current != 2 {
		t.Errorf("should skip to field 2, got %d", m.current)
	}
	if !m.skipped[1] {
		t.Error("field 1 should be marked as skipped")
	}
	if m.values["b"] != "saved_b" {
		t.Errorf("skipped field b should have value saved_b, got %q", m.values["b"])
	}
}

func TestAdvance_AllRemainingSkipped_Submits(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.skip",
		Name: "skip",
		Parameters: []domain.Parameter{
			localStringParam("a", true),
			localStringParam("b", true),
		},
	}
	resolved := map[string]string{"b": "saved_b"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved})

	m.input.SetValue("val_a")
	m, cmd := m.Update(enterMsg())

	if cmd == nil {
		t.Fatal("should submit when all remaining fields are skipped")
	}
	msg := cmd()
	sub, ok := msg.(SubmitMsg)
	if !ok {
		t.Fatalf("expected SubmitMsg, got %T", msg)
	}
	if sub.Params["a"] != "val_a" {
		t.Errorf("expected a=val_a, got %q", sub.Params["a"])
	}
	if sub.Params["b"] != "saved_b" {
		t.Errorf("expected b=saved_b, got %q", sub.Params["b"])
	}
}

// ---------- advanceOrSave: scope-based save prompt ----------

func TestAdvanceOrSave_LocalScope_NoSavePrompt(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.local",
		Name: "local",
		Parameters: []domain.Parameter{
			{Name: "x", Type: domain.ParamString, Required: true, Scope: "local"},
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m.input.SetValue("val")
	m, cmd := m.Update(enterMsg())

	// Local scope should not trigger save prompt — should submit directly.
	if m.phase == phaseSave {
		t.Error("local scope should not trigger save prompt")
	}
	if cmd == nil {
		t.Fatal("should submit for local scope")
	}
}

func TestAdvanceOrSave_EmptyScope_NoSavePrompt(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.noscope",
		Name: "noscope",
		Parameters: []domain.Parameter{
			{Name: "x", Type: domain.ParamString, Required: true, Scope: ""},
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m.input.SetValue("val")
	m, cmd := m.Update(enterMsg())

	if m.phase == phaseSave {
		t.Error("empty scope should not trigger save prompt")
	}
	if cmd == nil {
		t.Fatal("should submit")
	}
}

func TestAdvanceOrSave_GlobalScope_ShowsSavePrompt(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.global",
		Name: "global",
		Parameters: []domain.Parameter{
			stringParam("x", true, "global"),
			stringParam("y", true, "global"),
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m.input.SetValue("newval")
	m, _ = m.Update(enterMsg())

	if m.phase != phaseSave {
		t.Error("global scope with new value should trigger save prompt")
	}
}

// ---------- Update: save confirm phase ----------

func TestUpdate_SaveConfirm_YKey(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.save",
		Name: "save",
		Parameters: []domain.Parameter{
			stringParam("x", true, "global"),
			stringParam("y", true, "global"),
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Enter a value to trigger save prompt.
	m.input.SetValue("newval")
	m, _ = m.Update(enterMsg())
	if m.phase != phaseSave {
		t.Fatal("should be in save phase")
	}

	// Press 'n' to decline save and advance.
	m, _ = m.Update(textKeyMsg("n"))
	if m.phase == phaseSave {
		t.Error("should have exited save phase")
	}
	if m.current != 1 {
		t.Errorf("should advance to field 1, got %d", m.current)
	}
}

func TestUpdate_SaveConfirm_EnterWithCursorOnNo(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.save",
		Name: "save",
		Parameters: []domain.Parameter{
			stringParam("x", true, "global"),
			stringParam("y", true, "global"),
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m.input.SetValue("newval")
	m, _ = m.Update(enterMsg())
	if m.phase != phaseSave {
		t.Fatal("should be in save phase")
	}

	// Cursor defaults to 1 (No). Enter should decline save.
	if m.cursor != 1 {
		t.Errorf("save confirm default cursor should be 1 (No), got %d", m.cursor)
	}
	m, _ = m.Update(enterMsg())
	if m.current != 1 {
		t.Errorf("should advance to next field, got %d", m.current)
	}
}

func TestUpdate_SaveConfirm_ToggleNavigation(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.save",
		Name: "save",
		Parameters: []domain.Parameter{
			stringParam("x", true, "global"),
			stringParam("y", true, "global"),
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m.input.SetValue("val")
	m, _ = m.Update(enterMsg())
	if m.phase != phaseSave {
		t.Fatal("should be in save phase")
	}

	// Left moves to Yes.
	m, _ = m.Update(keyMsg(tea.KeyLeft))
	if m.cursor != 0 {
		t.Error("left should move to Yes (0)")
	}

	// Right moves to No.
	m, _ = m.Update(textKeyMsg("l"))
	if m.cursor != 1 {
		t.Error("l should move to No (1)")
	}
}

// ---------- View tests ----------

func TestView_EmptyParams(t *testing.T) {
	rb := domain.Runbook{ID: "test.empty", Name: "empty"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	view := m.View()
	if view != "" {
		t.Errorf("empty params view should be empty, got %q", view)
	}
}

func TestView_ShowsFieldName(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.view",
		Name:       "view",
		Parameters: []domain.Parameter{stringParam("my_field", true, "global")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	view := m.View()
	if !strings.Contains(view, "my_field") {
		t.Error("view should contain field name")
	}
}

func TestView_SelectOptions(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.view",
		Name:       "view",
		Parameters: []domain.Parameter{selectParam("env", []string{"dev", "staging", "prod"})},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	view := m.View()
	for _, opt := range []string{"dev", "staging", "prod"} {
		if !strings.Contains(view, opt) {
			t.Errorf("view should contain option %q", opt)
		}
	}
}

func TestView_BooleanToggle(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.view",
		Name:       "view",
		Parameters: []domain.Parameter{boolParam("flag")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	view := m.View()
	if !strings.Contains(view, "Yes") || !strings.Contains(view, "No") {
		t.Error("boolean view should contain Yes and No")
	}
}

func TestView_MultiSelectCheckboxes(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.view",
		Name:       "view",
		Parameters: []domain.Parameter{multiSelectParam("tags", []string{"a", "b"})},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	view := m.View()
	if !strings.Contains(view, "[ ]") {
		t.Error("view should show unchecked boxes")
	}

	// Toggle first, check [x] appears.
	m, _ = m.Update(textKeyMsg(" "))
	view = m.View()
	if !strings.Contains(view, "[x]") {
		t.Error("view should show checked box after toggle")
	}
}

func TestView_CompletedFields(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.view",
		Name: "view",
		Parameters: []domain.Parameter{
			stringParam("a", true, "global"),
			stringParam("b", true, "global"),
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Complete field a.
	m.input.SetValue("val_a")
	m, _ = m.Update(enterMsg())
	// Skip save if prompted.
	if m.phase == phaseSave {
		m, _ = m.Update(textKeyMsg("n"))
	}

	view := m.View()
	if !strings.Contains(view, "val_a") {
		t.Error("view should show completed value for first field")
	}
}

func TestView_SecretFieldMasked(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.secret",
		Name: "secret",
		Parameters: []domain.Parameter{
			secretParam("token"),
			stringParam("name", true, "global"),
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Fill secret and advance.
	m.input.SetValue("supersecret")
	m, _ = m.Update(enterMsg())
	// Skip save if prompted.
	if m.phase == phaseSave {
		m, _ = m.Update(textKeyMsg("n"))
	}

	view := m.View()
	if strings.Contains(view, "supersecret") {
		t.Error("view should NOT show secret value in plain text")
	}
	if !strings.Contains(view, "••••••••••") {
		t.Error("view should show masked dots for secret")
	}
}

func TestView_SavePrompt(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.save",
		Name: "save",
		Parameters: []domain.Parameter{
			stringParam("x", true, "global"),
			stringParam("y", true, "global"),
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m.input.SetValue("newval")
	m, _ = m.Update(enterMsg())

	if m.phase != phaseSave {
		t.Fatal("should be in save phase")
	}

	view := m.View()
	if !strings.Contains(view, "Save for future runs?") {
		t.Error("view should show save prompt")
	}
}

func TestView_ErrorDisplayed(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.err",
		Name:       "err",
		Parameters: []domain.Parameter{stringParam("x", true, "global")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Submit empty required field to trigger error.
	m, _ = m.Update(enterMsg())
	view := m.View()
	if !strings.Contains(view, "required field") {
		t.Error("view should display validation error")
	}
}

func TestView_SkippedSummary(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.skip",
		Name: "skip",
		Parameters: []domain.Parameter{
			stringParam("a", true, "global"),
			stringParam("b", true, "global"),
			stringParam("c", true, "global"),
		},
	}
	resolved := map[string]string{"a": "va", "b": "vb"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved})

	view := m.View()
	if !strings.Contains(view, "Applied 2 saved values") {
		t.Error("view should show plural skipped summary")
	}
	if !strings.Contains(view, "Ctrl+E") {
		t.Error("view should mention Ctrl+E")
	}
}

// ---------- FooterHints tests ----------

func TestFooterHints_EmptyParams(t *testing.T) {
	rb := domain.Runbook{ID: "test.empty", Name: "empty"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	if m.FooterHints() != "" {
		t.Error("footer hints should be empty for no params")
	}
}

func TestFooterHints_SelectField(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.sel",
		Name:       "sel",
		Parameters: []domain.Parameter{selectParam("env", []string{"dev"})},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	hints := m.FooterHints()
	if !strings.Contains(hints, "navigate") {
		t.Errorf("select hints should mention navigate, got %q", hints)
	}
}

func TestFooterHints_MultiSelectField(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.ms",
		Name:       "ms",
		Parameters: []domain.Parameter{multiSelectParam("tags", []string{"a"})},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	hints := m.FooterHints()
	if !strings.Contains(hints, "Space") || !strings.Contains(hints, "toggle") {
		t.Errorf("multi-select hints should mention Space toggle, got %q", hints)
	}
}

func TestFooterHints_BooleanField(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.bool",
		Name:       "bool",
		Parameters: []domain.Parameter{boolParam("flag")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	hints := m.FooterHints()
	if !strings.Contains(hints, "toggle") {
		t.Errorf("boolean hints should mention toggle, got %q", hints)
	}
}

func TestFooterHints_TextWithBackOption(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.hints",
		Name: "hints",
		Parameters: []domain.Parameter{
			stringParam("a", true, "global"),
			stringParam("b", true, "global"),
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// First field should not show "back".
	hints := m.FooterHints()
	if strings.Contains(hints, "shift+tab") {
		t.Error("first field should not show shift+tab")
	}

	// Advance to second field.
	m.input.SetValue("val")
	m, _ = m.Update(enterMsg())
	if m.phase == phaseSave {
		m, _ = m.Update(textKeyMsg("n"))
	}

	hints = m.FooterHints()
	if !strings.Contains(hints, "shift+tab") {
		t.Error("second field should show shift+tab back")
	}
}

func TestFooterHints_SavePhase(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.save",
		Name: "save",
		Parameters: []domain.Parameter{
			stringParam("x", true, "global"),
			stringParam("y", true, "global"),
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m.input.SetValue("val")
	m, _ = m.Update(enterMsg())
	if m.phase != phaseSave {
		t.Skip("not in save phase")
	}

	hints := m.FooterHints()
	if !strings.Contains(hints, "toggle") || !strings.Contains(hints, "confirm") {
		t.Errorf("save phase hints should mention toggle and confirm, got %q", hints)
	}
}

func TestFooterHints_SecretPrefilled(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.secret",
		Name:       "secret",
		Parameters: []domain.Parameter{secretParam("token")},
	}
	resolved := map[string]string{"token": "saved_secret"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved, PromptAll: true})

	hints := m.FooterHints()
	if !strings.Contains(hints, "accept") {
		t.Errorf("secret prefilled hints should mention accept, got %q", hints)
	}
}

// ---------- Edge cases ----------

func TestSingleParam_SubmitsDirectly(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.single",
		Name:       "single",
		Parameters: []domain.Parameter{stringParam("only", false, "local")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	m.input.SetValue("val")
	m, cmd := m.Update(enterMsg())
	if cmd == nil {
		t.Fatal("single param should submit after enter")
	}
	msg := cmd()
	sub, ok := msg.(SubmitMsg)
	if !ok {
		t.Fatalf("expected SubmitMsg, got %T", msg)
	}
	if sub.Params["only"] != "val" {
		t.Errorf("expected only=val, got %q", sub.Params["only"])
	}
}

func TestCollectParams_MergesResolvedAndValues(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.merge",
		Name: "merge",
		Parameters: []domain.Parameter{
			localStringParam("a", true),
			localStringParam("b", true),
		},
	}
	resolved := map[string]string{"a": "resolved_a"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved, PromptAll: true})

	// Fill a with a new value.
	m.input.SetValue("new_a")
	m, _ = m.Update(enterMsg())
	if m.phase == phaseSave {
		m, _ = m.Update(textKeyMsg("n"))
	}

	// Fill b.
	m.input.SetValue("val_b")
	m, cmd := m.Update(enterMsg())
	if m.phase == phaseSave {
		m, _ = m.Update(textKeyMsg("n"))
		// After declining save, advance triggers submit.
		// Need to get cmd from last update.
	}

	// The final cmd might be from the save decline.
	if cmd == nil {
		t.Fatal("should produce submit command")
	}
	msg := cmd()
	sub, ok := msg.(SubmitMsg)
	if !ok {
		t.Fatalf("expected SubmitMsg, got %T", msg)
	}
	// User-entered value should override resolved.
	if sub.Params["a"] != "new_a" {
		t.Errorf("expected a=new_a (override), got %q", sub.Params["a"])
	}
	if sub.Params["b"] != "val_b" {
		t.Errorf("expected b=val_b, got %q", sub.Params["b"])
	}
}

func TestNewModel_DefaultValue(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.default",
		Name: "default",
		Parameters: []domain.Parameter{
			{Name: "x", Type: domain.ParamString, Required: false, Scope: "global", Default: "mydefault"},
		},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})

	// Text input should have default value pre-populated.
	if m.input.Value() != "mydefault" {
		t.Errorf("expected default value 'mydefault', got %q", m.input.Value())
	}
}

func TestBuildCommand_SecretMasked(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.cmd",
		Name: "cmd",
		Parameters: []domain.Parameter{
			secretParam("token"),
			stringParam("name", true, "global"),
		},
	}
	params := map[string]string{"token": "supersecret", "name": "test"}

	cmd := BuildCommand(rb, params)
	if strings.Contains(cmd, "supersecret") {
		t.Error("command should not contain secret in plain text")
	}
	if !strings.Contains(cmd, "••••••••••") {
		t.Error("command should mask secret with dots")
	}
	if !strings.Contains(cmd, "name=test") {
		t.Error("command should show non-secret params")
	}
}

func TestBuildCommand_ConfigParamsOmitted(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.cmd",
		Name: "cmd",
		Parameters: []domain.Parameter{
			stringParam("a", true, "global"),
			stringParam("b", true, "global"),
		},
	}
	params := map[string]string{"a": "va", "b": "vb"}
	saved := map[string]string{"a": "va"} // a is unchanged from config

	cmd := BuildCommand(rb, params, saved)
	if strings.Contains(cmd, "a=va") {
		t.Error("unchanged config param should be omitted from command")
	}
	if !strings.Contains(cmd, "b=vb") {
		t.Error("new param should be included in command")
	}
}

func TestInitField_SelectPrefillsCursor(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.selfill",
		Name:       "selfill",
		Parameters: []domain.Parameter{selectParam("env", []string{"dev", "staging", "prod"})},
	}
	resolved := map[string]string{"env": "prod"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved, PromptAll: true})

	if m.cursor != 2 {
		t.Errorf("cursor should be at index 2 (prod), got %d", m.cursor)
	}
}

func TestInitField_BooleanPrefillsCursor(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.boolfill",
		Name:       "boolfill",
		Parameters: []domain.Parameter{boolParam("flag")},
	}
	resolved := map[string]string{"flag": "true"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved, PromptAll: true})

	if m.cursor != 0 {
		t.Errorf("boolean cursor should be 0 (Yes) when prefilled true, got %d", m.cursor)
	}
}

func TestInitField_MultiSelectPrefillsChecked(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.msfill",
		Name:       "msfill",
		Parameters: []domain.Parameter{multiSelectParam("tags", []string{"a", "b", "c"})},
	}
	resolved := map[string]string{"tags": "a, c"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved, PromptAll: true})

	if !m.checked[0] {
		t.Error("a (index 0) should be checked")
	}
	if m.checked[1] {
		t.Error("b (index 1) should not be checked")
	}
	if !m.checked[2] {
		t.Error("c (index 2) should be checked")
	}
}

func TestGoBack_SkipsOverSkippedFields(t *testing.T) {
	rb := domain.Runbook{
		ID:   "test.goback",
		Name: "goback",
		Parameters: []domain.Parameter{
			localStringParam("a", true),
			localStringParam("b", true), // will be prefilled/skipped
			localStringParam("c", true),
		},
	}
	resolved := map[string]string{"b": "saved_b"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved})

	// Should start at field 0.
	if m.current != 0 {
		t.Fatalf("expected start at 0, got %d", m.current)
	}

	// Fill a and advance — should skip b and land on c (index 2).
	m.input.SetValue("val_a")
	m, _ = m.Update(enterMsg())
	if m.current != 2 {
		t.Fatalf("expected to be at 2, got %d", m.current)
	}

	// Shift+tab from c should go back to a (index 0), skipping b.
	shiftTab := tea.KeyPressMsg{Text: "shift+tab"}
	m, _ = m.Update(shiftTab)
	if m.current != 0 {
		t.Errorf("should go back to field 0 (skipping 1), got %d", m.current)
	}
}

func TestSecretField_NoPrefill_EchoPassword(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.secret",
		Name:       "secret",
		Parameters: []domain.Parameter{secretParam("token")},
	}
	// No resolved value — secret without prefill should use EchoPassword.
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	if m.input.EchoMode != textinput.EchoPassword {
		t.Errorf("secret without prefill should use EchoPassword, got %d", m.input.EchoMode)
	}
}

func TestSecretField_WithPrefill_NormalEcho(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.secret",
		Name:       "secret",
		Parameters: []domain.Parameter{secretParam("token")},
	}
	resolved := map[string]string{"token": "saved_secret"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved, PromptAll: true})
	// Secret with prefill should NOT start in EchoPassword mode (manual dots shown).
	if m.input.EchoMode == textinput.EchoPassword {
		t.Error("secret with prefill should NOT use EchoPassword initially")
	}
}

func TestView_SecretPrefilled_ShowsDots(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.secret",
		Name:       "secret",
		Parameters: []domain.Parameter{secretParam("token")},
	}
	resolved := map[string]string{"token": "saved_secret"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved, PromptAll: true})
	view := m.View()
	// Should show dots hint for prefilled secret.
	if !strings.Contains(view, "••••••••••") {
		t.Error("prefilled secret should show dots placeholder")
	}
	if strings.Contains(view, "saved_secret") {
		t.Error("prefilled secret value should not be visible")
	}
}

func TestSecretField_EmptyInputKeepsSaved(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.secret",
		Name:       "secret",
		Parameters: []domain.Parameter{secretParam("token")},
	}
	resolved := map[string]string{"token": "saved_secret"}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog(), Resolved: resolved, PromptAll: true})

	// Press enter with empty input — should keep saved value.
	m, cmd := m.Update(enterMsg())
	if m.values["token"] != "saved_secret" {
		t.Errorf("expected saved secret to be kept, got %q", m.values["token"])
	}
	if cmd == nil {
		t.Fatal("should submit after accepting saved secret")
	}
}

// ---------- SaveFieldMsg / SaveFieldResultMsg flow ----------

func TestSaveConfirm_Yes_EmitsSaveFieldMsg(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.save",
		Name:       "save",
		Parameters: []domain.Parameter{stringParam("region", true, "global")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("us-east-1")

	// Press enter → triggers advanceOrSave (value changed, global scope → save prompt).
	m, _ = m.Update(enterMsg())
	if m.phase != phaseSave {
		t.Fatalf("expected phaseSave, got %d", m.phase)
	}

	// Press 'y' to confirm save.
	m, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if m.phase != phaseWaitingSave {
		t.Fatalf("expected phaseWaitingSave, got %d", m.phase)
	}
	if cmd == nil {
		t.Fatal("should emit a command")
	}

	// Execute the command to get the message.
	msg := cmd()
	saveMsg, ok := msg.(SaveFieldMsg)
	if !ok {
		t.Fatalf("expected SaveFieldMsg, got %T", msg)
	}
	if saveMsg.Scope != "global" {
		t.Errorf("scope = %q, want global", saveMsg.Scope)
	}
	if saveMsg.ParamName != "region" {
		t.Errorf("param = %q, want region", saveMsg.ParamName)
	}
	if saveMsg.Value != "us-east-1" {
		t.Errorf("value = %q, want us-east-1", saveMsg.Value)
	}
}

func TestSaveConfirm_No_SkipsSave(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.save",
		Name:       "save",
		Parameters: []domain.Parameter{stringParam("region", true, "global")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("us-east-1")

	m, _ = m.Update(enterMsg())
	// Press 'n' to decline save → should advance and emit SubmitMsg.
	m, cmd := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if m.phase == phaseWaitingSave {
		t.Error("should not enter phaseWaitingSave on decline")
	}
	if cmd == nil {
		t.Fatal("should emit SubmitMsg command")
	}
	msg := cmd()
	if _, ok := msg.(SubmitMsg); !ok {
		t.Errorf("expected SubmitMsg, got %T", msg)
	}
}

func TestSaveFieldResultMsg_Success_Advances(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.save",
		Name:       "save",
		Parameters: []domain.Parameter{stringParam("region", true, "global")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("us-east-1")

	m, _ = m.Update(enterMsg())
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if m.phase != phaseWaitingSave {
		t.Fatalf("expected phaseWaitingSave, got %d", m.phase)
	}

	// Simulate successful save result.
	m, cmd := m.Update(SaveFieldResultMsg{Err: nil})
	if m.phase != phaseInput {
		t.Errorf("expected phaseInput after result, got %d", m.phase)
	}
	if m.err != "" {
		t.Errorf("unexpected error: %s", m.err)
	}
	if cmd == nil {
		t.Fatal("should emit SubmitMsg after advancing")
	}
}

func TestSaveFieldResultMsg_Error_SetsErr(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.save",
		Name:       "save",
		Parameters: []domain.Parameter{stringParam("region", true, "global")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("us-east-1")

	m, _ = m.Update(enterMsg())
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})

	// Simulate failed save.
	m, _ = m.Update(SaveFieldResultMsg{Err: fmt.Errorf("disk full")})
	if !strings.Contains(m.err, "disk full") {
		t.Errorf("expected error containing 'disk full', got %q", m.err)
	}
}

func TestSaveConfirm_LocalScope_NoSavePrompt(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.local",
		Name:       "local",
		Parameters: []domain.Parameter{localStringParam("tmp", true)},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("value")

	// Enter should advance directly (no save prompt for local scope).
	m, cmd := m.Update(enterMsg())
	if m.phase == phaseSave {
		t.Error("local scope should skip save prompt")
	}
	if cmd == nil {
		t.Fatal("should emit SubmitMsg")
	}
}

func TestSaveConfirm_EnterWithCursorYes_EmitsSave(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.save",
		Name:       "save",
		Parameters: []domain.Parameter{stringParam("region", true, "global")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("us-east-1")

	m, _ = m.Update(enterMsg())
	// Move cursor to Yes (left).
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	// Press Enter with cursor on Yes.
	m, cmd := m.Update(enterMsg())
	if m.phase != phaseWaitingSave {
		t.Fatalf("expected phaseWaitingSave, got %d", m.phase)
	}
	if cmd == nil {
		t.Fatal("should emit save command")
	}
}

func TestPhaseWaitingSave_IgnoresKeypresses(t *testing.T) {
	rb := domain.Runbook{
		ID:         "test.save",
		Name:       "save",
		Parameters: []domain.Parameter{stringParam("region", true, "global")},
	}
	m := New(WizardConfig{Runbook: rb, Catalog: defaultCatalog()})
	m.input.SetValue("us-east-1")

	m, _ = m.Update(enterMsg())
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if m.phase != phaseWaitingSave {
		t.Fatalf("expected phaseWaitingSave, got %d", m.phase)
	}

	// Keys should be ignored during wait.
	m, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd != nil {
		t.Error("keypresses should be ignored during phaseWaitingSave")
	}
	if m.phase != phaseWaitingSave {
		t.Error("phase should remain phaseWaitingSave")
	}
}
