package validate_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cengebretson/orc/internal/validate"
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

func TestRun_ValidTicketPasses(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := fixtureFeatureDir(ws, "STORY-123")
	if featureDir == "" {
		t.Fatal("fixture STORY-123 not found")
	}

	report := validate.Run(ws, featureDir)

	if report.Ticket == "" {
		t.Error("Ticket is empty")
	}
	// Core checks must pass; worktree checks may fail since test env has no real worktrees.
	coreChecks := map[string]bool{"STATE.yaml": false, "orc.yaml": false, "workflow": false, "stage in workflow": false}
	for _, c := range report.Checks {
		if _, isCoreCheck := coreChecks[c.Name]; isCoreCheck {
			coreChecks[c.Name] = true
			if c.Status == validate.Fail {
				t.Errorf("core check failed: %s — %s", c.Name, c.Detail)
			}
		}
	}
	for name, seen := range coreChecks {
		if !seen {
			t.Errorf("core check %q not present in report", name)
		}
	}
}

func TestRun_ReturnsTicketID(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := fixtureFeatureDir(ws, "STORY-123")
	if featureDir == "" {
		t.Fatal("fixture STORY-123 not found")
	}

	report := validate.Run(ws, featureDir)

	if report.Ticket != "STORY-123" {
		t.Errorf("Ticket = %q, want STORY-123", report.Ticket)
	}
}

func TestRun_MissingFeatureDirFails(t *testing.T) {
	ws := fixtureWorkspace()
	report := validate.Run(ws, "/nonexistent/feature/dir")

	if report.OK() {
		t.Error("expected report to fail for missing feature dir")
	}
}

func TestRun_WorkflowCheckPresent(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := fixtureFeatureDir(ws, "STORY-123")
	if featureDir == "" {
		t.Fatal("fixture STORY-123 not found")
	}

	report := validate.Run(ws, featureDir)

	var found bool
	for _, c := range report.Checks {
		if c.Name == "workflow" {
			found = true
			if c.Status == validate.Fail {
				t.Errorf("workflow check failed: %s", c.Detail)
			}
		}
	}
	if !found {
		t.Error("workflow check not present in report")
	}
}

func TestRun_StageCheckPresent(t *testing.T) {
	ws := fixtureWorkspace()
	featureDir := fixtureFeatureDir(ws, "STORY-123")
	if featureDir == "" {
		t.Fatal("fixture STORY-123 not found")
	}

	report := validate.Run(ws, featureDir)

	var found bool
	for _, c := range report.Checks {
		if c.Name == "stage in workflow" {
			found = true
			if c.Status == validate.Fail {
				t.Errorf("stage check failed: %s", c.Detail)
			}
		}
	}
	if !found {
		t.Error("stage check not present in report")
	}
}

func TestRun_InvalidWorkspaceFailsConfig(t *testing.T) {
	report := validate.Run("/nonexistent/workspace", "/nonexistent/workspace/features/X-1")

	if report.OK() {
		t.Error("expected report to fail for invalid workspace")
	}
}
