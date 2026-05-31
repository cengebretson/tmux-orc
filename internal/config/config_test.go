package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cengebretson/orc/internal/config"
)

func writeOrcYAML(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "orc.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultWorkflow_ReturnsEmptyWhenNotSet(t *testing.T) {
	cfg := &config.Config{}
	if got := cfg.DefaultWorkflow(); got != "" {
		t.Errorf("DefaultWorkflow() = %q, want \"\"", got)
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
	writeOrcYAML(t, dir, `
settings:
  default_workflow: hotfix
  auto_archive: true
repos: []
`)

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
	if cfg.DefaultWorkflow() != "" {
		t.Errorf("DefaultWorkflow() = %q, want \"\"", cfg.DefaultWorkflow())
	}
	if len(cfg.Names()) != 0 {
		t.Errorf("expected no workflows, got %d", len(cfg.Names()))
	}
}

func TestLoad_BasicWorkflow(t *testing.T) {
	dir := t.TempDir()
	writeOrcYAML(t, dir, `
workflows:
  default:
    stages:
      - name: intake
        worker: fred-documentor
        advance: auto
      - name: develop
        worker: bob-developer
        advance: manual
`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := cfg.Names()
	if len(names) != 1 || names[0] != "default" {
		t.Errorf("Names() = %v, want [default]", names)
	}

	stages := cfg.StageNames("default")
	if len(stages) != 2 || stages[0] != "intake" || stages[1] != "develop" {
		t.Errorf("StageNames(default) = %v, want [intake develop]", stages)
	}
}

func TestLoad_StageConfig(t *testing.T) {
	dir := t.TempDir()
	writeOrcYAML(t, dir, `
workflows:
  default:
    stages:
      - name: intake
        worker: fred-documentor
        advance: auto
      - name: develop
        worker: bob-developer
        advance: manual
`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sc, ok := cfg.StageConfig("default", "intake")
	if !ok {
		t.Fatal("StageConfig(default, intake) not found")
	}
	if sc.Worker != "fred-documentor" {
		t.Errorf("Worker = %q, want fred-documentor", sc.Worker)
	}
	if sc.Advance != "auto" {
		t.Errorf("Advance = %q, want auto", sc.Advance)
	}
}

func TestLoad_NextStage(t *testing.T) {
	dir := t.TempDir()
	writeOrcYAML(t, dir, `
workflows:
  default:
    stages:
      - name: intake
        advance: auto
      - name: develop
        advance: manual
      - name: pr-open
        advance: auto
`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if next := cfg.NextStage("default", "intake"); next != "develop" {
		t.Errorf("NextStage(intake) = %q, want develop", next)
	}
	if next := cfg.NextStage("default", "develop"); next != "pr-open" {
		t.Errorf("NextStage(develop) = %q, want pr-open", next)
	}
	if next := cfg.NextStage("default", "pr-open"); next != "" {
		t.Errorf("NextStage(pr-open) = %q, want empty (last stage)", next)
	}
}

func TestLoad_RepairStages(t *testing.T) {
	dir := t.TempDir()
	writeOrcYAML(t, dir, `
workflows:
  default:
    stages:
      - name: pr-open
        advance: auto

repair_stages:
  pr-repair:
    repairs: pr-open
    worker: bob-developer
    advance: auto
    max_retries: 3
`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rd, ok := cfg.RepairStage("pr-repair")
	if !ok {
		t.Fatal("RepairStage(pr-repair) not found")
	}
	if rd.Repairs != "pr-open" {
		t.Errorf("Repairs = %q, want pr-open", rd.Repairs)
	}
	if rd.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", rd.MaxRetries)
	}
	if !cfg.IsRepairStage("pr-repair") {
		t.Error("IsRepairStage(pr-repair) = false, want true")
	}
	if cfg.IsRepairStage("pr-open") {
		t.Error("IsRepairStage(pr-open) = true, want false")
	}
}

func TestLoad_MultipleWorkflows(t *testing.T) {
	dir := t.TempDir()
	writeOrcYAML(t, dir, `
workflows:
  default:
    stages:
      - name: intake
      - name: develop
  hotfix:
    stages:
      - name: develop
      - name: pr-open
`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := cfg.Names()
	if len(names) != 2 {
		t.Errorf("Names() = %v, want 2 workflows", names)
	}

	if stages := cfg.StageNames("hotfix"); len(stages) != 2 || stages[0] != "develop" {
		t.Errorf("StageNames(hotfix) = %v, want [develop pr-open]", stages)
	}
}
