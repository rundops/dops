package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dops/internal/domain"
	"dops/internal/executor"

	"github.com/fsnotify/fsnotify"
)

// --- isYAMLFile tests (0% covered) ---

func TestIsYAMLFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"runbook.yaml", true},
		{"runbook.yml", true},
		{"runbook.YAML", true},
		{"runbook.YML", true},
		{"runbook.Yaml", true},
		{"path/to/file.yaml", true},
		{"path/to/file.yml", true},
		{"runbook.json", false},
		{"runbook.txt", false},
		{"runbook", false},
		{"", false},
		{".yaml", true},
		{".yml", true},
		{"file.yaml.bak", false},
		{"yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isYAMLFile(tt.name)
			if got != tt.want {
				t.Errorf("isYAMLFile(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// --- isWriteEvent tests (0% covered) ---

func TestIsWriteEvent(t *testing.T) {
	tests := []struct {
		name string
		op   fsnotify.Op
		want bool
	}{
		{"Write", fsnotify.Write, true},
		{"Create", fsnotify.Create, true},
		{"Remove", fsnotify.Remove, true},
		{"Rename", fsnotify.Rename, true},
		{"Chmod", fsnotify.Chmod, false},
		{"Write|Create", fsnotify.Write | fsnotify.Create, true},
		{"Chmod|Write", fsnotify.Chmod | fsnotify.Write, true},
		{"NoOp", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := fsnotify.Event{Name: "test.yaml", Op: tt.op}
			got := isWriteEvent(e)
			if got != tt.want {
				t.Errorf("isWriteEvent(op=%v) = %v, want %v", tt.op, got, tt.want)
			}
		})
	}
}

// --- validateRiskConfirmation tests (25% covered) ---

func TestValidateRiskConfirmation_LowRisk(t *testing.T) {
	rb := domain.Runbook{ID: "test.rb", RiskLevel: domain.RiskLow}
	err := validateRiskConfirmation(rb, nil)
	if err != nil {
		t.Errorf("low risk should not require confirmation: %v", err)
	}
}

func TestValidateRiskConfirmation_MediumRisk(t *testing.T) {
	rb := domain.Runbook{ID: "test.rb", RiskLevel: domain.RiskMedium}
	err := validateRiskConfirmation(rb, nil)
	if err != nil {
		t.Errorf("medium risk should not require confirmation: %v", err)
	}
}

func TestValidateRiskConfirmation_HighRisk_Correct(t *testing.T) {
	rb := domain.Runbook{ID: "infra.deploy", RiskLevel: domain.RiskHigh}
	args := map[string]any{"_confirm_id": "infra.deploy"}
	err := validateRiskConfirmation(rb, args)
	if err != nil {
		t.Errorf("correct confirm_id should pass: %v", err)
	}
}

func TestValidateRiskConfirmation_HighRisk_Wrong(t *testing.T) {
	rb := domain.Runbook{ID: "infra.deploy", RiskLevel: domain.RiskHigh}
	args := map[string]any{"_confirm_id": "wrong-id"}
	err := validateRiskConfirmation(rb, args)
	if err == nil {
		t.Error("wrong confirm_id should fail")
	}
}

func TestValidateRiskConfirmation_HighRisk_Missing(t *testing.T) {
	rb := domain.Runbook{ID: "infra.deploy", RiskLevel: domain.RiskHigh}
	err := validateRiskConfirmation(rb, map[string]any{})
	if err == nil {
		t.Error("missing confirm_id should fail")
	}
}

func TestValidateRiskConfirmation_HighRisk_NilArgs(t *testing.T) {
	rb := domain.Runbook{ID: "infra.deploy", RiskLevel: domain.RiskHigh}
	err := validateRiskConfirmation(rb, nil)
	if err == nil {
		t.Error("nil args should fail for high risk")
	}
}

func TestValidateRiskConfirmation_CriticalRisk_Correct(t *testing.T) {
	rb := domain.Runbook{ID: "infra.nuke", RiskLevel: domain.RiskCritical}
	args := map[string]any{"_confirm_word": "CONFIRM"}
	err := validateRiskConfirmation(rb, args)
	if err != nil {
		t.Errorf("correct confirm_word should pass: %v", err)
	}
}

func TestValidateRiskConfirmation_CriticalRisk_Wrong(t *testing.T) {
	rb := domain.Runbook{ID: "infra.nuke", RiskLevel: domain.RiskCritical}
	args := map[string]any{"_confirm_word": "confirm"}
	err := validateRiskConfirmation(rb, args)
	if err == nil {
		t.Error("lowercase confirm should fail for critical risk")
	}
}

func TestValidateRiskConfirmation_CriticalRisk_Missing(t *testing.T) {
	rb := domain.Runbook{ID: "infra.nuke", RiskLevel: domain.RiskCritical}
	err := validateRiskConfirmation(rb, map[string]any{})
	if err == nil {
		t.Error("missing confirm_word should fail for critical risk")
	}
}

func TestValidateRiskConfirmation_EmptyRiskLevel(t *testing.T) {
	rb := domain.Runbook{ID: "test.rb", RiskLevel: ""}
	err := validateRiskConfirmation(rb, nil)
	if err != nil {
		t.Errorf("empty risk level should not require confirmation: %v", err)
	}
}

// --- paramToSchemaProperty / RunbookToInputSchema additional tests (39.3% covered) ---

func TestRunbookToInputSchema_NumberType(t *testing.T) {
	rb := domain.Runbook{
		ID: "test.rb",
		Parameters: []domain.Parameter{
			{Name: "replicas", Type: domain.ParamNumber},
		},
	}

	schema, _ := RunbookToInputSchema(rb, nil)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)
	prop := props["replicas"].(map[string]any)
	if prop["type"] != "integer" {
		t.Errorf("number type should map to integer, got %v", prop["type"])
	}
	if prop["minimum"] != float64(0) {
		t.Errorf("number type should have minimum 0, got %v", prop["minimum"])
	}
}

func TestRunbookToInputSchema_FloatType(t *testing.T) {
	rb := domain.Runbook{
		ID: "test.rb",
		Parameters: []domain.Parameter{
			{Name: "ratio", Type: domain.ParamFloat},
		},
	}

	schema, _ := RunbookToInputSchema(rb, nil)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)
	prop := props["ratio"].(map[string]any)
	if prop["type"] != "number" {
		t.Errorf("float type should map to number, got %v", prop["type"])
	}
}

