package executor

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"path/filepath"
	"time"
)

// DemoRunner simulates script execution with realistic output.
// Used for public demos where real execution is disabled.
type DemoRunner struct{}

func NewDemoRunner() *DemoRunner { return &DemoRunner{} }

func (d *DemoRunner) Run(ctx context.Context, scriptPath string, env map[string]string) (<-chan OutputLine, <-chan error) {
	lines := make(chan OutputLine, 100)
	errs := make(chan error, 1)

	// Derive runbook name from script path for context-aware output.
	name := filepath.Base(filepath.Dir(scriptPath))

	go func() {
		defer close(lines)
		defer close(errs)

		for _, line := range demoOutput(name, env) {
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			default:
			}

			// Simulate realistic output timing.
			jitter, err := rand.Int(rand.Reader, big.NewInt(120))
			if err != nil {
				jitter = big.NewInt(60) // fallback to midpoint
			}
			delay := time.Duration(450+jitter.Int64()) * time.Millisecond
			time.Sleep(delay)

			lines <- OutputLine{Text: line}
		}

		errs <- nil
	}()

	return lines, errs
}

func demoOutput(name string, env map[string]string) []string {
	// Check for specific runbook simulations.
	if out, ok := runbookOutputs[name]; ok {
		return resolveEnv(out, env)
	}

	// Generic fallback.
	return []string{
		fmt.Sprintf("$ %s", name),
		"Initializing...",
		"Running pre-flight checks... OK",
		"Executing task...",
		"",
		"Step 1/3: Preparing environment",
		"  → Loading configuration",
		"  → Validating parameters",
		"Step 2/3: Running operation",
		"  → Processing...",
		"  → Done",
		"Step 3/3: Cleanup",
		"  → Removing temporary files",
		"",
		fmt.Sprintf("\033[32m✓\033[0m %s completed successfully", name),
	}
}

