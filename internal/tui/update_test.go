package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cengebretson/orc/internal/doctor"
	"github.com/cengebretson/orc/internal/state"
	"github.com/cengebretson/orc/internal/workers"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// asModel asserts a tea.Model back to the concrete Model.
func asModel(t *testing.T, tm tea.Model) Model {
	t.Helper()
	m, ok := tm.(Model)
	if !ok {
		t.Fatalf("model type = %T, want Model", tm)
	}
	return m
}

// press feeds a sequence of keys through handleKey and returns the final model.
func press(t *testing.T, m Model, keys ...string) (Model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	var tm tea.Model = m
	for _, k := range keys {
		tm, cmd = asModel(t, tm).handleKey(keyMsg(k))
	}
	return asModel(t, tm), cmd
}

// testModel builds a dashboard model with three features and worker/workflow
// sections, sized so enter handlers can construct viewports.
func testModel(t *testing.T) Model {
	t.Helper()
	m := New("")
	m.width = 100
	m.height = 40
	m.features = []*featureRow{
		testRow("STORY-1", "active", "develop"),
		testRow("STORY-2", "paused", "code-review"),
		testRow("AUTH-9", "pending", "intake"),
	}
	m.workflows = testChains()
	m.sectionItems = map[string][]sectionItem{
		"workflows": {{label: "default", path: ""}},
		"workers":   {{label: "Bob", path: "bob.md"}},
	}
	return m
}

func TestHandleKeyQuit(t *testing.T) {
	for _, k := range []string{"q", "ctrl+c"} {
		_, cmd := press(t, testModel(t), k)
		if cmd == nil {
			t.Fatalf("%s should return a quit command", k)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%s returned %T, want tea.QuitMsg", k, cmd())
		}
	}
}

func TestHandleKeyCursorBounds(t *testing.T) {
	m, _ := press(t, testModel(t), "j", "j", "j", "j", "j")
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want clamped at 2", m.cursor)
	}
	m, _ = press(t, m, "k", "k", "k", "k")
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want clamped at 0", m.cursor)
	}
}

func TestHandleKeyArchiveToggle(t *testing.T) {
	m, _ := press(t, testModel(t), "j", "a")
	if !m.showArchived {
		t.Error("a should toggle showArchived on")
	}
	if m.cursor != 0 {
		t.Error("a should reset cursor")
	}
	m, _ = press(t, m, "a")
	if m.showArchived {
		t.Error("a should toggle showArchived off")
	}
}

func TestHandleKeySearch(t *testing.T) {
	m, _ := press(t, testModel(t), "/")
	if !m.searching {
		t.Fatal("/ should enter search mode")
	}

	m, _ = press(t, m, "a", "u", "t", "h")
	if m.search.Value() != "auth" {
		t.Errorf("search value = %q, want auth", m.search.Value())
	}
	if got := len(m.visibleFeatures()); got != 1 {
		t.Errorf("filtered rows = %d, want 1", got)
	}

	// enter keeps the filter, esc clears it
	m, _ = press(t, m, "enter")
	if m.searching || m.search.Value() != "auth" {
		t.Errorf("enter should exit search mode keeping value, got searching=%v value=%q", m.searching, m.search.Value())
	}
	m, _ = press(t, m, "esc")
	if m.search.Value() != "" {
		t.Errorf("esc should clear the filter, got %q", m.search.Value())
	}
}

func TestHandleKeyTabCyclesSections(t *testing.T) {
	m := testModel(t)
	// navigable: health (always), workflows, workers
	m, _ = press(t, m, "tab")
	if m.focusedPane != "section" || m.sectionFocus != "health" {
		t.Fatalf("tab: pane=%q focus=%q, want section/health", m.focusedPane, m.sectionFocus)
	}
	if !m.expanded["health"] {
		t.Error("tab should expand the focused section")
	}
	m, _ = press(t, m, "tab")
	if m.sectionFocus != "workflows" {
		t.Errorf("second tab: focus=%q, want workflows", m.sectionFocus)
	}
	m, _ = press(t, m, "tab", "tab")
	if m.focusedPane != "features" {
		t.Errorf("tab past last section should return to features, got %q", m.focusedPane)
	}

	m, _ = press(t, m, "shift+tab")
	if m.sectionFocus != "workers" {
		t.Errorf("shift+tab from features: focus=%q, want last section workers", m.sectionFocus)
	}

	m, _ = press(t, m, "esc")
	if m.focusedPane != "features" {
		t.Errorf("esc should return focus to features, got %q", m.focusedPane)
	}
}

