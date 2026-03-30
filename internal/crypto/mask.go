package crypto

import "dops/internal/domain"

const maskedValue = "****"

func MaskSecrets(vars domain.Vars) domain.Vars {
	out := vars

	out.Global = maskMap(vars.Global)

	out.Catalog = make(map[string]domain.CatalogVars, len(vars.Catalog))
	for catName, cat := range vars.Catalog {
		masked := domain.CatalogVars{
			Vars:     maskMap(cat.Vars),
			Runbooks: make(map[string]map[string]any, len(cat.Runbooks)),
		}
		for rbName, rb := range cat.Runbooks {
			masked.Runbooks[rbName] = maskMap(rb)
		}
		out.Catalog[catName] = masked
	}

	return out
}

func maskMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok && IsEncrypted(s) {
			out[k] = maskedValue
		} else {
			out[k] = v
		}
	}
	return out
}
