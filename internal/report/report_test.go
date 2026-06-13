package report

import (
	"testing"
	"time"

	"github.com/cengebretson/orc/internal/state"
)

// at returns an RFC3339 stamp h hours after a fixed base, for readable fixtures.
var base = time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

func at(d time.Duration) string { return base.Add(d).Format(time.RFC3339) }

func h(n float64) time.Duration { return time.Duration(n * float64(time.Hour)) }

// stageStat finds a stage in a report, failing if absent.
func stageStat(t *testing.T, r Report, name string) StageStat {
	t.Helper()
	for _, s := range r.Stages {
		if s.Stage == name {
			return s
		}
	}
	t.Fatalf("stage %q not in report (have %v)", name, stageNames(r))
	return StageStat{}
}

func stageNames(r Report) []string {
	var out []string
	for _, s := range r.Stages {
		out = append(out, s.Stage)
	}
	return out
}

func TestComputeClosedTicket(t *testing.T) {
	s := &state.State{
		Ticket: "STORY-1",
		Status: "done",
		Stage:  state.Stage{Name: "code-review"},
		History: []state.HistoryEntry{
			{At: at(0), Stage: "intake", Result: "feature context created by orc work"},
			{At: at(h(1)), Stage: "intake", Result: "started"},
			{At: at(h(3)), Stage: "intake", Result: "intake done"},       // enters develop at +3h
			{At: at(h(9)), Stage: "develop", Result: "ready for review"}, // enters code-review at +9h
			{At: at(h(11)), Stage: "code-review", Result: "approved"},    // done at +11h
		},
	}
	r := Compute(s, base.Add(h(99)))

	if r.Open {
		t.Error("done ticket should not be Open")
	}
	if got := stageStat(t, r, "intake").Wall; got != h(3) {
		t.Errorf("intake wall = %v, want 3h", got)
	}
	if got := stageStat(t, r, "develop").Wall; got != h(6) {
		t.Errorf("develop wall = %v, want 6h", got)
	}
	if got := stageStat(t, r, "code-review").Wall; got != h(2) {
		t.Errorf("code-review wall = %v, want 2h", got)
	}
	if r.Wall != h(11) {
		t.Errorf("total wall = %v, want 11h", r.Wall)
	}
	for _, name := range []string{"intake", "develop", "code-review"} {
		if v := stageStat(t, r, name).Visits; v != 1 {
			t.Errorf("%s visits = %d, want 1", name, v)
		}
	}
}

func TestComputeOpenTicketMeasuredToNow(t *testing.T) {
	s := &state.State{
		Ticket: "STORY-2",
		Status: "active",
		Stage:  state.Stage{Name: "develop"},
		History: []state.HistoryEntry{
			{At: at(0), Stage: "intake", Result: "started"},
			{At: at(h(1)), Stage: "intake", Result: "intake done"}, // enters develop at +1h
		},
	}
	now := base.Add(h(4)) // sitting in develop for 3h and counting
	r := Compute(s, now)

	if !r.Open {
		t.Error("active ticket should be Open")
	}
	if got := stageStat(t, r, "develop").Wall; got != h(3) {
		t.Errorf("open develop wall = %v, want 3h (measured to now)", got)
	}
	if got := stageStat(t, r, "develop").Visits; got != 1 {
		t.Errorf("develop visits = %d, want 1", got)
	}
}

func TestComputeSubtractsPause(t *testing.T) {
	s := &state.State{
		Ticket: "STORY-3",
		Status: "done",
		Stage:  state.Stage{Name: "develop"},
		History: []state.HistoryEntry{
			{At: at(0), Stage: "develop", Result: "started"},
			{At: at(h(1)), Stage: "develop", Result: "paused — waiting on API key"}, // pause begins +1h
			{At: at(h(4)), Stage: "develop", Result: "resumed"},                     // 3h paused
			{At: at(h(5)), Stage: "develop", Result: "done"},                        // +1h active
		},
	}
	r := Compute(s, base.Add(h(99)))

	dev := stageStat(t, r, "develop")
	if dev.Wall != h(5) {
		t.Errorf("develop wall = %v, want 5h", dev.Wall)
	}
	if dev.Active != h(2) {
		t.Errorf("develop active = %v, want 2h (5h wall − 3h paused)", dev.Active)
	}
	if dev.Visits != 1 {
		t.Errorf("develop visits = %d, want 1 (pause/resume is one visit)", dev.Visits)
	}
}

