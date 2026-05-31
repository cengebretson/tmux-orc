package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cengebretson/orc/internal/config"
)

func TestDefaultWorkflow_FallsBackToDefault(t *testing.T) {
	cfg := &config.Config{}
	if got := cfg.DefaultWorkflow(); got != "default" {
		t.Errorf("DefaultWorkflow() = %q, want \"default\"", got)
	}
}

func TestDefaultWorkflow_UsesConfiguredValue(t *testing.T) {
	cfg := &config.Config{Settings: config.Settings{DefaultWorkflow: "hotfix"}}
	if got := cfg.DefaultWorkflow(); got != "hotfix" {
		t.Errorf("DefaultWorkflow() = %q, want \"hotfix\"", got)
	}
}

func TestLoad_Settings(t *testing.T) {
	dir := t.TempDir()
	content := `
settings:
  default_workflow: hotfix
  auto_archive: true
repos: []
`
	if err := os.WriteFile(filepath.Join(dir, "orc.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Settings.DefaultWorkflow != "hotfix" {
		t.Errorf("default_workflow = %q, want \"hotfix\"", cfg.Settings.DefaultWorkflow)
	}
	if !cfg.Settings.AutoArchive {
		t.Error("auto_archive = false, want true")
	}
	if got := cfg.DefaultWorkflow(); got != "hotfix" {
		t.Errorf("DefaultWorkflow() = %q, want \"hotfix\"", got)
	}
}

func TestLoad_MissingFile_ReturnsEmptyConfig(t *testing.T) {
	cfg, err := config.Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Repos) != 0 {
		t.Errorf("expected empty repos, got %d", len(cfg.Repos))
	}
	if cfg.DefaultWorkflow() != "default" {
		t.Errorf("DefaultWorkflow() = %q, want \"default\"", cfg.DefaultWorkflow())
	}
}
