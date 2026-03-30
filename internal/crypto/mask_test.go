package crypto

import (
	"dops/internal/domain"
	"testing"
)

func TestMaskSecrets(t *testing.T) {
	vars := domain.Vars{
		Global: map[string]any{
			"region": "us-east-1",
			"token":  "age1qyqszqgpqyqszqgp",
		},
		Catalog: map[string]domain.CatalogVars{
			"default": {
				Vars: map[string]any{
					"namespace":  "platform",
					"secret_key": "age1abcdefgh",
				},
				Runbooks: map[string]map[string]any{
					"hello": {
						"flag":   true,
						"apikey": "age1zzzzz",
					},
				},
			},
		},
	}

	masked := MaskSecrets(vars)

	// Original should not be modified
	if vars.Global["token"] != "age1qyqszqgpqyqszqgp" {
		t.Error("original vars were modified")
	}

	// Plain values should pass through
	if masked.Global["region"] != "us-east-1" {
		t.Errorf("region = %v, want us-east-1", masked.Global["region"])
	}
	if masked.Catalog["default"].Vars["namespace"] != "platform" {
		t.Errorf("namespace = %v, want platform", masked.Catalog["default"].Vars["namespace"])
	}

	// Encrypted values should be masked
	if masked.Global["token"] != "****" {
		t.Errorf("token = %v, want ****", masked.Global["token"])
	}
	if masked.Catalog["default"].Vars["secret_key"] != "****" {
		t.Errorf("secret_key = %v, want ****", masked.Catalog["default"].Vars["secret_key"])
	}
	if masked.Catalog["default"].Runbooks["hello"]["apikey"] != "****" {
		t.Errorf("apikey = %v, want ****", masked.Catalog["default"].Runbooks["hello"]["apikey"])
	}

	// Non-string values should pass through
	if masked.Catalog["default"].Runbooks["hello"]["flag"] != true {
		t.Errorf("flag = %v, want true", masked.Catalog["default"].Runbooks["hello"]["flag"])
	}
}
