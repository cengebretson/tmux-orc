package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cengebretson/orc/internal/workspace"
)

func TestInit_CreatesExpectedFiles(t *testing.T) {
	dir := t.TempDir()

	err := workspace.Init(workspace.InitOptions{Root: dir})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	required := []string{
		"AGENTS.md",
		"CLAUDE.md",
		"ROUTER.md",
		"TOOLS.md",
		"RULES.md",
		"SETUP.md",
		"features/_template/STATE.yaml",
		"features/_template/TICKET.md",
		"workers/_template.md",
		"orc.yaml",
		"ORC.md",
		"stages/intake.md",
		"stages/develop.md",
		"stages/pr-open.md",
		"stages/pr-repair.md",
		"stages/qa-automation.md",
	}

	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("missing expected file: %s", rel)
		}
	}
}

func TestInit_DryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()

	err := workspace.Init(workspace.InitOptions{Root: dir, DryRun: true})
	if err != nil {
		t.Fatalf("Init dry-run: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("dry-run wrote %d files, want 0", len(entries))
	}
}

func TestInit_SkipsExistingWithoutForce(t *testing.T) {
	dir := t.TempDir()

	// write once
	if err := workspace.Init(workspace.InitOptions{Root: dir}); err != nil {
		t.Fatalf("first Init: %v", err)
	}

	// overwrite a file to detect if it gets reset
	agentsPath := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("custom"), 0644); err != nil {
		t.Fatal(err)
	}

	// write again without force
	if err := workspace.Init(workspace.InitOptions{Root: dir}); err != nil {
		t.Fatalf("second Init: %v", err)
	}

	data, _ := os.ReadFile(agentsPath)
	if string(data) != "custom" {
		t.Error("Init without --force overwrote an existing file")
	}
}

func TestInit_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()

	if err := workspace.Init(workspace.InitOptions{Root: dir}); err != nil {
		t.Fatalf("first Init: %v", err)
	}

	agentsPath := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("custom"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := workspace.Init(workspace.InitOptions{Root: dir, Force: true}); err != nil {
		t.Fatalf("forced Init: %v", err)
	}

	data, _ := os.ReadFile(agentsPath)
	if string(data) == "custom" {
		t.Error("Init --force did not overwrite existing file")
	}
}

func TestInit_WithSampleWorkers(t *testing.T) {
	dir := t.TempDir()

	if err := workspace.Init(workspace.InitOptions{Root: dir, WithSampleWorkers: true}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	samples := []string{
		"workers/bob-the-developer.md",
		"workers/fred-the-documentor.md",
		"workers/intake-agent.md",
	}
	for _, rel := range samples {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("missing sample worker: %s", rel)
		}
	}
}

func TestWork_CreatesFeatureFolder(t *testing.T) {
	dir := t.TempDir()

	if err := workspace.Init(workspace.InitOptions{Root: dir}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	result, err := workspace.Work(workspace.WorkOptions{Root: dir, Ticket: "TEST-001"})
	if err != nil {
		t.Fatalf("Work: %v", err)
	}

	if _, err := os.Stat(result.FeatureDir); err != nil {
		t.Errorf("feature dir not created: %v", err)
	}

	stateFile := filepath.Join(result.FeatureDir, "STATE.yaml")
	if _, err := os.Stat(stateFile); err != nil {
		t.Error("STATE.yaml not created")
	}
}

func TestWork_RejectsDuplicate(t *testing.T) {
	dir := t.TempDir()

	if err := workspace.Init(workspace.InitOptions{Root: dir}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if _, err := workspace.Work(workspace.WorkOptions{Root: dir, Ticket: "TEST-001"}); err != nil {
		t.Fatalf("first Work: %v", err)
	}

	_, err := workspace.Work(workspace.WorkOptions{Root: dir, Ticket: "TEST-001"})
	if err == nil {
		t.Error("expected error for duplicate ticket, got nil")
	}
}

func TestWork_SlugOverride(t *testing.T) {
	dir := t.TempDir()

	if err := workspace.Init(workspace.InitOptions{Root: dir}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	result, err := workspace.Work(workspace.WorkOptions{
		Root:   dir,
		Ticket: "TEST-002",
		Slug:   "add-login",
	})
	if err != nil {
		t.Fatalf("Work: %v", err)
	}

	if result.Slug != "TEST-002-add-login" {
		t.Errorf("slug = %q, want TEST-002-add-login", result.Slug)
	}
}
