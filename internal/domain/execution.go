package domain

import (
	"crypto/rand"
	"fmt"
	"time"
)

// ExecStatus represents the state of an execution.
type ExecStatus string

const (
	ExecRunning   ExecStatus = "running"
	ExecSuccess   ExecStatus = "success"
	ExecFailed    ExecStatus = "failed"
	ExecCancelled ExecStatus = "cancelled"
)

// ExecInterface identifies which interface initiated the execution.
type ExecInterface string

const (
	ExecTUI ExecInterface = "tui"
	ExecCLI ExecInterface = "cli"
	ExecWeb ExecInterface = "web"
	ExecMCP ExecInterface = "mcp"
)

// ExecutionRecord is a persistent record of a single runbook execution.
type ExecutionRecord struct {
	ID            string            `json:"id"`
	RunbookID     string            `json:"runbook_id"`
	RunbookName   string            `json:"runbook_name"`
	CatalogName   string            `json:"catalog_name"`
	Parameters    map[string]string `json:"parameters,omitempty"`
	Status        ExecStatus        `json:"status"`
	ExitCode      int               `json:"exit_code"`
	StartTime     time.Time         `json:"start_time"`
	EndTime       time.Time         `json:"end_time,omitempty"`
	Duration      string            `json:"duration,omitempty"`
	OutputLines   int               `json:"output_lines"`
	OutputSummary string            `json:"output_summary,omitempty"`
	LogPath       string            `json:"log_path,omitempty"`
	Interface     ExecInterface     `json:"interface"`
}

// NewExecutionRecord creates a new record in running state.
func NewExecutionRecord(runbookID, runbookName, catalogName string, iface ExecInterface) *ExecutionRecord {
	return &ExecutionRecord{
		ID:          generateExecID(),
		RunbookID:   runbookID,
		RunbookName: runbookName,
		CatalogName: catalogName,
		Status:      ExecRunning,
		StartTime:   time.Now(),
		Interface:   iface,
	}
}

// Complete marks the execution as finished.
func (r *ExecutionRecord) Complete(exitCode int, outputLines int, summary string) {
	r.EndTime = time.Now()
	r.Duration = r.EndTime.Sub(r.StartTime).Round(time.Millisecond).String()
	r.ExitCode = exitCode
	r.OutputLines = outputLines
	r.OutputSummary = summary
	if exitCode == 0 {
		r.Status = ExecSuccess
	} else {
		r.Status = ExecFailed
	}
}

// Cancel marks the execution as cancelled.
func (r *ExecutionRecord) Cancel() {
	r.EndTime = time.Now()
	r.Duration = r.EndTime.Sub(r.StartTime).Round(time.Millisecond).String()
	r.Status = ExecCancelled
	r.ExitCode = -1
}

// MaskSecrets replaces secret parameter values with "****".
func (r *ExecutionRecord) MaskSecrets(secretNames []string) {
	for _, name := range secretNames {
		if _, ok := r.Parameters[name]; ok {
			r.Parameters[name] = "****"
		}
	}
}

func generateExecID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	// UUID v4 format.
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 1
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