func TestComputePausedOpenTicketFreezesActive(t *testing.T) {
	s := &state.State{
		Ticket: "STORY-4",
		Status: "paused",
		Stage:  state.Stage{Name: "develop"},
		History: []state.HistoryEntry{
			{At: at(0), Stage: "develop", Result: "started"},
			{At: at(h(2)), Stage: "develop", Result: "paused — blocked"}, // open interval is paused
		},
	}
	r := Compute(s, base.Add(h(10)))

	dev := stageStat(t, r, "develop")
	if dev.Active != h(2) {
		t.Errorf("develop active = %v, want 2h frozen at the pause", dev.Active)
	}
	if dev.Wall != h(10) {
		t.Errorf("develop wall = %v, want 10h (wall keeps running)", dev.Wall)
	}
}

func TestComputeArchivedTicketIsClosed(t *testing.T) {
	s := &state.State{
		Ticket: "STORY-8",
		Status: "archived", // terminal: no open interval to now
		Stage:  state.Stage{Name: "qa-automation"},
		History: []state.HistoryEntry{
			{At: at(0), Stage: "develop", Result: "implementation complete"},
			{At: at(h(2)), Stage: "qa-automation", Result: "all tests passing, feature archived"},
		},
	}
	r := Compute(s, base.Add(h(500))) // long after the last entry

	if r.Open {
		t.Error("archived ticket should not be Open")
	}
	if r.Wall != h(2) {
		t.Errorf("total wall = %v, want 2h (no open interval measured to now)", r.Wall)
	}
}

func TestComputePausedStatusFreezesActiveRegardlessOfResult(t *testing.T) {
	s := &state.State{
		Ticket: "STORY-9",
		Status: "paused", // status is authoritative even though result says "blocked"
		Stage:  state.Stage{Name: "pr-repair"},
		History: []state.HistoryEntry{
			{At: at(0), Stage: "pr-open", Result: "PR opened"},                   // enters pr-repair
			{At: at(h(2)), Stage: "pr-repair", Result: "blocked — staging down"}, // 2h working, then blocked
		},
	}
	r := Compute(s, base.Add(h(100)))

	repair := stageStat(t, r, "pr-repair")
	if repair.Active != h(2) {
		t.Errorf("pr-repair active = %v, want 2h (blocked period frozen via status)", repair.Active)
	}
	if repair.Wall != h(100) {
		t.Errorf("pr-repair wall = %v, want 100h (wall keeps running while blocked)", repair.Wall)
	}
}

func TestComputeRepairLoopCountsVisits(t *testing.T) {
	s := &state.State{
		Ticket: "STORY-5",
		Status: "done",
		Stage:  state.Stage{Name: "code-review"},
		History: []state.HistoryEntry{
			{At: at(0), Stage: "develop", Result: "started"},
			{At: at(h(2)), Stage: "develop", Result: "ready"},        // enters code-review
			{At: at(h(3)), Stage: "code-review", Result: "changes"},  // enters pr-repair
			{At: at(h(4)), Stage: "pr-repair", Result: "fixed"},      // re-enters develop
			{At: at(h(6)), Stage: "develop", Result: "ready"},        // enters code-review again
			{At: at(h(7)), Stage: "code-review", Result: "approved"}, // done
		},
	}
	r := Compute(s, base.Add(h(99)))

	if v := stageStat(t, r, "develop").Visits; v != 2 {
		t.Errorf("develop visits = %d, want 2 (looped back through pr-repair)", v)
	}
	if v := stageStat(t, r, "code-review").Visits; v != 2 {
		t.Errorf("code-review visits = %d, want 2", v)
	}
	// develop time is summed across both visits: 2h + 2h
	if got := stageStat(t, r, "develop").Wall; got != h(4) {
		t.Errorf("develop wall = %v, want 4h summed across visits", got)
	}
}

