package mcp

import (
	"encoding/json"
	"fmt"

	"dops/internal/domain"
)

// RunbookToInputSchema generates a JSON Schema for a runbook's parameters.
func RunbookToInputSchema(rb domain.Runbook, resolved map[string]string) json.RawMessage {
	properties := make(map[string]any)
	var required []string

	for _, p := range rb.Parameters {
		prop := map[string]any{}

		switch p.Type {
		case domain.ParamBoolean:
			prop["type"] = "boolean"
		case domain.ParamInteger:
			prop["type"] = "integer"
		case domain.ParamSelect:
			prop["type"] = "string"
			if len(p.Options) > 0 {
				prop["enum"] = p.Options
			}
		case domain.ParamMultiSelect:
			prop["type"] = "array"
			items := map[string]any{"type": "string"}
			if len(p.Options) > 0 {
				items["enum"] = p.Options
			}
			prop["items"] = items
		case domain.ParamFilePath:
			prop["type"] = "string"
			prop["description"] = "file path"
		case domain.ParamResourceID:
			prop["type"] = "string"
			prop["description"] = "resource identifier"
		default: // string
			prop["type"] = "string"
		}

		if p.Description != "" {
			desc := p.Description
			if _, saved := resolved[p.Name]; saved {
				desc += " (pre-configured, optional override)"
			}
			prop["description"] = desc
		}

		if p.Default != nil {
			prop["default"] = p.Default
		}

		properties[p.Name] = prop

		// Saved params are optional (can be overridden).
		if p.Required {
			if _, saved := resolved[p.Name]; !saved {
				required = append(required, p.Name)
			}
		}
	}

	// Add synthetic confirmation fields for risk gates.
	switch rb.RiskLevel {
	case domain.RiskHigh:
		properties["_confirm_id"] = map[string]any{
			"type":        "string",
			"description": fmt.Sprintf("Must be %q to confirm execution (high risk)", rb.ID),
		}
		required = append(required, "_confirm_id")
	case domain.RiskCritical:
		properties["_confirm_word"] = map[string]any{
			"type":        "string",
			"description": `Must be "CONFIRM" to confirm execution (critical risk)`,
		}
		required = append(required, "_confirm_word")
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	data, _ := json.Marshal(schema)
	return data
}

// RunbookToDescription generates a tool description for a runbook.
func RunbookToDescription(rb domain.Runbook) string {
	desc := rb.Description
	if rb.RiskLevel != "" {
		desc += fmt.Sprintf(" [risk: %s]", rb.RiskLevel)
	}
	return desc
}
