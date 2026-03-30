package confirm

import "dops/internal/domain"

// AcceptMsg is sent when the user confirms execution.
type AcceptMsg struct {
	Runbook domain.Runbook
	Catalog domain.Catalog
	Params  map[string]string
}

// CancelMsg is sent when the user cancels confirmation.
type CancelMsg struct{}
