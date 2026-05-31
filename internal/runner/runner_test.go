package runner_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cengebretson/orc/internal/runner"
	"github.com/cengebretson/orc/internal/state"
)

func fixtureWorkspace() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "workspace")
}

func fixtureFeatureDir(ws, ticket string) string {
	entries, _ := filepath.Glob(filepath.Join(ws, "features", ticket+"*"))
	if len(entries) == 0 {
		return ""
	}
	return entries[0]
}

func TestCompute_ResolvesWorkerFromConfig(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := fixtureFeatureDir(ws, "STORY-123")
	if featureDir == "" {
		t.Fatal("fixture STORY-123 not found")
	}

	plan, err := runner.Compute(ws, featureDir, "")
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if plan.Worker == nil {
		t.Fatal("expected a worker, got nil")
	}
	// STORY-123 has stage.owner set, so it resolves via stage owner
	if plan.WorkerReason != "stage owner" && plan.WorkerReason != "workflow default" {
		t.Errorf("WorkerReason = %q, want stage owner or workflow default", plan.WorkerReason)
	}
}

func TestCompute_FlagOverrideTakesPriority(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := fixtureFeatureDir(ws, "STORY-123")
	if featureDir == "" {
		t.Fatal("fixture STORY-123 not found")
	}

	plan, err := runner.Compute(ws, featureDir, "fred-documentor")
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if plan.Worker.ID != "fred-documentor" {
		t.Errorf("Worker.ID = %q, want fred-documentor", plan.Worker.ID)
	}
	if plan.WorkerReason != "flag override" {
		t.Errorf("WorkerReason = %q, want \"flag override\"", plan.WorkerReason)
	}
}

func TestCompute_PromptContainsPreamble(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := fixtureFeatureDir(ws, "STORY-123")
	if featureDir == "" {
		t.Fatal("fixture STORY-123 not found")
	}

	plan, err := runner.Compute(ws, featureDir, "")
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if !strings.Contains(plan.Prompt, "orc start") {
		t.Errorf("prompt missing preamble orc start instruction")
	}
	if !strings.Contains(plan.Prompt, "AGENTS.md") {
		t.Errorf("prompt missing AGENTS.md reference")
	}
}

func TestCompute_PromptContainsEndInstruction(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := fixtureFeatureDir(ws, "STORY-123")
	if featureDir == "" {
		t.Fatal("fixture STORY-123 not found")
	}

	plan, err := runner.Compute(ws, featureDir, "")
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	// STORY-123 is at develop stage which has a next stage, so end instruction must be present
	if plan.EndInstruction == "" {
		t.Error("expected end instruction for non-final stage, got empty")
	}
	if !strings.Contains(plan.Prompt, plan.EndInstruction) {
		t.Error("end instruction not appended to prompt")
	}
}

func TestCompute_LaunchCommandNonEmpty(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := fixtureFeatureDir(ws, "STORY-123")
	if featureDir == "" {
		t.Fatal("fixture STORY-123 not found")
	}

	plan, err := runner.Compute(ws, featureDir, "")
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if plan.LaunchCommand == "" {
		t.Error("LaunchCommand is empty")
	}
	if len(plan.LaunchArgv) == 0 {
		t.Error("LaunchArgv is empty")
	}
}

func TestCompute_WorkflowAndStagePopulated(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := fixtureFeatureDir(ws, "STORY-123")
	if featureDir == "" {
		t.Fatal("fixture STORY-123 not found")
	}

	s, _ := state.Load(featureDir)
	plan, err := runner.Compute(ws, featureDir, "")
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if plan.Stage != s.Stage.Name {
		t.Errorf("Stage = %q, want %q", plan.Stage, s.Stage.Name)
	}
	if plan.Workflow == "" {
		t.Error("Workflow is empty")
	}
	if plan.Ticket != s.Ticket {
		t.Errorf("Ticket = %q, want %q", plan.Ticket, s.Ticket)
	}
}
