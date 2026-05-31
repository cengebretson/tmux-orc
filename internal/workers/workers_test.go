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
}

func TestFindByID_Found(t *testing.T) {
	all, _ := workers.Load(fixtureWorkersDir())

	w := workers.FindByID(all, "bob-developer")
	if w == nil {
		t.Fatal("expected bob-developer, got nil")
	}
	if w.ID != "bob-developer" {
		t.Errorf("id = %q, want bob-developer", w.ID)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	all, _ := workers.Load(fixtureWorkersDir())

	w := workers.FindByID(all, "nonexistent-worker")
	if w != nil {
		t.Errorf("expected nil, got %q", w.ID)
	}
}

func TestLaunchCommand_Codex(t *testing.T) {
	all, _ := workers.Load(fixtureWorkersDir())
	bob := findWorker(all, "bob-developer")

	cmd := workers.LaunchCommand(bob, "/workspace", "/workspace/worktrees/app/FLYWL-123", "do the thing")
	if cmd == "" {
		t.Error("expected non-empty launch command")
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
