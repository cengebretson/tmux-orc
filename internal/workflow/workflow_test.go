package workflow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cengebretson/orc/internal/workflow"
)

func writeWorkflowsYAML(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "workflows.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := workflow.Load(dir)
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(cfg.Names()) != 0 {
		t.Errorf("expected empty config, got %d workflows", len(cfg.Names()))
	}
}

func TestLoad_BasicWorkflow(t *testing.T) {
	dir := t.TempDir()
	writeWorkflowsYAML(t, dir, `
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

	cfg, err := workflow.Load(dir)
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
	writeWorkflowsYAML(t, dir, `
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

	cfg, err := workflow.Load(dir)
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
	writeWorkflowsYAML(t, dir, `
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

	cfg, err := workflow.Load(dir)
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
	writeWorkflowsYAML(t, dir, `
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

	cfg, err := workflow.Load(dir)
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
	writeWorkflowsYAML(t, dir, `
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

	cfg, err := workflow.Load(dir)
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
