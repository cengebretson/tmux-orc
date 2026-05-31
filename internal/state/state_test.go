package state_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cengebretson/orc/internal/state"
)

func fixtureWorkspace() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "workspace")
}

func TestLoad(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := filepath.Join(ws, "features", "STORY-123-add-user-auth")

	s, err := state.Load(featureDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if s.Ticket != "STORY-123" {
		t.Errorf("ticket = %q, want STORY-123", s.Ticket)
	}
	if s.Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", s.Status)
	}
	if s.Stage.Workflow != "develop" {
		t.Errorf("stage.workflow = %q, want develop", s.Stage.Workflow)
	}
	if s.Stage.Owner != "bob-developer" {
		t.Errorf("stage.owner = %q, want bob-developer", s.Stage.Owner)
	}
	if s.NextAction.Worker != "bob-developer" {
		t.Errorf("next_action.worker = %q, want bob-developer", s.NextAction.Worker)
	}
}

func TestLoad_Missing(t *testing.T) {
	_, err := state.Load("/nonexistent/feature")
	if err == nil {
		t.Fatal("expected error for missing STATE.yaml, got nil")
	}
}

func TestFindFeatureDir_ExactSlug(t *testing.T) {
	ws := fixtureWorkspace()
	dir, err := state.FindFeatureDir(ws, "STORY-123-add-user-auth")
	if err != nil {
		t.Fatalf("FindFeatureDir: %v", err)
	}
	if filepath.Base(dir) != "STORY-123-add-user-auth" {
		t.Errorf("dir = %q, want STORY-123-add-user-auth", filepath.Base(dir))
	}
}

func TestFindFeatureDir_TicketPrefix(t *testing.T) {
	ws := fixtureWorkspace()
	dir, err := state.FindFeatureDir(ws, "STORY-456")
	if err != nil {
		t.Fatalf("FindFeatureDir: %v", err)
	}
	if filepath.Base(dir) != "STORY-456-export-api" {
		t.Errorf("dir = %q, want STORY-456-export-api", filepath.Base(dir))
	}
}

func TestFindFeatureDir_NotFound(t *testing.T) {
	ws := fixtureWorkspace()
	_, err := state.FindFeatureDir(ws, "NOTREAL-999")
	if err == nil {
		t.Fatal("expected error for missing feature, got nil")
	}
}

func TestFindFeatureDir_Ambiguous(t *testing.T) {
	ws := fixtureWorkspace()
	// "STORY" matches STORY-123, STORY-456, and STORY-789
	_, err := state.FindFeatureDir(ws, "STORY")
	if err == nil {
		t.Fatal("expected error for ambiguous slug, got nil")
	}
}

func TestResolveCWD_Dot(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := filepath.Join(ws, "features", "STORY-456-export-api")
	s, _ := state.Load(featureDir)

	cwd := s.ResolveCWD(ws, featureDir)
	if cwd != ws {
		t.Errorf("cwd = %q, want workspace root %q", cwd, ws)
	}
}

func TestResolveCWD_Relative(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := filepath.Join(ws, "features", "STORY-123-add-user-auth")
	s, _ := state.Load(featureDir)

	cwd := s.ResolveCWD(ws, featureDir)
	// cwd in fixture is ../../worktrees/my-app/STORY-123-add-user-auth
	// resolved from featureDir that should produce a path ending in worktrees/...
	if cwd == "" || cwd == "." {
		t.Errorf("expected resolved path, got %q", cwd)
	}
}
