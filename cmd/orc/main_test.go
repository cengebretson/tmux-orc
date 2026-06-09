package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cengebretson/orc/internal/state"
)

func TestRunStatusTicketPrintsDetail(t *testing.T) {
	resetCommandGlobals(t)
	globalWorkspace = fixtureWorkspace()

	out, err := captureStdout(func() error {
		return runStatus(nil, []string{"STORY-123"})
	})
	if err != nil {
		t.Fatalf("runStatus: %v", err)
	}

	for _, want := range []string{
		"Ticket:   STORY-123",
		"Stage:     default · develop",
		"Worker:  Bob (Developer) (codex)",
		"Run `orc next` to launch.",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("status output missing %q:\n%s", want, out)
		}
	}
}

func TestRunStatusJSONPrintsActiveAndArchived(t *testing.T) {
	resetCommandGlobals(t)
	globalWorkspace = fixtureWorkspace()
	statusJSON = true

	out, err := captureStdout(func() error {
		return runStatus(nil, nil)
	})
	if err != nil {
		t.Fatalf("runStatus --json: %v", err)
	}

	var payload struct {
		Active   []map[string]any `json:"active"`
		Archived []map[string]any `json:"archived"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("unmarshal status json: %v\n%s", err, out)
	}
	if len(payload.Active) == 0 {
		t.Fatal("active list is empty")
	}
	if len(payload.Archived) != 1 {
		t.Fatalf("archived count = %d, want 1", len(payload.Archived))
	}
	if payload.Archived[0]["Ticket"] != "STORY-101" {
		t.Fatalf("archived ticket = %v, want STORY-101", payload.Archived[0]["Ticket"])
	}
}

func TestRunHealthTicketPrintsValidationReport(t *testing.T) {
	resetCommandGlobals(t)
	globalWorkspace = fixtureWorkspace()

	out, err := captureStdout(func() error {
		return runHealth(nil, []string{"STORY-123"})
	})
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("runHealth err = %v, want validation failed", err)
	}
	for _, want := range []string{
		"Ticket: STORY-123",
		"✓  STATE.yaml",
		"✗  STATE.yaml.repos.worktree",
		"Some checks failed",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("health output missing %q:\n%s", want, out)
		}
	}
}

func TestRunJITDryPrintsResolvedWorkerAndPrompt(t *testing.T) {
	resetCommandGlobals(t)
	globalWorkspace = fixtureWorkspace()
	jitDry = true
	jitWorker = "bob-developer"

	out, err := captureStdout(func() error {
		return runJIT(nil, []string{"STORY-123", "check state"})
	})
	if err != nil {
		t.Fatalf("runJIT --dry: %v", err)
	}

	for _, want := range []string{
		"Worker:  Bob (Developer) (codex)",
		"Model:   gpt-5.5",
		"Would run:",
		"## JIT task: STORY-123",
		"check state",
		"orc mark STORY-123 jit",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("jit output missing %q:\n%s", want, out)
		}
	}
}

func TestRunMarkPauseUpdatesCopiedFixture(t *testing.T) {
	resetCommandGlobals(t)
	globalWorkspace = mutableFixtureWorkspace(t)

	out, err := captureStdout(func() error {
		return runMark(nil, []string{"HOT-42", "pause", "waiting", "for", "ops"})
	})
	if err != nil {
		t.Fatalf("runMark pause: %v", err)
	}
	if !strings.Contains(out, "Status:  paused") || !strings.Contains(out, "Reason:  waiting for ops") {
		t.Fatalf("pause output unexpected:\n%s", out)
	}

	s := loadTicketState(t, globalWorkspace, "HOT-42")
	if s.Status != "paused" {
		t.Fatalf("status = %q, want paused", s.Status)
	}
	if got := s.History[len(s.History)-1].Result; got != "paused — waiting for ops" {
		t.Fatalf("last history result = %q", got)
	}
}

func TestRunMarkStartUpdatesPendingTicket(t *testing.T) {
	resetCommandGlobals(t)
	globalWorkspace = mutableFixtureWorkspace(t)
	featureDir := filepath.Join(globalWorkspace, "features", "HOT-42-login-500-error")
	if err := state.Update(featureDir, func(s *state.State) error {
		s.Status = "pending"
		return nil
	}); err != nil {
		t.Fatalf("Update setup: %v", err)
	}

	out, err := captureStdout(func() error {
		return runMark(nil, []string{"HOT-42", "start"})
	})
	if err != nil {
		t.Fatalf("runMark start: %v", err)
	}
	if !strings.Contains(out, "Status:  active") {
		t.Fatalf("start output unexpected:\n%s", out)
	}

	s := loadTicketState(t, globalWorkspace, "HOT-42")
	if s.Status != "active" {
		t.Fatalf("status = %q, want active", s.Status)
	}
}

func TestRunMarkDoneUpdatesCopiedFixture(t *testing.T) {
	resetCommandGlobals(t)
	globalWorkspace = mutableFixtureWorkspace(t)
	markResult = "implemented and verified"

	out, err := captureStdout(func() error {
		return runMark(nil, []string{"HOT-42", "done"})
	})
	if err != nil {
		t.Fatalf("runMark done: %v", err)
	}
	if !strings.Contains(out, "Status:  done") {
		t.Fatalf("done output unexpected:\n%s", out)
	}

	s := loadTicketState(t, globalWorkspace, "HOT-42")
	if s.Status != "done" {
		t.Fatalf("status = %q, want done", s.Status)
	}
	if got := s.History[len(s.History)-1].Result; got != "implemented and verified" {
		t.Fatalf("last history result = %q", got)
	}
}

func TestRunMarkDoneRejectsPendingTicket(t *testing.T) {
	resetCommandGlobals(t)
	globalWorkspace = mutableFixtureWorkspace(t)
	featureDir := filepath.Join(globalWorkspace, "features", "HOT-42-login-500-error")
	if err := state.Update(featureDir, func(s *state.State) error {
		s.Status = "pending"
		return nil
	}); err != nil {
		t.Fatalf("Update setup: %v", err)
	}

	_, err := captureStdout(func() error {
		return runMark(nil, []string{"HOT-42", "done"})
	})
	if err == nil || !strings.Contains(err.Error(), "cannot mark HOT-42 done from status \"pending\"") {
		t.Fatalf("runMark done err = %v", err)
	}
}

func TestRunMarkJITClearsRuntimeAndRecordsHistory(t *testing.T) {
	resetCommandGlobals(t)
	globalWorkspace = mutableFixtureWorkspace(t)

	out, err := captureStdout(func() error {
		return runMark(nil, []string{"PROJ-099", "jit", "review", "completed"})
	})
	if err != nil {
		t.Fatalf("runMark jit: %v", err)
	}
	if !strings.Contains(out, "Done: jit task recorded for PROJ-099") {
		t.Fatalf("jit output unexpected:\n%s", out)
	}

	s := loadTicketState(t, globalWorkspace, "PROJ-099")
	if s.Runtime.JIT != nil {
		t.Fatalf("runtime.jit still present: %#v", s.Runtime.JIT)
	}
	last := s.History[len(s.History)-1]
	if last.Stage != "jit" || last.Worker != "zach-the-reviewer" || last.Result != "review completed" {
		t.Fatalf("last history = %#v", last)
	}
}

func TestRunArchiveMovesDoneTicketToArchive(t *testing.T) {
	resetCommandGlobals(t)
	globalWorkspace = mutableFixtureWorkspace(t)

	if _, err := captureStdout(func() error {
		return runMark(nil, []string{"HOT-42", "done"})
	}); err != nil {
		t.Fatalf("mark done before archive: %v", err)
	}

	out, err := captureStdout(func() error {
		return runArchive(nil, []string{"HOT-42"})
	})
	if err != nil {
		t.Fatalf("runArchive: %v", err)
	}
	if !strings.Contains(out, "Archived: features/_archive/HOT-42-login-500-error/") {
		t.Fatalf("archive output unexpected:\n%s", out)
	}

	activeDir := filepath.Join(globalWorkspace, "features", "HOT-42-login-500-error")
	if _, err := os.Stat(activeDir); !os.IsNotExist(err) {
		t.Fatalf("active dir still exists or stat failed unexpectedly: %v", err)
	}
	archivedDir := filepath.Join(globalWorkspace, "features", "_archive", "HOT-42-login-500-error")
	s, err := state.Load(archivedDir)
	if err != nil {
		t.Fatalf("load archived state: %v", err)
	}
	if s.Status != "archived" {
		t.Fatalf("archived status = %q, want archived", s.Status)
	}
}

func TestRunDeleteRefusesActiveTicket(t *testing.T) {
	resetCommandGlobals(t)
	globalWorkspace = mutableFixtureWorkspace(t)

	_, err := captureStdout(func() error {
		return runDelete(nil, []string{"STORY-123"})
	})
	if err == nil || !strings.Contains(err.Error(), "must be done or archived") {
		t.Fatalf("runDelete err = %v, want refusal", err)
	}
	if _, statErr := os.Stat(filepath.Join(globalWorkspace, "features", "STORY-123-add-user-auth")); statErr != nil {
		t.Fatalf("active ticket folder missing after refused delete: %v", statErr)
	}
}

func fixtureWorkspace() string {
	return "../../testdata/workspace"
}

func mutableFixtureWorkspace(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.CopyFS(root, os.DirFS(fixtureWorkspace())); err != nil {
		t.Fatalf("copy fixture workspace: %v", err)
	}
	return root
}

func loadTicketState(t *testing.T, root, query string) *state.State {
	t.Helper()
	featureDir, err := state.FindFeatureDirWithArchive(root, query)
	if err != nil {
		t.Fatalf("FindFeatureDirWithArchive(%q): %v", query, err)
	}
	s, err := state.Load(featureDir)
	if err != nil {
		t.Fatalf("Load(%q): %v", featureDir, err)
	}
	return s
}

func resetCommandGlobals(t *testing.T) {
	t.Helper()

	oldWorkspace := globalWorkspace
	oldStatusJSON := statusJSON
	oldJITDry := jitDry
	oldJITWorker := jitWorker
	oldJITTmux := jitTmux
	oldMarkWorker := markWorker
	oldMarkResult := markResult
	oldMarkStage := markStage
	t.Cleanup(func() {
		globalWorkspace = oldWorkspace
		statusJSON = oldStatusJSON
		jitDry = oldJITDry
		jitWorker = oldJITWorker
		jitTmux = oldJITTmux
		markWorker = oldMarkWorker
		markResult = oldMarkResult
		markStage = oldMarkStage
	})

	globalWorkspace = "."
	statusJSON = false
	jitDry = false
	jitWorker = ""
	jitTmux = false
	markWorker = ""
	markResult = ""
	markStage = ""
}

func captureStdout(fn func() error) (string, error) {
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	var buf bytes.Buffer
	readDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(&buf, r)
		readDone <- copyErr
	}()

	fnErr := fn()
	closeErr := w.Close()
	os.Stdout = orig
	readErr := <-readDone
	_ = r.Close()

	return buf.String(), errors.Join(fnErr, closeErr, readErr)
}
