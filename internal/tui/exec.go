package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"dops/internal/adapters"
	"dops/internal/domain"
	"dops/internal/tui/output"
	"dops/internal/tui/wizard"

	tea "charm.land/bubbletea/v2"
)

// executionDoneMsg signals that a script execution has finished.
type executionDoneMsg struct {
	LogPath string
	Err     error
}

func (m App) startExecution(rb domain.Runbook, cat domain.Catalog, params map[string]string) (tea.Model, tea.Cmd) {
	m.output.Clear()
	resolved := m.resolveVars()
	cmdStr := wizard.BuildCommand(rb, params, resolved)
	m.output.SetCommand(cmdStr)
	m.resizeAll()

	if m.deps.Runner == nil {
		return m, nil
	}

	scriptPath := filepath.Join(adapters.ExpandHome(cat.RunbookRoot()), rb.Name, rb.Script)

	if m.deps.DryRun {
		m.emitDryRun(scriptPath, params)
		return m, nil
	}

	env := buildEnv(params)
	logPath := m.createLogFile(cat.Name, rb.Name)

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelExec = cancel
	m.execRunning = true

	// Stream lines live via p.Send() if we have a program reference,
	// otherwise fall back to a tea.Cmd (tests).
	if prog := m.program(); prog != nil {
		go m.runStreaming(ctx, prog, scriptPath, env, logPath)
		return m, nil
	}

	return m, m.runBlocking(ctx, scriptPath, env, logPath)
}

// emitDryRun writes dry-run output to the output pane without executing.
func (m *App) emitDryRun(scriptPath string, params map[string]string) {
	m.output, _ = m.output.Update(output.OutputLineMsg{Text: "[DRY RUN] Would execute:"})
	m.output, _ = m.output.Update(output.OutputLineMsg{Text: fmt.Sprintf("  Script: %s", scriptPath)})
	m.output, _ = m.output.Update(output.OutputLineMsg{Text: ""})
	if len(params) > 0 {
		m.output, _ = m.output.Update(output.OutputLineMsg{Text: "  Environment:"})
		for k, v := range params {
			m.output, _ = m.output.Update(output.OutputLineMsg{
				Text: fmt.Sprintf("    %s=%s", strings.ToUpper(k), v),
			})
		}
	}
	m.output, _ = m.output.Update(output.ExecutionDoneMsg{})
}

// runStreaming executes a script in a goroutine, streaming lines to the
// tea.Program via Send(). Used for live output in the TUI.
func (m App) runStreaming(ctx context.Context, prog *tea.Program, scriptPath string, env map[string]string, logPath string) {
	lw := m.deps.LogWriter
	lines, errs := m.deps.Runner.Run(ctx, scriptPath, env)
	for line := range lines {
		if lw != nil {
			lw.WriteLine(line.Text)
		}
		prog.Send(output.OutputLineMsg{Text: line.Text, IsStderr: line.IsStderr})
	}
	if lw != nil {
		lw.Close()
	}
	prog.Send(executionDoneMsg{LogPath: logPath, Err: <-errs})
}

// runBlocking returns a tea.Cmd that executes a script synchronously.
// Used in tests where no tea.Program reference is available.
func (m App) runBlocking(ctx context.Context, scriptPath string, env map[string]string, logPath string) tea.Cmd {
	runner := m.deps.Runner
	lw := m.deps.LogWriter
	return func() tea.Msg {
		lines, errs := runner.Run(ctx, scriptPath, env)
		for line := range lines {
			if lw != nil {
				lw.WriteLine(line.Text)
			}
		}
		if lw != nil {
			lw.Close()
		}
		return executionDoneMsg{LogPath: logPath, Err: <-errs}
	}
}

// program returns the tea.Program reference, or nil if not set.
func (m App) program() *tea.Program {
	if m.deps.ProgramRef != nil {
		return m.deps.ProgramRef.P
	}
	return nil
}

// createLogFile creates a log file for the execution and returns its path.
func (m App) createLogFile(catalogName, runbookName string) string {
	if m.deps.LogWriter == nil {
		return ""
	}
	logPath, err := m.deps.LogWriter.Create(catalogName, runbookName, time.Now())
	if err != nil {
		return ""
	}
	return logPath
}

// buildEnv converts parameter keys to uppercase environment variable names.
func buildEnv(params map[string]string) map[string]string {
	env := make(map[string]string, len(params))
	for k, v := range params {
		env[strings.ToUpper(k)] = v
	}
	return env
}
