package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"dops/internal/adapters"

	"dops/internal/domain"
	"dops/internal/executor"
	"dops/internal/vars"
)

const maxOutputLines = 50

// ToolResult is the structured result returned from a tool call.
type ToolResult struct {
	ExitCode    int    `json:"exit_code"`
	Duration    string `json:"duration"`
	LogPath     string `json:"log_path,omitempty"`
	OutputLines int    `json:"output_lines"`
	Output      string `json:"output"`
	Summary     string `json:"summary,omitempty"`
}

// HandleToolCall executes a runbook and returns a truncated result.
// The optional onProgress callback receives batched output during execution.
func HandleToolCall(
	ctx context.Context,
	rb domain.Runbook,
	cat domain.Catalog,
	cfg *domain.Config,
	runner executor.Runner,
	args map[string]any,
	onProgress ProgressCallback,
) (*ToolResult, error) {
	// Validate risk confirmation.
	if err := validateRiskConfirmation(rb, args); err != nil {
		return nil, err
	}

	// Resolve saved vars and merge with provided args.
	resolver := vars.NewDefaultResolver()
	resolved := resolver.Resolve(cfg, cat.Name, rb.Name, rb.Parameters)
	for k, v := range args {
		if strings.HasPrefix(k, "_confirm") {
			continue // skip synthetic confirmation fields
		}
		resolved[k] = fmt.Sprintf("%v", v)
	}

	// Build env and script path.
	catPath := expandTilde(cat.RunbookRoot())
	scriptPath := filepath.Join(catPath, rb.Name, rb.Script)

	env := make(map[string]string)
	for k, v := range resolved {
		env[strings.ToUpper(k)] = v
	}

	// Create log file.
	lw := adapters.NewLogWriter("/tmp")
	logPath := ""
	if lp, err := lw.Create(cat.Name, rb.Name, time.Now()); err == nil {
		logPath = lp
	}

	// Execute with progress streaming.
	start := time.Now()
	lines, errs := runner.Run(ctx, scriptPath, env)

	pw := NewProgressWriter(defaultBatchSize, onProgress)
	var allLines []string
	for line := range lines {
		allLines = append(allLines, line.Text)
		_, _ = pw.Write([]byte(line.Text + "\n")) // error not actionable in streaming loop
		lw.WriteLine(line.Text)
	}
	pw.Flush()
	lw.Close()

	err := <-errs
	duration := time.Since(start)

	// Determine exit code.
	exitCode := 0
	if err != nil {
		exitCode = 1
	}

	// Truncate output to last N lines.
	output := allLines
	if len(output) > maxOutputLines {
		output = output[len(output)-maxOutputLines:]
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
	}, nil
}

func expandTilde(path string) string {
	return adapters.ExpandHome(path)
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
