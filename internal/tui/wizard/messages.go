package wizard

import "dops/internal/domain"

type SubmitMsg struct {
	Runbook domain.Runbook
	Catalog domain.Catalog
	Params  map[string]string
}

type CancelMsg struct{}