func TestHandleKeySectionToggleCollapseReturnsFocus(t *testing.T) {
	m := testModel(t)
	m, _ = press(t, m, "tab", "tab") // focus workflows (expands it)
	if m.sectionFocus != "workflows" {
		t.Fatalf("setup: focus=%q", m.sectionFocus)
	}
	m, _ = press(t, m, "2") // collapse focused section
	if m.expanded["workflows"] {
		t.Error("2 should collapse workflows")
	}
	if m.focusedPane != "features" {
		t.Errorf("collapsing the focused section should return focus to features, got %q", m.focusedPane)
	}
}

func TestHandleKeyEnterOpensDetail(t *testing.T) {
	m, _ := press(t, testModel(t), "j", "enter")
	if m.view != viewDetail {
		t.Fatalf("view = %v, want viewDetail", m.view)
	}
	if m.detail == nil || m.detail.s.Ticket != "STORY-2" {
		t.Errorf("detail should hold the row under the cursor")
	}

	m, _ = press(t, m, "esc")
	if m.view != viewDashboard {
		t.Errorf("esc should return to dashboard, got %v", m.view)
	}
}

func TestHandleKeyWorkerFileViewer(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bob.md")
	if err := os.WriteFile(path, []byte("---\nid: bob\nname: Bob\nengine: claude\n---\n\nbody"), 0644); err != nil {
		t.Fatal(err)
	}
	m := testModel(t)
	m.allWorkers = []*workers.Worker{{ID: "bob", Name: "Bob", Engine: "claude", FilePath: path}}
	m.sectionItems["workers"] = []sectionItem{{label: "Bob", path: path}}

	// shift+tab focuses last section (workers), enter opens the file viewer
	m, _ = press(t, m, "shift+tab", "enter")
	if m.view != viewFile {
		t.Fatalf("view = %v, want viewFile", m.view)
	}
	if m.viewerReturn != viewDashboard {
		t.Errorf("viewerReturn = %v, want viewDashboard", m.viewerReturn)
	}
	if m.charSheetWorker == nil || m.charSheetWorker.ID != "bob" {
		t.Fatalf("charSheetWorker not resolved: %+v", m.charSheetWorker)
	}

	// ! opens the character sheet, esc returns to the file viewer, esc again to dashboard
	m, _ = press(t, m, "!")
	if m.view != viewCharacterSheet {
		t.Fatalf("! should open character sheet, got %v", m.view)
	}
	m, _ = press(t, m, "esc")
	if m.view != viewFile {
		t.Fatalf("esc should return to file viewer, got %v", m.view)
	}
	m, _ = press(t, m, "esc")
	if m.view != viewDashboard {
		t.Errorf("esc should return to dashboard, got %v", m.view)
	}
}

func TestHandleKeyHealthDrillInOpensReport(t *testing.T) {
	m := testModel(t)
	m.healthItems = []doctor.Check{
		{Group: "workspace", Name: "AGENTS.md", Status: doctor.OK},
		{Group: "workspace", Name: "worktrees/", Status: doctor.Warning, Detail: "not created yet"},
	}

	// tab focuses the first navigable section (always health); enter drills in
	m, _ = press(t, m, "tab")
	if m.sectionFocus != "health" {
		t.Fatalf("tab should focus health, got %q", m.sectionFocus)
	}
	m, _ = press(t, m, "enter")
	if m.view != viewFile {
		t.Fatalf("view = %v, want viewFile", m.view)
	}
	if m.viewerTitle != "doctor report" {
		t.Errorf("viewerTitle = %q, want \"doctor report\"", m.viewerTitle)
	}
	if m.viewerReturn != viewDashboard {
		t.Errorf("viewerReturn = %v, want viewDashboard", m.viewerReturn)
	}
	if body := ansi.Strip(m.viewport.View()); !strings.Contains(body, "not created yet") {
		t.Errorf("report viewport missing check detail:\n%s", body)
	}

	m, _ = press(t, m, "esc")
	if m.view != viewDashboard {
		t.Errorf("esc should return to dashboard, got %v", m.view)
	}
}

