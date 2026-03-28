package domain

import (
	"strings"
	"testing"
)

func TestValidateAlias(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		wantErr bool
		errMsg  string
	}{
		{name: "valid simple", alias: "deploy"},
		{name: "valid with hyphen", alias: "deploy-app"},
		{name: "valid with dot", alias: "infra.deploy"},
		{name: "valid with numbers", alias: "v1.2.3"},
		{name: "empty", alias: "", wantErr: true, errMsg: "must not be empty"},
		{name: "uppercase", alias: "Deploy", wantErr: true, errMsg: "lowercase"},
		{name: "spaces", alias: "deploy app", wantErr: true, errMsg: "lowercase"},
		{name: "starts with hyphen", alias: "-deploy", wantErr: true, errMsg: "lowercase"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAlias(tt.alias)
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

func TestValidateRunbookID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "valid id", id: "default.hello-world", wantErr: false},
		{name: "valid with numbers", id: "local.rotate-tls-123", wantErr: false},
		{name: "missing dot", id: "hello-world", wantErr: true},
		{name: "empty string", id: "", wantErr: true},
		{name: "empty catalog segment", id: ".hello-world", wantErr: true},
		{name: "empty runbook segment", id: "default.", wantErr: true},
		{name: "multiple dots", id: "a.b.c", wantErr: true},
		{name: "dot only", id: ".", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRunbookID(tt.id)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateRunbookID(%q) expected error, got nil", tt.id)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateRunbookID(%q) unexpected error: %v", tt.id, err)
			}
		})
	}
}
