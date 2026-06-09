package doctor_test

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cengebretson/orc/internal/doctor"
)

func fixtureWorkspace() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "workspace")
}

func TestRunWithOptionsRequiredToolsPresent(t *testing.T) {
	report := doctor.RunWithOptions(fixtureWorkspace(), doctor.Options{
		LookPath: func(name string) (string, error) {
			return "/bin/" + name, nil
		},
	})

	if !report.OK() {
		for _, c := range report.Checks {
			if c.Status == doctor.Fail {
				t.Errorf("unexpected failure: %s/%s %s", c.Group, c.Name, c.Detail)
			}
		}
	}
}

func TestRunWithOptionsMissingRequiredWorkerEngineFails(t *testing.T) {
	report := doctor.RunWithOptions(fixtureWorkspace(), doctor.Options{
		LookPath: func(name string) (string, error) {
			if name == "codex" {
				return "", errors.New("missing")
			}
			return "/bin/" + name, nil
		},
	})

	check := findCheck(report, "tools", "codex")
	if check == nil {
		t.Fatal("codex check not found")
	}
	if check.Status != doctor.Fail {
		t.Fatalf("codex status = %v, want Fail", check.Status)
	}
	if report.OK() {
		t.Fatal("report should fail when required worker engine is missing")
	}
}

func TestRunWithOptionsMissingTmuxWarns(t *testing.T) {
	report := doctor.RunWithOptions(fixtureWorkspace(), doctor.Options{
		LookPath: func(name string) (string, error) {
			if name == "tmux" {
				return "", errors.New("missing")
			}
			return "/bin/" + name, nil
		},
	})

	check := findCheck(report, "tools", "tmux")
	if check == nil {
		t.Fatal("tmux check not found")
	}
	if check.Status != doctor.Warning {
		t.Fatalf("tmux status = %v, want Warning", check.Status)
	}
	if !report.OK() {
		t.Fatal("report should remain OK when only tmux is missing")
	}
}

func findCheck(report *doctor.Report, group, name string) *doctor.Check {
	for i := range report.Checks {
		if report.Checks[i].Group == group && report.Checks[i].Name == name {
			return &report.Checks[i]
		}
	}
	return nil
}
