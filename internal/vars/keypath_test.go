package vars

import "testing"

func TestVarKeyPath(t *testing.T) {
	tests := []struct {
		name        string
		scope       string
		paramName   string
		catalogName string
		runbookName string
		want        string
	}{
		{
			name:      "global scope",
			scope:     "global",
			paramName: "region",
			want:      "vars.global.region",
		},
		{
			name:        "catalog scope",
			scope:       "catalog",
			paramName:   "env",
			catalogName: "deploy",
			want:        "vars.catalog.deploy.env",
		},
		{
			name:        "runbook scope",
			scope:       "runbook",
			paramName:   "tag",
			catalogName: "deploy",
			runbookName: "rollout",
			want:        "vars.catalog.deploy.runbooks.rollout.tag",
		},
		{
			name:      "empty scope defaults to global",
			scope:     "",
			paramName: "foo",
			want:      "vars.global.foo",
		},
		{
			name:      "unknown scope defaults to global",
			scope:     "something",
			paramName: "bar",
			want:      "vars.global.bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VarKeyPath(tt.scope, tt.paramName, tt.catalogName, tt.runbookName)
			if got != tt.want {
				t.Errorf("VarKeyPath(%q, %q, %q, %q) = %q, want %q",
					tt.scope, tt.paramName, tt.catalogName, tt.runbookName, got, tt.want)
			}
		})
	}
}