func TestRunbookToInputSchema_MultiSelectType(t *testing.T) {
	rb := domain.Runbook{
		ID: "test.rb",
		Parameters: []domain.Parameter{
			{Name: "regions", Type: domain.ParamMultiSelect, Options: []string{"us", "eu"}},
		},
	}

	schema, _ := RunbookToInputSchema(rb, nil)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)
	prop := props["regions"].(map[string]any)
	if prop["type"] != "array" {
		t.Errorf("multi_select type should map to array, got %v", prop["type"])
	}
	items := prop["items"].(map[string]any)
	if items["type"] != "string" {
		t.Errorf("items type should be string, got %v", items["type"])
	}
}

func TestRunbookToInputSchema_MultiSelectNoOptions(t *testing.T) {
	rb := domain.Runbook{
		ID: "test.rb",
		Parameters: []domain.Parameter{
			{Name: "tags", Type: domain.ParamMultiSelect},
		},
	}

	schema, _ := RunbookToInputSchema(rb, nil)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)
	prop := props["tags"].(map[string]any)
	items := prop["items"].(map[string]any)
	if _, hasEnum := items["enum"]; hasEnum {
		t.Error("multi_select with no options should not have enum in items")
	}
}

func TestRunbookToInputSchema_FilePathType(t *testing.T) {
	rb := domain.Runbook{
		ID: "test.rb",
		Parameters: []domain.Parameter{
			{Name: "config", Type: domain.ParamFilePath},
		},
	}

	schema, _ := RunbookToInputSchema(rb, nil)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)
	prop := props["config"].(map[string]any)
	if prop["type"] != "string" {
		t.Errorf("file_path type should map to string, got %v", prop["type"])
	}
}