func TestHandleKeyWorkflowDrillIn(t *testing.T) {
	m := testModel(t)
	m, _ = press(t, m, "tab", "tab") // focus workflows
	m, _ = press(t, m, "enter")
	if m.view != viewWorkflowDetail {
		t.Fatalf("view = %v, want viewWorkflowDetail", m.view)
	}
	if m.wfDetailName != "default" {
		t.Errorf("wfDetailName = %q, want default", m.wfDetailName)
	}

	// chain has 2 steps + 1 repair step → cursor clamps at 2
	m, _ = press(t, m, "down", "down", "down", "down")
	if m.wfDetailCursor != 2 {
		t.Errorf("wfDetailCursor = %d, want clamped at 2", m.wfDetailCursor)
	}
	m, _ = press(t, m, "up", "up", "up")
	if m.wfDetailCursor != 0 {
		t.Errorf("wfDetailCursor = %d, want clamped at 0", m.wfDetailCursor)
	}

	m, _ = press(t, m, "esc")
	if m.view != viewDashboard {
		t.Errorf("esc should return to dashboard, got %v", m.view)
	}
}

func TestHandleKeyWorkflowDetailLeftRightAliases(t *testing.T) {
	m := testModel(t)
	m, _ = press(t, m, "tab", "tab", "enter") // drill into workflows/default

	// chain has 2 steps + 1 repair step → cursor clamps at 2
	m, _ = press(t, m, "right", "l", "right")
	if m.wfDetailCursor != 2 {
		t.Errorf("wfDetailCursor = %d, want clamped at 2", m.wfDetailCursor)
	}
	m, _ = press(t, m, "left", "h", "left")
	if m.wfDetailCursor != 0 {
		t.Errorf("wfDetailCursor = %d, want clamped at 0", m.wfDetailCursor)
	}
}

func TestHandleKeyStageViewerLeftRight(t *testing.T) {
	m := testModel(t)
	m.root = t.TempDir()
	stagesDir := filepath.Join(m.root, "stages")
	if err := os.MkdirAll(stagesDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"develop", "code-review", "pr-repair"} {
		if err := os.WriteFile(filepath.Join(stagesDir, name+".md"), []byte("# "+name), 0644); err != nil {
			t.Fatal(err)
		}
	}

	m, _ = press(t, m, "tab", "tab", "enter") // drill into workflows/default
	m, _ = press(t, m, "enter")               // open stage 0 in the viewer
	if m.view != viewFile {
		t.Fatalf("view = %v, want viewFile", m.view)
	}
	if !strings.Contains(m.viewerTitle, "develop · step 1 of 3") {
		t.Fatalf("viewerTitle = %q, want develop step 1 of 3", m.viewerTitle)
	}

	// right walks pipeline order, updating cursor, title, and path
	m, _ = press(t, m, "right")
	if m.wfDetailCursor != 1 {
		t.Errorf("wfDetailCursor = %d, want 1", m.wfDetailCursor)
	}
	if !strings.Contains(m.viewerTitle, "code-review · step 2 of 3") {
		t.Errorf("viewerTitle = %q, want code-review step 2 of 3", m.viewerTitle)
	}
	if want := filepath.Join(stagesDir, "code-review.md"); m.viewerPath != want {
		t.Errorf("viewerPath = %q, want %q", m.viewerPath, want)
	}

	// continues into repair steps and clamps at the end
	m, _ = press(t, m, "l", "right")
	if m.wfDetailCursor != 2 {
		t.Errorf("wfDetailCursor = %d, want clamped at 2", m.wfDetailCursor)
	}
	if !strings.Contains(m.viewerTitle, "pr-repair · step 3 of 3") {
		t.Errorf("viewerTitle = %q, want pr-repair step 3 of 3", m.viewerTitle)
	}

	// left walks back and clamps at the start
	m, _ = press(t, m, "left", "h", "left")
	if m.wfDetailCursor != 0 {
		t.Errorf("wfDetailCursor = %d, want clamped at 0", m.wfDetailCursor)
	}

	// esc returns to the workflow detail page with the cursor where we left it
	m, _ = press(t, m, "right", "esc")
	if m.view != viewWorkflowDetail {
		t.Fatalf("esc should return to workflow detail, got %v", m.view)
	}
	if m.wfDetailCursor != 1 {
		t.Errorf("wfDetailCursor = %d after esc, want 1", m.wfDetailCursor)
	}
}

