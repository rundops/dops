package domain

import (
	"fmt"
	"regexp"
	"strings"
)

type ParameterType string

const (
	ParamString      ParameterType = "string"
	ParamBoolean     ParameterType = "boolean"
	ParamInteger     ParameterType = "integer" // any whole number (negative ok)
	ParamNumber      ParameterType = "number"  // non-negative whole number (0+)
	ParamFloat       ParameterType = "float"   // decimal number
	ParamSelect      ParameterType = "select"
	ParamMultiSelect ParameterType = "multi_select"
	ParamFilePath    ParameterType = "file_path"
	ParamResourceID  ParameterType = "resource_id"
)

type Parameter struct {
	Name        string        `yaml:"name" json:"name"`
	Type        ParameterType `yaml:"type" json:"type"`
	Required    bool          `yaml:"required" json:"required"`
	Scope       string        `yaml:"scope" json:"scope"`
	Secret      bool          `yaml:"secret" json:"secret"`
	Default     any           `yaml:"default,omitempty" json:"default,omitempty"`
	Description string        `yaml:"description" json:"description"`
	Options     []string      `yaml:"options,omitempty" json:"options,omitempty"`
}

type Runbook struct {
	ID          string      `yaml:"id" json:"id"`
	Name        string      `yaml:"name" json:"name"`
	Aliases     []string    `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	Description string      `yaml:"description" json:"description"`
	Version     string      `yaml:"version" json:"version"`
	RiskLevel   RiskLevel   `yaml:"risk_level" json:"risk_level"`
	Script      string      `yaml:"script" json:"script"`
	Parameters  []Parameter `yaml:"parameters" json:"parameters"`
}

// validAlias matches lowercase alphanumeric, hyphens, and dots.
var validAlias = regexp.MustCompile(`^[a-z0-9][a-z0-9.\-]*$`)

// ValidateAlias checks that an alias follows naming rules.
func ValidateAlias(alias string) error {
	if alias == "" {
		return fmt.Errorf("alias must not be empty")
	}
	if !validAlias.MatchString(alias) {
		return fmt.Errorf("alias %q must be lowercase alphanumeric with hyphens and dots only", alias)
	}
	return nil
}

func ValidateRunbookID(id string) error {
	if id == "" {
		return fmt.Errorf("runbook id must not be empty")
	}
	parts := strings.Split(id, ".")
	if len(parts) != 2 {
		return fmt.Errorf("runbook id must be in <catalog>.<runbook> format, got %q", id)
	}
	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("runbook id must have non-empty catalog and runbook segments, got %q", id)
	}
	return nil
}
