package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dops/internal/adapters"

	"dops/internal/domain"
	"dops/internal/executor"
	"dops/internal/vars"
)

const maxToolOutputLines = 50

// ToolResult is the structured result returned from a tool call.
type ToolResult struct {
	ExitCode    int    `json:"exit_code"`
	Duration    string `json:"duration"`
	LogPath     string `json:"log_path,omitempty"`
	OutputLines int    `json:"output_lines"`
	Output      string `json:"output"`
	Summary     string `json:"summary,omitempty"`
}

// ToolCallRequest groups the inputs for HandleToolCall.
type ToolCallRequest struct {
	Runbook    domain.Runbook
	Catalog    domain.Catalog
	Config     *domain.Config
	Runner     executor.Runner
	Args       map[string]any
	OnProgress ProgressCallback
}

// HandleToolCall executes a runbook and returns a truncated result.
// The optional OnProgress callback receives batched output during execution.
func HandleToolCall(ctx context.Context, req ToolCallRequest) (*ToolResult, error) {
	scriptPath, env, err := prepareExecution(req)
	if err != nil {
		return nil, err
	}

	// Create log file.
	logWriter := adapters.NewLogWriter(os.TempDir())
	logPath := ""
	if lp, err := logWriter.Create(req.Catalog.Name, req.Runbook.Name, time.Now()); err == nil {
		logPath = lp
	}

	// Execute with progress streaming.
	start := time.Now()
	lines, errs := req.Runner.Run(ctx, scriptPath, env)

	pw := NewProgressWriter(defaultProgressBatchSize, req.OnProgress)
	var allLines []string
	for line := range lines {
		allLines = append(allLines, line.Text)
		_, _ = pw.Write([]byte(line.Text + "\n")) // error not actionable in streaming loop
		logWriter.WriteLine(line.Text)
	}
	pw.Flush()
	logWriter.Close()

	return collectResult(allLines, start, logPath, <-errs), nil
}

// prepareExecution validates the request, resolves variables, and builds
// the script path and environment map needed for execution.
func prepareExecution(req ToolCallRequest) (scriptPath string, env map[string]string, err error) {
	if err = validateRiskConfirmation(req.Runbook, req.Args); err != nil {
		return "", nil, err
	}

	// Resolve saved vars and merge with provided args.
	resolver := vars.NewDefaultResolver()
	resolved := resolver.Resolve(req.Config, req.Catalog.Name, req.Runbook.Name, req.Runbook.Parameters)
	for k, v := range req.Args {
		if strings.HasPrefix(k, "_confirm") {
			continue // skip synthetic confirmation fields
		}
		resolved[k] = fmt.Sprintf("%v", v)
	}

	// Build env and script path.
	catPath := adapters.ExpandHome(req.Catalog.RunbookRoot())
	scriptPath = filepath.Join(catPath, req.Runbook.Name, req.Runbook.Script)

	env = make(map[string]string)
	for k, v := range resolved {
		env[strings.ToUpper(k)] = v
	}

	return scriptPath, env, nil
}

// collectResult assembles a ToolResult from the raw execution output,
// determining exit code, truncating output, and extracting a summary.
func collectResult(allLines []string, start time.Time, logPath string, err error) *ToolResult {
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		exitCode = 1
	}

	// Truncate output to last N lines.
	output := allLines
	if len(output) > maxToolOutputLines {
		output = output[len(output)-maxToolOutputLines:]
	}

	// Summary: last non-empty line.
	summary := ""
	for i := len(allLines) - 1; i >= 0; i-- {
		if strings.TrimSpace(allLines[i]) != "" {
			summary = strings.TrimSpace(allLines[i])
			break
		}
	}

	return &ToolResult{
		ExitCode:    exitCode,
		Duration:    duration.Round(time.Millisecond).String(),
		LogPath:     logPath,
		OutputLines: len(allLines),
		Output:      strings.Join(output, "\n"),
		Summary:     summary,
	}
}

func validateRiskConfirmation(rb domain.Runbook, args map[string]any) error {
	switch rb.RiskLevel {
	case domain.RiskHigh:
		confirmID, _ := args["_confirm_id"].(string)
		if confirmID != rb.ID {
			return fmt.Errorf("high risk: _confirm_id must be %q, got %q", rb.ID, confirmID)
		}
	case domain.RiskCritical:
		confirmWord, _ := args["_confirm_word"].(string)
		if confirmWord != "CONFIRM" {
			return fmt.Errorf("critical risk: _confirm_word must be \"CONFIRM\", got %q", confirmWord)
		}
	}
	return nil
}