func TestComputeSingleEntry(t *testing.T) {
	s := &state.State{
		Ticket: "STORY-6",
		Status: "pending",
		Stage:  state.Stage{Name: "intake"},
		History: []state.HistoryEntry{
			{At: at(0), Stage: "intake", Result: "feature context created by orc work"},
		},
	}
	r := Compute(s, base.Add(h(2)))

	if !r.Open {
		t.Error("brand-new ticket should be Open")
	}
	if got := stageStat(t, r, "intake").Wall; got != h(2) {
		t.Errorf("intake wall = %v, want 2h since creation", got)
	}
}

func TestComputeToleratesBadTimestamps(t *testing.T) {
	s := &state.State{
		Ticket: "STORY-7",
		Status: "done",
		Stage:  state.Stage{Name: "develop"},
		History: []state.HistoryEntry{
			{At: at(0), Stage: "intake", Result: "started"},
			{At: "not-a-timestamp", Stage: "intake", Result: "garbage"}, // skipped
			{At: at(h(2)), Stage: "intake", Result: "ready"},            // enters develop
			{At: at(h(1)), Stage: "develop", Result: "done"},            // out of order → clamped to 0
		},
	}
	// Must not panic and must not produce negative totals.
	r := Compute(s, base.Add(h(99)))
	if r.Wall < 0 || r.Active < 0 {
		t.Errorf("totals went negative: wall=%v active=%v", r.Wall, r.Active)
	}
	if got := stageStat(t, r, "intake").Wall; got != h(2) {
		t.Errorf("intake wall = %v, want 2h (bad entry skipped)", got)
	}
	if got := stageStat(t, r, "develop").Wall; got != 0 {
		t.Errorf("develop wall = %v, want 0 (out-of-order interval clamped)", got)
	}
}

func TestAggregate(t *testing.T) {
	reports := []Report{
		{Stages: []StageStat{
			{Stage: "intake", Active: h(1), Visits: 1},
			{Stage: "develop", Active: h(2), Visits: 1},
		}},
		{Stages: []StageStat{
			{Stage: "intake", Active: h(3), Visits: 1},
			{Stage: "develop", Active: h(6), Visits: 2},
		}},
		{Stages: []StageStat{
			{Stage: "develop", Active: h(4), Visits: 1},
		}},
	}
	aggs := Aggregate(reports)

	if len(aggs) != 2 {
		t.Fatalf("got %d stages, want 2", len(aggs))
	}
	// first-appearance order: intake then develop
	if aggs[0].Stage != "intake" || aggs[1].Stage != "develop" {
		t.Errorf("stage order = %s,%s, want intake,develop", aggs[0].Stage, aggs[1].Stage)
	}
	dev := aggs[1]
	if dev.Tickets != 3 {
		t.Errorf("develop tickets = %d, want 3", dev.Tickets)
	}
	if dev.Visits != 4 {
		t.Errorf("develop visits = %d, want 4", dev.Visits)
	}
	if dev.AvgActive != h(4) { // (2+6+4)/3
		t.Errorf("develop avg = %v, want 4h", dev.AvgActive)
	}
	if dev.MedActive != h(4) { // sorted 2,4,6 → 4
		t.Errorf("develop median = %v, want 4h", dev.MedActive)
	}
	in := aggs[0]
	if in.Tickets != 2 || in.AvgActive != h(2) { // (1+3)/2
		t.Errorf("intake tickets=%d avg=%v, want 2 and 2h", in.Tickets, in.AvgActive)
	}
	if in.MedActive != h(2) { // even count: (1+3)/2
		t.Errorf("intake median = %v, want 2h", in.MedActive)
	}
}

func TestHumanize(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{12 * time.Minute, "12m"},
		{2 * time.Hour, "2h"},
		{2*time.Hour + 14*time.Minute, "2h 14m"},
		{49 * time.Hour, "2d 1h"},
		{48 * time.Hour, "2d"},
		{-5 * time.Hour, "0s"},
	}
	for _, c := range cases {
		if got := Humanize(c.d); got != c.want {
			t.Errorf("Humanize(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}