func TestRunbookToInputSchema_ResourceIDType(t *testing.T) {
	rb := domain.Runbook{
		ID: "test.rb",
		Parameters: []domain.Parameter{
			{Name: "resource", Type: domain.ParamResourceID},
		},
	}

	schema, _ := RunbookToInputSchema(rb, nil)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)
	prop := props["resource"].(map[string]any)
	if prop["type"] != "string" {
		t.Errorf("resource_id type should map to string, got %v", prop["type"])
	}
	if prop["description"] != "resource identifier" {
		t.Errorf("resource_id should have description 'resource identifier', got %v", prop["description"])
	}
}

func TestRunbookToInputSchema_WithDefault(t *testing.T) {
	rb := domain.Runbook{
		ID: "test.rb",
		Parameters: []domain.Parameter{
			{Name: "env", Type: domain.ParamString, Default: "dev"},
		},
	}

	schema, _ := RunbookToInputSchema(rb, nil)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)
	prop := props["env"].(map[string]any)
	if prop["default"] != "dev" {
		t.Errorf("default should be 'dev', got %v", prop["default"])
	}
}

func TestRunbookToInputSchema_ResolvedParamOptional(t *testing.T) {
	rb := domain.Runbook{
		ID: "test.rb",
		Parameters: []domain.Parameter{
			{Name: "region", Type: domain.ParamString, Required: true, Description: "AWS region"},
		},
	}

	resolved := map[string]string{"region": "us-east-1"}
	schema, _ := RunbookToInputSchema(rb, resolved)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	// Resolved required param should NOT be in required list.
	if req, ok := parsed["required"]; ok {
		reqList := req.([]any)
		for _, r := range reqList {
			if r == "region" {
				t.Error("resolved param 'region' should not be required")
			}
		}
	}
}

func TestRunbookToInputSchema_ResolvedValueAsDefault(t *testing.T) {
	rb := domain.Runbook{
		ID: "test.rb",
		Parameters: []domain.Parameter{
			{Name: "region", Type: domain.ParamString, Required: true},
			{Name: "count", Type: domain.ParamInteger},
		},
	}

	resolved := map[string]string{"region": "us-east-1", "count": "3"}
	schema, _ := RunbookToInputSchema(rb, resolved)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)

	// region should have default from resolved value
	regionProp := props["region"].(map[string]any)
	if regionProp["default"] != "us-east-1" {
		t.Errorf("region default = %v, want us-east-1", regionProp["default"])
	}

	// count should have default from resolved value
	countProp := props["count"].(map[string]any)
	if countProp["default"] != "3" {
		t.Errorf("count default = %v, want 3", countProp["default"])
	}
}

func TestRunbookToInputSchema_NoResolvedVars_NoExtraDefaults(t *testing.T) {
	rb := domain.Runbook{
		ID: "test.rb",
		Parameters: []domain.Parameter{
			{Name: "region", Type: domain.ParamString, Required: true},
		},
	}

	schema, _ := RunbookToInputSchema(rb, nil)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)
	regionProp := props["region"].(map[string]any)
	if _, hasDefault := regionProp["default"]; hasDefault {
		t.Error("param without resolved value should not have default")
	}
}

func TestRunbookToInputSchema_HighRiskConfirmation(t *testing.T) {
	rb := domain.Runbook{
		ID:        "infra.deploy",
		RiskLevel: domain.RiskHigh,
	}

	schema, _ := RunbookToInputSchema(rb, nil)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)
	if _, exists := props["_confirm_id"]; !exists {
		t.Error("high risk should have _confirm_id field")
	}
}

