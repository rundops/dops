package mcp

import (
	"encoding/json"

	"dops/internal/catalog"
	"dops/internal/domain"
)

// RunbookSummary is the JSON representation of a runbook in the catalog listing.
type RunbookSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Catalog     string `json:"catalog"`
	Description string `json:"description"`
	RiskLevel   string `json:"risk_level"`
	Version     string `json:"version"`
}

// CatalogListJSON returns a JSON representation of all runbooks.
func CatalogListJSON(catalogs []catalog.CatalogWithRunbooks) (string, error) {
	var summaries []RunbookSummary
	for _, c := range catalogs {
		for _, rb := range c.Runbooks {
			summaries = append(summaries, RunbookSummary{
				ID:          rb.ID,
				Name:        rb.Name,
				Catalog:     c.Catalog.Name,
				Description: rb.Description,
				RiskLevel:   string(rb.RiskLevel),
				Version:     rb.Version,
			})
		}
	}

	data, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RunbookDetailJSON returns a JSON representation of a single runbook.
func RunbookDetailJSON(rb domain.Runbook, cat domain.Catalog) (string, error) {
	detail := map[string]any{
		"id":          rb.ID,
		"name":        rb.Name,
		"catalog":     cat.Name,
		"description": rb.Description,
		"risk_level":  string(rb.RiskLevel),
		"version":     rb.Version,
		"script":      rb.Script,
		"parameters":  rb.Parameters,
	}

	data, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
