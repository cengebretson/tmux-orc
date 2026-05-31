package workers_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cengebretson/orc/internal/workers"
)

func fixtureWorkersDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "workspace", "workers")
}

func TestLoad(t *testing.T) {
	all, err := workers.Load(fixtureWorkersDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("loaded %d workers, want 3", len(all))
	}
}

func TestLoad_ParsesFrontmatter(t *testing.T) {
	all, err := workers.Load(fixtureWorkersDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	bob := findWorker(all, "bob-developer")
	if bob == nil {
		t.Fatal("bob-developer not found")
	}
	if bob.Product != "codex" {
		t.Errorf("product = %q, want codex", bob.Product)
	}
	if bob.CostTier != "medium" {
		t.Errorf("cost_tier = %q, want medium", bob.CostTier)
	}
	if len(bob.Workflows) == 0 {
		t.Error("expected workflows to be populated")
	}
}

func TestMatch_ByWorkflow(t *testing.T) {
	all, _ := workers.Load(fixtureWorkersDir())

	matched := workers.Match(all, "develop")
	if len(matched) != 2 {
		t.Fatalf("matched %d workers, want 2 (both support develop)", len(matched))
	}
}

func TestMatch_QAAutomation(t *testing.T) {
	all, _ := workers.Load(fixtureWorkersDir())

	matched := workers.Match(all, "qa-automation")
	if len(matched) != 3 {
		t.Fatalf("matched %d workers for qa-automation, want 3", len(matched))
	}
}

func TestMatch_PRRepair(t *testing.T) {
	all, _ := workers.Load(fixtureWorkersDir())

	matched := workers.Match(all, "pr-repair")
	if len(matched) != 1 {
		t.Fatalf("matched %d workers for pr-repair, want 1", len(matched))
	}
	if matched[0].ID != "bob-developer" {
		t.Errorf("matched worker = %q, want bob-developer", matched[0].ID)
	}
}

func TestMatch_NoMatch(t *testing.T) {
	all, _ := workers.Load(fixtureWorkersDir())

	matched := workers.Match(all, "nonexistent-workflow")
	if len(matched) != 0 {
		t.Errorf("expected no matches, got %d", len(matched))
	}
}

func TestPreferred_Found(t *testing.T) {
	all, _ := workers.Load(fixtureWorkersDir())
	matched := workers.Match(all, "develop")

	preferred := workers.Preferred(matched, "bob-developer")
	if preferred == nil {
		t.Fatal("expected preferred worker, got nil")
	}
	if preferred.ID != "bob-developer" {
		t.Errorf("preferred = %q, want bob-developer", preferred.ID)
	}
}

func TestPreferred_NotFound(t *testing.T) {
	all, _ := workers.Load(fixtureWorkersDir())
	matched := workers.Match(all, "develop")

	preferred := workers.Preferred(matched, "nonexistent-worker")
	if preferred != nil {
		t.Errorf("expected nil, got %q", preferred.ID)
	}
}

func TestLaunchCommand_Codex(t *testing.T) {
	all, _ := workers.Load(fixtureWorkersDir())
	bob := findWorker(all, "bob-developer")

	cmd := workers.LaunchCommand(bob, "/workspace", "/workspace/worktrees/app/FLYWL-123", "do the thing")
	if cmd == "" {
		t.Error("expected non-empty launch command")
	}
	if bob.Product != "codex" {
		t.Skip("bob is not codex in this fixture")
	}
}

func TestLaunchCommand_Claude(t *testing.T) {
	all, _ := workers.Load(fixtureWorkersDir())
	fred := findWorker(all, "fred-documentor")

	cmd := workers.LaunchCommand(fred, "/workspace", "/workspace", "do the thing")
	if cmd == "" {
		t.Error("expected non-empty launch command")
	}
}

func findWorker(list []*workers.Worker, id string) *workers.Worker {
	for _, w := range list {
		if w.ID == id {
			return w
		}
	}
	return nil
}
