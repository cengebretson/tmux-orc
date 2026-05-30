package workflow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cengebretson/orc/internal/workflow"
)

func writeWorkflow(t *testing.T, dir, name, content string) {
	t.Helper()
	wfDir := filepath.Join(dir, name)
	if err := os.MkdirAll(wfDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "WORKFLOW.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := workflow.Load(dir, "nonexistent")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.NextWorkflow != "" || cfg.Advance != "" || cfg.Worker != "" {
		t.Errorf("expected empty config, got %+v", cfg)
	}
}

func TestLoad_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "intake", "# Workflow: intake\n\nSome content here.\n")

	cfg, err := workflow.Load(dir, "intake")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.NextWorkflow != "" || cfg.Advance != "" {
		t.Errorf("expected empty config for file without frontmatter, got %+v", cfg)
	}
}

func TestLoad_FullFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "intake", `---
next_workflow: develop
next_stage: implementation
advance: auto
worker: fred-documentor
---

# Workflow: intake
`)

	cfg, err := workflow.Load(dir, "intake")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.NextWorkflow != "develop" {
		t.Errorf("NextWorkflow: got %q, want %q", cfg.NextWorkflow, "develop")
	}
	if cfg.NextStage != "implementation" {
		t.Errorf("NextStage: got %q, want %q", cfg.NextStage, "implementation")
	}
	if cfg.Advance != "auto" {
		t.Errorf("Advance: got %q, want %q", cfg.Advance, "auto")
	}
	if cfg.Worker != "fred-documentor" {
		t.Errorf("Worker: got %q, want %q", cfg.Worker, "fred-documentor")
	}
}

func TestLoad_PartialFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "qa-automation", `---
advance: manual
worker: fred-documentor
---

# Workflow: qa-automation
`)

	cfg, err := workflow.Load(dir, "qa-automation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Advance != "manual" {
		t.Errorf("Advance: got %q, want %q", cfg.Advance, "manual")
	}
	if cfg.NextWorkflow != "" {
		t.Errorf("expected empty NextWorkflow, got %q", cfg.NextWorkflow)
	}
	if cfg.NextStage != "" {
		t.Errorf("expected empty NextStage, got %q", cfg.NextStage)
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "broken", "---\n: : : invalid yaml\n---\n\n# Workflow: broken\n")

	cfg, err := workflow.Load(dir, "broken")
	if err != nil {
		t.Fatalf("expected no error on malformed YAML, got %v", err)
	}
	if cfg.NextWorkflow != "" || cfg.Advance != "" {
		t.Errorf("expected empty config on malformed YAML, got %+v", cfg)
	}
}
