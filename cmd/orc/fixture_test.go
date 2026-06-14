package main

import (
	"testing"

	"github.com/cengebretson/orc/internal/health"
)

// TestFixtureWorkspace_ClosureComplete guards the hand-maintained
// testdata/workspace fixture against drift. The fixture is a parallel copy of
// template-shaped content (orc.yaml, workers, stages) curated to be richer than
// a fresh scaffold — multiple workflows, archived tickets, per-stage subfolders.
// Because it is hand-maintained, a bad edit (a workflow stage routing to a
// worker or stage file that does not exist) would silently make it
// unrepresentative. This runs the same closure check orc doctor runs (health
// "workflow refs") against the fixture in place — it is read-only, so no copy is
// needed — and fails CI on drift.
func TestFixtureWorkspace_ClosureComplete(t *testing.T) {
	report := health.Run(fixtureWorkspace())

	var refs *health.Result
	for i := range report.Results {
		if report.Results[i].Name == "workflow refs" {
			refs = &report.Results[i]
			break
		}
	}
	if refs == nil {
		t.Fatal("no 'workflow refs' result in health report")
	}
	if refs.Status != health.OK {
		t.Errorf("testdata/workspace fixture is not closure-complete: %s", refs.Detail)
	}
}
