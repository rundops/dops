package domain

import (
	"strings"
	"testing"
)

func TestCatalog_Label(t *testing.T) {
	tests := []struct {
		name        string
		catalog     Catalog
		wantLabel   string
	}{
		{
			name:      "returns display name when set",
			catalog:   Catalog{Name: "my-repo", DisplayName: "Production Ops"},
			wantLabel: "Production Ops",
		},
		{
			name:      "falls back to name when display name empty",
			catalog:   Catalog{Name: "my-repo"},
			wantLabel: "my-repo",
		},
		{
			name:      "falls back to name when display name blank",
			catalog:   Catalog{Name: "my-repo", DisplayName: ""},
			wantLabel: "my-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.catalog.Label(); got != tt.wantLabel {
				t.Errorf("Label() = %q, want %q", got, tt.wantLabel)
			}
		})
	}
}

func TestCatalog_RunbookRoot(t *testing.T) {
	tests := []struct {
		name    string
		catalog Catalog
		want    string
	}{
		{
			name:    "no subpath",
			catalog: Catalog{Path: "/home/user/.dops/catalogs/repo"},
			want:    "/home/user/.dops/catalogs/repo",
		},
		{
			name:    "with subpath",
			catalog: Catalog{Path: "/home/user/.dops/catalogs/repo", SubPath: "src/runbooks"},
			want:    "/home/user/.dops/catalogs/repo/src/runbooks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.catalog.RunbookRoot(); got != tt.want {
				t.Errorf("RunbookRoot() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateDisplayName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:  "valid short name",
			input: "Prod Ops",
		},
		{
			name:  "valid at limit",
			input: strings.Repeat("a", 50),
		},
		{
			name:    "too long",
			input:   strings.Repeat("a", 51),
			wantErr: true,
			errMsg:  "50 characters or fewer",
		},
		{
			name:    "non-printable character",
			input:   "hello\x00world",
			wantErr: true,
			errMsg:  "non-printable",
		},
		{
			name:  "empty is valid",
			input: "",
		},
		{
			name:  "unicode is valid",
			input: "Producción Ops",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDisplayName(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