func TestRunbookToInputSchema_SelectNoOptions(t *testing.T) {
	rb := domain.Runbook{
		ID: "test.rb",
		Parameters: []domain.Parameter{
			{Name: "env", Type: domain.ParamSelect},
		},
	}

	schema, _ := RunbookToInputSchema(rb, nil)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)
	prop := props["env"].(map[string]any)
	if _, hasEnum := prop["enum"]; hasEnum {
		t.Error("select with no options should not have enum")
	}
}

func TestRunbookToInputSchema_NoRequired(t *testing.T) {
	rb := domain.Runbook{
		ID: "test.rb",
		Parameters: []domain.Parameter{
			{Name: "optional", Type: domain.ParamString},
		},
	}

	schema, _ := RunbookToInputSchema(rb, nil)
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	if _, ok := parsed["required"]; ok {
		t.Error("schema with no required params should not have required field")
	}
}

// --- generateParamVars tests ---

func TestGenerateParamVars_BashRequired(t *testing.T) {
	params := []domain.Parameter{
		{Name: "endpoint", Type: domain.ParamString, Required: true},
	}
	out := generateParamVars(params, "bash")
	if !strings.Contains(out, `ENDPOINT="${ENDPOINT:?endpoint is required}"`) {
		t.Errorf("bash required param incorrect:\n%s", out)
	}
}

func TestGenerateParamVars_BashOptionalWithDefault(t *testing.T) {
	params := []domain.Parameter{
		{Name: "region", Type: domain.ParamString, Default: "us-east-1"},
	}
	out := generateParamVars(params, "bash")
	if !strings.Contains(out, `REGION="${REGION:-us-east-1}"`) {
		t.Errorf("bash optional with default incorrect:\n%s", out)
	}
}

func TestGenerateParamVars_BashOptionalNoDefault(t *testing.T) {
	params := []domain.Parameter{
		{Name: "tag", Type: domain.ParamString},
	}
	out := generateParamVars(params, "bash")
	if !strings.Contains(out, `TAG="${TAG:-}"`) {
		t.Errorf("bash optional no default incorrect:\n%s", out)
	}
}

func TestGenerateParamVars_BashSecretParam(t *testing.T) {
	params := []domain.Parameter{
		{Name: "api_key", Type: domain.ParamString, Required: true, Secret: true},
	}
	out := generateParamVars(params, "bash")
	if !strings.Contains(out, "# (secret") {
		t.Errorf("bash secret param should have comment:\n%s", out)
	}
}

func TestGenerateParamVars_PowerShellRequired(t *testing.T) {
	params := []domain.Parameter{
		{Name: "endpoint", Type: domain.ParamString, Required: true},
	}
	out := generateParamVars(params, "powershell")
	if !strings.Contains(out, "$env:ENDPOINT") {
		t.Errorf("powershell required param incorrect:\n%s", out)
	}
	if !strings.Contains(out, "throw") {
		t.Errorf("powershell required param should throw:\n%s", out)
	}
}

func TestGenerateParamVars_PowerShellOptional(t *testing.T) {
	params := []domain.Parameter{
		{Name: "region", Type: domain.ParamString, Default: "us-east-1"},
	}
	out := generateParamVars(params, "powershell")
	if !strings.Contains(out, `"us-east-1"`) {
		t.Errorf("powershell optional with default incorrect:\n%s", out)
	}
}

func TestGenerateParamVars_NoParams(t *testing.T) {
	out := generateParamVars(nil, "bash")
	if out != "" {
		t.Errorf("no params should produce empty string, got:\n%s", out)
	}
}

func TestGenerateParamVars_MixedParams(t *testing.T) {
	params := []domain.Parameter{
		{Name: "endpoint", Type: domain.ParamString, Required: true},
		{Name: "region", Type: domain.ParamString, Default: "us-east-1"},
		{Name: "token", Type: domain.ParamString, Required: true, Secret: true},
	}
	out := generateParamVars(params, "bash")
	if !strings.Contains(out, "ENDPOINT") {
		t.Error("missing endpoint var")
	}
	if !strings.Contains(out, "REGION") {
		t.Error("missing region var")
	}
	if !strings.Contains(out, "TOKEN") {
		t.Error("missing token var")
	}
	if !strings.Contains(out, "# (secret") {
		t.Error("missing secret comment")
	}
}

