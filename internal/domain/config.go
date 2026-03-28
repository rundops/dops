package domain

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"unicode"
)

type Config struct {
	Theme    string    `json:"theme"`
	Defaults Defaults  `json:"defaults"`
	Catalogs []Catalog `json:"catalogs"`
	Vars     Vars      `json:"-"`
}

type Defaults struct {
	MaxRiskLevel RiskLevel `json:"max_risk_level"`
}

type Catalog struct {
	Name        string        `json:"name"`
	DisplayName string        `json:"display_name,omitempty"`
	Path        string        `json:"path"`
	SubPath     string        `json:"sub_path,omitempty"`
	URL         string        `json:"url,omitempty"`
	Active      bool          `json:"active"`
	Policy      CatalogPolicy `json:"policy"`
}

// Label returns the display name if set, otherwise the canonical name.
func (c Catalog) Label() string {
	if c.DisplayName != "" {
		return c.DisplayName
	}
	return c.Name
}

// ValidateDisplayName checks that a display name is within length limits
// and contains only printable characters.
func ValidateDisplayName(name string) error {
	if len(name) > 50 {
		return fmt.Errorf("display name must be 50 characters or fewer (got %d)", len(name))
	}
	for _, r := range name {
		if !unicode.IsPrint(r) {
			return fmt.Errorf("display name contains non-printable character: %U", r)
		}
	}
	return nil
}

// RunbookRoot returns the effective directory the loader should read runbooks
// from. When SubPath is set it is joined to Path; otherwise Path is returned.
func (c Catalog) RunbookRoot() string {
	if c.SubPath != "" {
		return filepath.Join(c.Path, c.SubPath)
	}
	return c.Path
}

type CatalogPolicy struct {
	MaxRiskLevel RiskLevel `json:"max_risk_level"`
}

type Vars struct {
	Global  map[string]any         `json:"global"`
	Catalog map[string]CatalogVars `json:"catalog"`
}

type CatalogVars struct {
	Vars     map[string]any            // catalog-scoped vars (flat keys)
	Runbooks map[string]map[string]any // runbook-scoped vars
}

func (cv CatalogVars) MarshalJSON() ([]byte, error) {
	m := make(map[string]any, len(cv.Vars)+1)
	for k, v := range cv.Vars {
		m[k] = v
	}
	if len(cv.Runbooks) > 0 {
		m["runbooks"] = cv.Runbooks
	}
	return json.Marshal(m)
}

func (cv *CatalogVars) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	cv.Vars = make(map[string]any)
	cv.Runbooks = make(map[string]map[string]any)

	for k, v := range raw {
		if k == "runbooks" {
			if err := json.Unmarshal(v, &cv.Runbooks); err != nil {
				return err
			}
			continue
		}
		var val any
		if err := json.Unmarshal(v, &val); err != nil {
			return err
		}
		cv.Vars[k] = val
	}

	return nil
}
