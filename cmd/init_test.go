package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"dops/internal/domain"
)

func TestInit_FreshInstall(t *testing.T) {
	dir := t.TempDir()
	dopsDir := filepath.Join(dir, ".dops")

	out, err := executeCmd([]string{"init"}, dopsDir)
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	// Config should exist.
	configPath := filepath.Join(dopsDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var cfg domain.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}

	// Should have one catalog registered.
	if len(cfg.Catalogs) != 1 {
		t.Fatalf("catalogs = %d, want 1", len(cfg.Catalogs))
	}
	if cfg.Catalogs[0].Name != "default" {
		t.Errorf("catalog name = %q, want default", cfg.Catalogs[0].Name)
	}

	// Hello-world runbook should exist.
	rbPath := filepath.Join(dopsDir, "catalogs", "default", "hello-world", "runbook.yaml")
	if _, err := os.Stat(rbPath); err != nil {
		t.Errorf("runbook.yaml missing: %v", err)
	}

	scriptPath := filepath.Join(dopsDir, "catalogs", "default", "hello-world", "script.sh")
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Errorf("script.sh missing: %v", err)
	} else if info.Mode()&0o111 == 0 {
		t.Error("script.sh is not executable")
	}
}

func TestInit_ExistingCatalogs(t *testing.T) {
	dir := t.TempDir()
	dopsDir := filepath.Join(dir, ".dops")
	os.MkdirAll(dopsDir, 0o750)

	// Create config with an existing catalog.
	cfg := &domain.Config{
		Theme:    "tokyomidnight",
		Defaults: domain.Defaults{MaxRiskLevel: domain.RiskMedium},
		Catalogs: []domain.Catalog{
			{Name: "mycat", Path: "/some/path", Active: true},
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(dopsDir, "config.json"), data, 0o644)

	out, err := executeCmd([]string{"init"}, dopsDir)
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	// Should NOT have created hello-world.
	rbDir := filepath.Join(dopsDir, "catalogs", "default", "hello-world")
	if _, err := os.Stat(rbDir); err == nil {
		t.Error("hello-world should not be created when catalogs already exist")
	}

	// Config should still have only the original catalog.
	data, _ = os.ReadFile(filepath.Join(dopsDir, "config.json"))
	var loaded domain.Config
	json.Unmarshal(data, &loaded)
	if len(loaded.Catalogs) != 1 {
		t.Errorf("catalogs = %d, want 1", len(loaded.Catalogs))
	}
	if loaded.Catalogs[0].Name != "mycat" {
		t.Errorf("catalog name = %q, want mycat", loaded.Catalogs[0].Name)
	}
}

func TestInit_Idempotent(t *testing.T) {
	dir := t.TempDir()
	dopsDir := filepath.Join(dir, ".dops")

	// Run init twice.
	if _, err := executeCmd([]string{"init"}, dopsDir); err != nil {
		t.Fatalf("first init: %v", err)
	}
	if _, err := executeCmd([]string{"init"}, dopsDir); err != nil {
		t.Fatalf("second init: %v", err)
	}

	// Should still have exactly one catalog (not duplicated).
	data, _ := os.ReadFile(filepath.Join(dopsDir, "config.json"))
	var cfg domain.Config
	json.Unmarshal(data, &cfg)
	if len(cfg.Catalogs) != 1 {
		t.Errorf("catalogs = %d after double init, want 1", len(cfg.Catalogs))
	}
}
