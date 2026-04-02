package sidebar

import "dops/internal/domain"

// RunbookSelectedMsg is emitted when the cursor moves to a runbook.
type RunbookSelectedMsg struct {
	Runbook domain.Runbook
	Catalog domain.Catalog
}

// RunbookExecuteMsg is emitted when Enter is pressed on a runbook.
type RunbookExecuteMsg struct {
	Runbook domain.Runbook
	Catalog domain.Catalog
}

// CatalogSwitchedMsg is emitted when the active catalog tab changes.
type CatalogSwitchedMsg struct {
	CatalogName string // empty string means "All"
}
