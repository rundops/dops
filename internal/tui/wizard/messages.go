package wizard

import "dops/internal/domain"

type SubmitMsg struct {
	Runbook domain.Runbook
	Catalog domain.Catalog
	Params  map[string]string
}

type CancelMsg struct{}

// SaveFieldMsg is emitted when the user confirms saving a parameter value.
// The parent (App) handles the actual persistence via config.Set + vault.Save.
type SaveFieldMsg struct {
	Scope       string
	ParamName   string
	CatalogName string
	RunbookName string
	Value       string
}

// SaveFieldResultMsg is sent back to the wizard after a save attempt.
type SaveFieldResultMsg struct {
	Err error
}