// --- HandleToolCall progress tests ---

func TestHandleToolCall_InvokesOnProgress(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "test", "script.sh")
	os.MkdirAll(filepath.Dir(scriptPath), 0o755)
	os.WriteFile(scriptPath, []byte("#!/bin/sh\necho line1\necho line2\necho line3\necho line4\necho line5\necho line6\n"), 0o755)

	var progressCalls int
	var lastLineCount int
	callback := func(chunk string, linesSoFar int) {
		progressCalls++
		lastLineCount = linesSoFar
	}

	result, err := HandleToolCall(context.Background(), ToolCallRequest{
		Runbook: domain.Runbook{
			Name:      "test",
			Script:    "script.sh",
			RiskLevel: domain.RiskLow,
		},
		Catalog:    domain.Catalog{Name: "default", Path: dir},
		Config:     &domain.Config{},
		Runner:     &fakeRunnerMultiLine{lines: 6},
		Args:       map[string]any{},
		OnProgress: callback,
	})

	if err != nil {
		t.Fatalf("HandleToolCall: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if progressCalls == 0 {
		t.Error("OnProgress should have been called at least once")
	}
	if lastLineCount == 0 {
		t.Error("lastLineCount should be > 0")
	}
}

func TestHandleToolCall_NilProgress_NoPanic(t *testing.T) {
	// Existing behavior: nil OnProgress should not panic.
	result, err := HandleToolCall(context.Background(), ToolCallRequest{
		Runbook: domain.Runbook{
			Name:      "test",
			Script:    "script.sh",
			RiskLevel: domain.RiskLow,
		},
		Catalog:    domain.Catalog{Name: "default", Path: t.TempDir()},
		Config:     &domain.Config{},
		Runner:     &fakeRunner{},
		Args:       map[string]any{},
		OnProgress: nil,
	})
	if err != nil {
		t.Fatalf("HandleToolCall: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

type fakeRunnerMultiLine struct {
	lines int
}

func (r *fakeRunnerMultiLine) Run(ctx context.Context, scriptPath string, env map[string]string) (<-chan executor.OutputLine, <-chan error) {
	ch := make(chan executor.OutputLine, r.lines)
	errs := make(chan error, 1)
	go func() {
		for i := 0; i < r.lines; i++ {
			ch <- executor.OutputLine{Text: fmt.Sprintf("line-%d", i+1)}
		}
		close(ch)
		errs <- nil
	}()
	return ch, errs
}

// --- collectResult tests ---

func TestCollectResult_TruncatesOutput(t *testing.T) {
	// Build exactly maxToolOutputLines+5 lines.
	lines := make([]string, maxToolOutputLines+5)
	for i := range lines {
		lines[i] = fmt.Sprintf("line-%d", i)
	}

	result := collectResult(lines, time.Now(), "/tmp/test.log", nil)

	outputLines := strings.Split(result.Output, "\n")
	// Output should be last maxToolOutputLines lines joined.
	if !strings.Contains(result.Output, fmt.Sprintf("line-%d", maxToolOutputLines+4)) {
		t.Error("truncated output should contain last line")
	}
	if strings.Contains(result.Output, "line-0") {
		t.Error("truncated output should not contain first line")
	}
	// Verify boundary: exactly maxToolOutputLines should NOT truncate.
	exactLines := make([]string, maxToolOutputLines)
	for i := range exactLines {
		exactLines[i] = fmt.Sprintf("exact-%d", i)
	}
	exactResult := collectResult(exactLines, time.Now(), "", nil)
	if !strings.Contains(exactResult.Output, "exact-0") {
		t.Error("exactly maxToolOutputLines should not truncate — first line should be present")
	}
	_ = outputLines
}

func TestCollectResult_ExitCode(t *testing.T) {
	// No error → exit code 0.
	r := collectResult([]string{"ok"}, time.Now(), "", nil)
	if r.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", r.ExitCode)
	}

	// With error → exit code 1.
	r = collectResult([]string{"fail"}, time.Now(), "", fmt.Errorf("oops"))
	if r.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", r.ExitCode)
	}
}

func TestCollectResult_Summary(t *testing.T) {
	// Summary is last non-empty line.
	r := collectResult([]string{"first", "middle", "last", ""}, time.Now(), "", nil)
	if r.Summary != "last" {
		t.Errorf("summary = %q, want 'last'", r.Summary)
	}

	// Empty lines → empty summary.
	r = collectResult([]string{""}, time.Now(), "", nil)
	if r.Summary != "" {
		t.Errorf("summary = %q, want empty", r.Summary)
	}
}

func TestCollectResult_LogPath(t *testing.T) {
	r := collectResult([]string{"ok"}, time.Now(), "/tmp/run.log", nil)
	if r.LogPath != "/tmp/run.log" {
		t.Errorf("log path = %q, want /tmp/run.log", r.LogPath)
	}
}

// --- prepareExecution tests ---

func TestPrepareExecution_MergesArgs(t *testing.T) {
	req := ToolCallRequest{
		Runbook: domain.Runbook{
			Name:      "test-rb",
			Script:    "run.sh",
			RiskLevel: domain.RiskLow,
			Parameters: []domain.Parameter{
				{Name: "region", Scope: "global"},
			},
		},
		Catalog: domain.Catalog{
			Name: "default",
			Path: t.TempDir(),
		},
		Config: &domain.Config{
			Vars: domain.Vars{
				Global: map[string]any{"region": "us-east-1"},
			},
		},
		Args: map[string]any{
			"region":      "eu-west-1",
			"_confirm_id": "test-rb", // should be skipped
		},
	}

	_, env, err := prepareExecution(req)
	if err != nil {
		t.Fatalf("prepareExecution: %v", err)
	}
	// Args should override resolved vars.
	if env["REGION"] != "eu-west-1" {
		t.Errorf("REGION = %q, want eu-west-1", env["REGION"])
	}
	// Confirm fields should be excluded.
	if _, ok := env["_CONFIRM_ID"]; ok {
		t.Error("_confirm_id should be excluded from env")
	}
}

func TestPrepareExecution_FailsOnRiskValidation(t *testing.T) {
	req := ToolCallRequest{
		Runbook: domain.Runbook{
			Name:      "test-rb",
			Script:    "run.sh",
			RiskLevel: domain.RiskHigh,
			ID:        "default.test-rb",
		},
		Catalog: domain.Catalog{Name: "default", Path: "/tmp"},
		Config:  &domain.Config{},
		Args:    map[string]any{}, // missing _confirm_id
	}

	_, _, err := prepareExecution(req)
	if err == nil {
		t.Error("should fail when high-risk confirmation is missing")
	}
}

// --- RunbookToDescription additional tests ---

func TestRunbookToDescription_WithAliases(t *testing.T) {
	rb := domain.Runbook{
		ID:          "test.deploy",
		Description: "Deploy app",
		Aliases:     []string{"d", "dep"},
	}

	desc := RunbookToDescription(rb)
	if desc != "Deploy app [aliases: d, dep]" {
		t.Errorf("description = %q, unexpected", desc)
	}
}

func TestRunbookToDescription_NoExtras(t *testing.T) {
	rb := domain.Runbook{
		ID:          "test.simple",
		Description: "Simple task",
	}

	desc := RunbookToDescription(rb)
	if desc != "Simple task" {
		t.Errorf("description = %q, want 'Simple task'", desc)
	}
}

func TestRunbookToDescription_RiskOnly(t *testing.T) {
	rb := domain.Runbook{
		ID:          "test.risky",
		Description: "Risky op",
		RiskLevel:   domain.RiskHigh,
	}

	desc := RunbookToDescription(rb)
	expected := "Risky op [risk: high]"
	if desc != expected {
		t.Errorf("description = %q, want %q", desc, expected)
	}
}
