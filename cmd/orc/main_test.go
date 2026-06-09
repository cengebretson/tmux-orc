package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
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
		"✗  worktree",
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

func fixtureWorkspace() string {
	return "../../testdata/workspace"
}

func resetCommandGlobals(t *testing.T) {
	t.Helper()

	oldWorkspace := globalWorkspace
	oldStatusJSON := statusJSON
	oldJITDry := jitDry
	oldJITWorker := jitWorker
	t.Cleanup(func() {
		globalWorkspace = oldWorkspace
		statusJSON = oldStatusJSON
		jitDry = oldJITDry
		jitWorker = oldJITWorker
	})

	globalWorkspace = "."
	statusJSON = false
	jitDry = false
	jitWorker = ""
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
