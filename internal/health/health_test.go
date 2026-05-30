package health_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cengebretson/orc/internal/health"
)

func fixtureWorkspace() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "workspace")
}

func TestRun_ValidWorkspace(t *testing.T) {
	report := health.Run(fixtureWorkspace())

	if !report.OK() {
		for _, r := range report.Results {
			if r.Status == health.Missing {
				t.Errorf("unexpected missing: %s — %s", r.Name, r.Detail)
			}
		}
	}
}

func TestRun_MissingWorkspace(t *testing.T) {
	report := health.Run("/nonexistent/workspace")

	if report.OK() {
		t.Error("expected report to not be OK for missing workspace")
	}
}

func TestRun_CountsWorkers(t *testing.T) {
	report := health.Run(fixtureWorkspace())

	var workersResult *health.Result
	for i := range report.Results {
		if report.Results[i].Name == "workers/" {
			workersResult = &report.Results[i]
			break
		}
	}

	if workersResult == nil {
		t.Fatal("workers/ result not found in report")
	}
	if workersResult.Status != health.OK {
		t.Errorf("workers/ status = %v, want OK", workersResult.Status)
	}
}
