package domain

import (
	"encoding/json"
	"path/filepath"
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
	Name    string        `json:"name"`
	Path    string        `json:"path"`
	SubPath string        `json:"sub_path,omitempty"`
	URL     string        `json:"url,omitempty"`
	Active  bool          `json:"active"`
	Policy  CatalogPolicy `json:"policy"`
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
