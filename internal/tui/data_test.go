package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cengebretson/orc/internal/state"
)

func TestBrokenRowSurfaced(t *testing.T) {
	broken := &featureRow{
		featureDir: "/ws/features/STORY-9-busted",
		loadErr:    errors.New("yaml: line 3: mapping values are not allowed"),
		hasIssues:  true,
	}
	healthy := &featureRow{
		s:          &state.State{Ticket: "STORY-1", Slug: "STORY-1-ok", Status: "active"},
		featureDir: "/ws/features/STORY-1-ok",
	}
	m := New("/ws")
	m.width = 120
	m.features = []*featureRow{broken, healthy}

	// broken rows show even though state can't say whether they're archived
	vis := m.visibleFeatures()
	if len(vis) != 2 {
		t.Fatalf("visibleFeatures = %d rows, want 2", len(vis))
	}
	if broken.ticketID() != "STORY-9-busted" {
		t.Errorf("ticketID = %q, want dir basename fallback", broken.ticketID())
	}

	// renderTable must not panic on a nil-state row and must flag it broken
	out := m.renderTable(vis, m.width, 0)
	if !strings.Contains(out, "broken") {
		t.Errorf("renderTable output missing broken marker:\n%s", out)
	}
	if !strings.Contains(out, "STORY-9-bus") { // truncated to the 12-col ticket width
		t.Errorf("renderTable output missing broken ticket id:\n%s", out)
	}

	// the broken feature viewer surfaces the parse error
	detail := renderBrokenFeature(broken)
	if !strings.Contains(detail, "could not be parsed") || !strings.Contains(detail, "mapping values") {
		t.Errorf("renderBrokenFeature missing error context:\n%s", detail)
	}
}

func TestBuildFileList(t *testing.T) {
	dir := t.TempDir()
	mustWrite := func(rel string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// top-level docs (DECISIONS intentionally absent) and per-stage outputs in
	// folders named after the stages, written out of pipeline order on disk.
	mustWrite("TICKET.md")
	mustWrite("SPEC.md")
	mustWrite("code-review/REVIEW.md")
	mustWrite("develop/HANDOFF.md")
	mustWrite("qa-automation/RESULT.md")
	mustWrite("qa-automation/PLAN.md")
	// a non-stage folder should still surface, after known stages
	mustWrite("scratch/notes.md")
	// hidden / underscore folders are skipped
	mustWrite("_archive/old.md")

	s := &state.State{
		Stage:   state.Stage{Name: "qa-automation"},
		History: []state.HistoryEntry{{Stage: "develop"}, {Stage: "code-review"}},
	}

	got := buildFileList(dir, s)
	var labels []string
	for _, f := range got {
		labels = append(labels, f.label)
	}

	want := []string{
		"TICKET.md",
		"SPEC.md",
		"PLAN.md", // core: always listed even when missing
		"develop/HANDOFF.md",
		"code-review/REVIEW.md",
		"qa-automation/PLAN.md",
		"qa-automation/RESULT.md",
		"scratch/notes.md",
	}
	if len(labels) != len(want) {
		t.Fatalf("labels = %v, want %v", labels, want)
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Errorf("labels[%d] = %q, want %q (full: %v)", i, labels[i], want[i], labels)
		}
	}
}
