package state_test

import (
	"os"
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
	if s.Status != "active" {
		t.Errorf("status = %q, want active", s.Status)
	}
	if s.Stage.Name != "develop" {
		t.Errorf("stage.name = %q, want develop", s.Stage.Name)
	}
	if s.Stage.Worker != "bob-developer" {
		t.Errorf("stage.worker = %q, want bob-developer", s.Stage.Worker)
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

func writeLegacyState(t *testing.T, featureDir, content string) {
	t.Helper()
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(featureDir, "STATE.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// Legacy STATE.yaml files (written before the workflow field existed) have no workflow key.
// Advance must still work and must not overwrite the stage when stageName is empty.
func TestNext_LegacyStateNoWorkflowField(t *testing.T) {
	dir := t.TempDir()
	featureDir := filepath.Join(dir, "features", "OLD-001")
	writeLegacyState(t, featureDir, `
ticket: OLD-001
slug: OLD-001
status: active
stage:
  worker: bob-developer
  name: develop
`)

	if err := state.Next(featureDir, "pr-open", "bob-developer", "done"); err != nil {
		t.Fatalf("Next: %v", err)
	}

	s, err := state.Load(featureDir)
	if err != nil {
		t.Fatalf("Load after Next: %v", err)
	}
	if s.Stage.Name != "pr-open" {
		t.Errorf("stage.name = %q, want pr-open", s.Stage.Name)
	}
	if s.Status != "ready" {
		t.Errorf("status = %q, want ready", s.Status)
	}
	// workflow field should remain empty — it was never set on this legacy ticket
	if s.Workflow != "" {
		t.Errorf("workflow = %q, want empty (legacy ticket)", s.Workflow)
	}
}

func TestValidateRepos_NoRepos(t *testing.T) {
	s := &state.State{}
	if err := state.ValidateRepos(s, t.TempDir()); err != nil {
		t.Errorf("expected nil for empty repos, got %v", err)
	}
}

func TestValidateRepos_ValidWorktree(t *testing.T) {
	root := t.TempDir()
	wt := filepath.Join(root, "worktrees", "my-app", "TICKET-1")
	if err := os.MkdirAll(wt, 0755); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(root, "my-app")
	if err := os.MkdirAll(mainPath, 0755); err != nil {
		t.Fatal(err)
	}

	s := &state.State{
		Repos: map[string]state.Repo{
			"my-app": {Main: mainPath, Worktree: wt, Branch: "feature/x"},
		},
		NextAction: state.NextAction{CWD: wt},
	}
	if err := state.ValidateRepos(s, root); err != nil {
		t.Errorf("expected nil for valid state, got %v", err)
	}
}

func TestValidateRepos_MissingMain(t *testing.T) {
	root := t.TempDir()
	s := &state.State{
		Repos: map[string]state.Repo{
			"my-app": {Main: "/nonexistent/path", Worktree: "", Branch: ""},
		},
	}
	if err := state.ValidateRepos(s, root); err == nil {
		t.Error("expected error for missing main path, got nil")
	}
}

func TestValidateRepos_WorktreeOutsideWorktreesDir(t *testing.T) {
	root := t.TempDir()
	outsidePath := filepath.Join(root, "somewhere-else", "TICKET-1")
	if err := os.MkdirAll(outsidePath, 0755); err != nil {
		t.Fatal(err)
	}
	s := &state.State{
		Repos: map[string]state.Repo{
			"my-app": {Worktree: outsidePath, Branch: "feature/x"},
		},
	}
	if err := state.ValidateRepos(s, root); err == nil {
		t.Error("expected error for worktree outside worktrees/, got nil")
	}
}

func TestValidateRepos_MissingBranchWhenWorktreeSet(t *testing.T) {
	root := t.TempDir()
	wt := filepath.Join(root, "worktrees", "my-app", "TICKET-1")
	if err := os.MkdirAll(wt, 0755); err != nil {
		t.Fatal(err)
	}
	s := &state.State{
		Repos: map[string]state.Repo{
			"my-app": {Worktree: wt, Branch: ""},
		},
	}
	if err := state.ValidateRepos(s, root); err == nil {
		t.Error("expected error for empty branch with worktree set, got nil")
	}
}

func TestValidateRepos_CWDNotUnderWorktree(t *testing.T) {
	root := t.TempDir()
	wt := filepath.Join(root, "worktrees", "my-app", "TICKET-1")
	if err := os.MkdirAll(wt, 0755); err != nil {
		t.Fatal(err)
	}
	s := &state.State{
		Repos: map[string]state.Repo{
			"my-app": {Worktree: wt, Branch: "feature/x"},
		},
		NextAction: state.NextAction{CWD: "/some/other/path"},
	}
	if err := state.ValidateRepos(s, root); err == nil {
		t.Error("expected error for cwd not under any worktree, got nil")
	}
}

func TestValidateRepos_CWDSkippedWhenNoWorktrees(t *testing.T) {
	root := t.TempDir()
	// Repos set but no worktrees recorded — cwd check should be skipped
	s := &state.State{
		Repos: map[string]state.Repo{
			"my-app": {Main: "", Worktree: "", Branch: ""},
		},
		NextAction: state.NextAction{CWD: "/some/other/path"},
	}
	if err := state.ValidateRepos(s, root); err != nil {
		t.Errorf("expected nil when no worktrees set, got %v", err)
	}
}

// When stageName is empty (last stage in pipeline), Next should set status to "done".
func TestNext_EmptyStageSetsDone(t *testing.T) {
	dir := t.TempDir()
	featureDir := filepath.Join(dir, "features", "DONE-001")
	writeLegacyState(t, featureDir, `
ticket: DONE-001
slug: DONE-001
status: active
workflow: default
stage:
  worker: bob-developer
  name: qa-automation
`)

	if err := state.Next(featureDir, "", "bob-developer", "all tests pass"); err != nil {
		t.Fatalf("Next: %v", err)
	}

	s, err := state.Load(featureDir)
	if err != nil {
		t.Fatalf("Load after Next: %v", err)
	}
	if s.Stage.Name != "qa-automation" {
		t.Errorf("stage.name = %q, want qa-automation (unchanged)", s.Stage.Name)
	}
	if s.Status != "done" {
		t.Errorf("status = %q, want done", s.Status)
	}
}