func TestHandleKeyAttachTmux(t *testing.T) {
	m := testModel(t)
	// no live session → no command
	_, cmd := press(t, m, "t")
	if cmd != nil {
		t.Error("t without a live tmux session should be a no-op")
	}
	m.features[0].s.Runtime.Tmux = &state.TmuxRuntime{Session: "story-1"}
	m.features[0].tmuxLive = true
	_, cmd = press(t, m, "t")
	if cmd == nil {
		t.Error("t with a live tmux session should return an attach command")
	}
}

func TestHandleKeyRainbowEasterEgg(t *testing.T) {
	m, cmd := press(t, testModel(t), "o", "r", "c")
	if m.rainbowStep != rainbowSteps {
		t.Errorf("rainbowStep = %d, want %d", m.rainbowStep, rainbowSteps)
	}
	if cmd == nil {
		t.Error("orc easter egg should schedule a rainbow tick")
	}
}

func TestUpdateDataMsgClampsCursor(t *testing.T) {
	m := testModel(t)
	m.cursor = 2
	tm, _ := m.Update(dataMsg{features: m.features[:1]})
	got := asModel(t, tm)
	if got.cursor != 0 {
		t.Errorf("cursor = %d, want clamped to 0 after data shrank", got.cursor)
	}
}

func TestUpdateWindowSize(t *testing.T) {
	m := testModel(t)
	tm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	got := asModel(t, tm)
	if got.width != 120 || got.height != 50 {
		t.Errorf("size = %dx%d, want 120x50", got.width, got.height)
	}
}

func TestUpdateWindowSizeReflowsFileViewer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ROUTER.md")
	long := "word " // a paragraph that must re-wrap when the width changes
	if err := os.WriteFile(path, []byte(strings.Repeat(long, 60)), 0644); err != nil {
		t.Fatal(err)
	}

	m := testModel(t)
	m.view = viewFile
	m.viewerPath = path
	m.viewport = viewport.New(m.width-4, m.height-6)
	wide, err := renderFile(path, m.width-4)
	if err != nil {
		t.Fatal(err)
	}
	m.viewport.SetContent(wide)
	wideLines := m.viewport.TotalLineCount()

	tm, _ := m.Update(tea.WindowSizeMsg{Width: 48, Height: 40})
	got := asModel(t, tm)
	if got.viewport.TotalLineCount() <= wideLines {
		t.Errorf("content lines = %d after shrinking from %d-wide render — viewer did not reflow",
			got.viewport.TotalLineCount(), wideLines)
	}
}

func TestUpdateWindowSizeReflowsWorkflowDetail(t *testing.T) {
	m := testModel(t)
	// a route chain long enough that it fits one row at width 100 but must
	// wrap onto more rows at width 60
	var steps []routeStep
	for _, n := range []string{"intake", "develop", "code-review", "qa-automation", "pr-open", "evidence"} {
		steps = append(steps, routeStep{name: n, advance: "auto"})
	}
	m.workflows = []workflowChain{{name: "default", steps: steps}}
	m.view = viewWorkflowDetail
	m.wfDetailName = "default"
	m.root = t.TempDir()
	m.viewport = viewport.New(m.width-4, m.height-6)
	m.viewport.SetContent(renderWorkflowDetail("default", m.workflows, nil, filepath.Join(m.root, "stages"), m.features, 0, m.width-4))
	wideLines := m.viewport.TotalLineCount()

	tm, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 40})
	got := asModel(t, tm)
	if got.viewport.TotalLineCount() <= wideLines {
		t.Errorf("content lines = %d after shrinking from %d — workflow detail did not reflow",
			got.viewport.TotalLineCount(), wideLines)
	}
}
