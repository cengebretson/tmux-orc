package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/cengebretson/orc/internal/health"
	"github.com/cengebretson/orc/internal/state"
	"github.com/cengebretson/orc/internal/tmux"
	"github.com/cengebretson/orc/internal/workers"
	"github.com/cengebretson/orc/internal/workflow"
)

// ── view states ──────────────────────────────────────────────────

type viewState int

const (
	viewDashboard viewState = iota
	viewDetail
	viewFile
)

// ── messages ─────────────────────────────────────────────────────

type tickMsg time.Time
type dataMsg struct {
	features      []*featureRow
	healthItems   []health.Result
	workflowNames []string
	workerNames   []string
	routeChain    []routeStep
	repos         []repoEntry
}

type routeStep struct {
	name    string
	advance string // "auto" or "manual"
}

type repoEntry struct {
	name    string
	purpose string
}

// ── data types ───────────────────────────────────────────────────

type featureRow struct {
	s          *state.State
	featureDir string
	workerName string
	tmuxLive   bool
}

// ── model ─────────────────────────────────────────────────────────

type Model struct {
	root          string
	view          viewState
	features      []*featureRow
	healthItems   []health.Result
	workflowNames []string
	workerNames   []string
	routeChain    []routeStep
	repos         []repoEntry
	expanded      map[string]bool
	cursor        int
	showArchived  bool
	lastRefresh   time.Time
	width         int
	height        int

	// detail
	detail      *featureRow
	detailFiles []detailFile
	fileIdx     int

	// file viewer
	viewport    viewport.Model
	viewerTitle string

	err error
}

type detailFile struct {
	label string
	path  string
}

func New(root string) Model {
	return Model{
		root:        root,
		lastRefresh: time.Now(),
		expanded: map[string]bool{
			"health":    false,
			"workflows": false,
			"workers":   false,
			"routes":    false,
		},
	}
}

func Run(root string) error {
	m := New(root)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// ── Init ─────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadData(m.root), tickEvery(5*time.Second))
}

// ── Update ───────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 6
		return m, nil

	case tickMsg:
		return m, tea.Batch(loadData(m.root), tickEvery(5*time.Second))

	case dataMsg:
		m.features = msg.features
		m.healthItems = msg.healthItems
		m.workflowNames = msg.workflowNames
		m.workerNames = msg.workerNames
		m.routeChain = msg.routeChain
		m.repos = msg.repos
		m.lastRefresh = time.Now()
		if rows := m.visibleFeatures(); m.cursor >= len(rows) && len(rows) > 0 {
			m.cursor = len(rows) - 1
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.view {

	case viewDashboard:
		rows := m.visibleFeatures()
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(rows)-1 {
				m.cursor++
			}
		case "a":
			m.showArchived = !m.showArchived
			m.cursor = 0
		case "1":
			m.expanded["health"] = !m.expanded["health"]
		case "2":
			m.expanded["workflows"] = !m.expanded["workflows"]
		case "3":
			m.expanded["workers"] = !m.expanded["workers"]
		case "4":
			m.expanded["routes"] = !m.expanded["routes"]
		case "r":
			return m, loadData(m.root)
		case "t":
			if m.cursor < len(rows) {
				row := rows[m.cursor]
				if row.s.Runtime.Tmux != nil && row.tmuxLive {
					return m, attachTmux(row.s.Slug, row.s.Stage.Workflow)
				}
			}
		case "enter":
			if m.cursor < len(rows) {
				m.detail = rows[m.cursor]
				m.detailFiles = buildFileList(m.detail.featureDir, m.detail.s)
				m.fileIdx = 0
				m.view = viewDetail
			}
		}

	case viewDetail:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "b":
			m.view = viewDashboard
		case "tab", "right", "l":
			if m.fileIdx < len(m.detailFiles)-1 {
				m.fileIdx++
			}
		case "shift+tab", "left", "h":
			if m.fileIdx > 0 {
				m.fileIdx--
			}
		case "t":
			if m.detail.s.Runtime.Tmux != nil && m.detail.tmuxLive {
				return m, attachTmux(m.detail.s.Slug, m.detail.s.Stage.Workflow)
			}
		case "enter":
			if m.fileIdx < len(m.detailFiles) {
				f := m.detailFiles[m.fileIdx]
				content, err := renderFile(f.path)
				if err != nil {
					content = styleHealthErr.Render("could not read file: " + err.Error())
				}
				m.viewport = viewport.New(m.width-4, m.height-6)
				m.viewport.SetContent(content)
				m.viewerTitle = f.label
				m.view = viewFile
			}
		}

	case viewFile:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "b":
			m.view = viewDetail
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// ── View ─────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	switch m.view {
	case viewDetail:
		return m.viewDetail()
	case viewFile:
		return m.viewFile()
	default:
		return m.viewDashboard()
	}
}

// ── Dashboard view ────────────────────────────────────────────────

func (m Model) viewDashboard() string {
	outerW := m.width - 2
	innerW := outerW - 2

	var b strings.Builder

	// ── Header: compact labeled box ─────────────────────────────────
	ago := time.Since(m.lastRefresh).Round(time.Second)
	active, blocked := 0, 0
	for _, f := range m.features {
		switch f.s.Status {
		case "in_progress", "ready", "waiting_for_human":
			active++
		case "blocked":
			blocked++
		}
	}

	headerTitle := styleHeader.Render("orc") + styleDim.Render("  workspace orchestrator")
	statsLine := "  " +
		styleSubtext.Render(fmt.Sprintf("%d features", len(m.features))) +
		styleDim.Render("  ·  ") +
		styleHealthOK.Render(fmt.Sprintf("%d active", active)) +
		styleDim.Render("  ·  ") +
		styleStatusBlocked.Render(fmt.Sprintf("%d blocked", blocked)) +
		styleDim.Render(fmt.Sprintf("  ·  ↺ %s ago", ago))
	b.WriteString("\n" + drawBoxLabeled(headerTitle, []string{statsLine}, outerW) + "\n")

	// ── Collapsible section boxes ─────────────────────────────────────
	b.WriteString(m.sectionBox("health", "1", "Health",
		fmt.Sprintf("%d checks", len(m.healthItems)),
		m.renderHealthLines(innerW-4), outerW) + "\n")

	b.WriteString(m.sectionBox("workflows", "2", "Workflows",
		fmt.Sprintf("%d", len(m.workflowNames)),
		renderNameList(innerW-4, m.workflowNames), outerW) + "\n")

	b.WriteString(m.sectionBox("workers", "3", "Workers",
		fmt.Sprintf("%d", len(m.workerNames)),
		renderNameList(innerW-4, m.workerNames), outerW) + "\n")

	b.WriteString(m.sectionBox("routes", "4", "Routes",
		fmt.Sprintf("%d repos", len(m.repos)),
		renderRepoList(m.repos, innerW-4), outerW) + "\n")

	// ── Features box ─────────────────────────────────────────────────
	archiveToggle := styleDim.Render("  [a] show archived")
	if m.showArchived {
		archiveToggle = styleDim.Render("  [a] hide archived")
	}
	featuresTitle := styleSection.Render("Features") + archiveToggle

	rows := m.visibleFeatures()
	var tableLines []string
	if len(rows) == 0 {
		tableLines = []string{"  " + styleDim.Render("No features found. Run orc work <ticket> to start one.")}
	} else {
		tableLines = strings.Split(m.renderTable(rows, innerW), "\n")
	}
	b.WriteString(drawBoxLabeled(featuresTitle, tableLines, outerW) + "\n")

	// ── Help bar ──────────────────────────────────────────────────────
	help := strings.Join([]string{
		helpItem("↑↓", "navigate"),
		helpItem("enter", "detail"),
		helpItem("t", "attach"),
		helpItem("a", "archived"),
		helpItem("1-4", "expand/collapse"),
		helpItem("r", "refresh"),
		helpItem("q", "quit"),
	}, "  ")
	b.WriteString(styleHelp.Render(" " + help))

	return b.String()
}

// drawBox renders a plain rounded box (no title in border).
func drawBox(title string, contentLines []string, outerW int) string {
	innerW := outerW - 2

	var all []string
	if title != "" {
		all = append(all, title)
		if len(contentLines) > 0 {
			all = append(all, styleDivider.Render(strings.Repeat("─", innerW)))
		}
	}
	all = append(all, contentLines...)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(surface1)).
		Width(innerW).
		Render(strings.Join(all, "\n"))
}

// drawBoxLabeled renders a rounded box with the title embedded in the top border,
// like: ╭─ Title ──────────╮
func drawBoxLabeled(title string, contentLines []string, outerW int) string {
	innerW := outerW - 2
	bd := lipgloss.NewStyle().Foreground(lipgloss.Color(surface1))

	label := " " + title + " "
	labelW := lipgloss.Width(label)
	dashRight := innerW - 1 - labelW
	if dashRight < 0 {
		dashRight = 0
	}

	top := bd.Render("╭─") + label + bd.Render(strings.Repeat("─", dashRight)+"╮")
	bot := bd.Render("╰" + strings.Repeat("─", innerW) + "╯")

	var lines []string
	lines = append(lines, top)
	for _, cl := range contentLines {
		clW := lipgloss.Width(cl)
		pad := innerW - clW
		if pad < 0 {
			pad = 0
		}
		lines = append(lines, bd.Render("│")+cl+strings.Repeat(" ", pad)+bd.Render("│"))
	}
	lines = append(lines, bot)
	return strings.Join(lines, "\n")
}

// padRight pads s to at least width visible characters, using lipgloss.Width
// to measure so ANSI escape codes don't throw off the count.
func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// renderHealthLines wraps health items into rows fitting within maxW.
func (m Model) renderHealthLines(maxW int) []string {
	var parts []string
	for _, item := range m.healthItems {
		var s lipgloss.Style
		switch item.Status {
		case health.OK:
			s = styleHealthOK
		case health.Empty:
			s = styleHealthWarn
		default:
			s = styleHealthErr
		}
		icon := "✓"
		if item.Status != health.OK {
			icon = "✗"
		}
		parts = append(parts, s.Render(icon+" "+strings.TrimSpace(item.Name)))
	}
	sep := styleDivider.Render("  ·  ")
	sepW := lipgloss.Width(sep)

	var rows []string
	row := ""
	rowW := 0
	for _, p := range parts {
		pW := lipgloss.Width(p)
		if rowW > 0 && rowW+sepW+pW > maxW {
			rows = append(rows, row)
			row = ""
			rowW = 0
		}
		if rowW > 0 {
			row += sep
			rowW += sepW
		}
		row += p
		rowW += pW
	}
	if row != "" {
		rows = append(rows, row)
	}
	return rows
}

// sectionBox renders a collapsible labeled box.
// Collapsed: just the top+bottom border with title and summary in the border line.
// Expanded: full box with content.
func (m Model) sectionBox(key, keyStr, name, summary string, content []string, outerW int) string {
	innerW := outerW - 2
	bd := lipgloss.NewStyle().Foreground(lipgloss.Color(surface1))
	title := styleDim.Render(keyStr) + " " + styleSection.Render(name)

	if !m.expanded[key] {
		label := " " + title
		if summary != "" {
			label += styleDim.Render("  "+summary)
		}
		label += " "
		labelW := lipgloss.Width(label)
		dashRight := innerW - 1 - labelW
		if dashRight < 0 {
			dashRight = 0
		}
		top := bd.Render("╭─") + label + bd.Render(strings.Repeat("─", dashRight)+"╮")
		bot := bd.Render("╰" + strings.Repeat("─", innerW) + "╯")
		return strings.Join([]string{top, bot}, "\n")
	}

	var indented []string
	for _, l := range content {
		indented = append(indented, "  "+l)
	}
	return drawBoxLabeled(title, indented, outerW)
}

// renderNameList wraps a list of names with · separators to fit maxW.
func renderNameList(maxW int, names []string) []string {
	sep := styleDivider.Render("  ·  ")
	sepW := lipgloss.Width(sep)

	var rows []string
	row := ""
	rowW := 0
	for _, name := range names {
		chip := styleSubtext.Render(name)
		chipW := lipgloss.Width(chip)
		if rowW > 0 && rowW+sepW+chipW > maxW {
			rows = append(rows, row)
			row = ""
			rowW = 0
		}
		if rowW > 0 {
			row += sep
			rowW += sepW
		}
		row += chip
		rowW += chipW
	}
	if row != "" {
		rows = append(rows, row)
	}
	return rows
}

// parseRouterRepos reads ROUTER.md and extracts the ## Repos table.
// Rows where the name starts/ends with _ (placeholder rows) are skipped.
func parseRouterRepos(path string) []repoEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	inRepos := false
	var repos []repoEntry
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## Repos") {
			inRepos = true
			continue
		}
		if inRepos && strings.HasPrefix(trimmed, "## ") {
			break
		}
		if !inRepos || !strings.HasPrefix(trimmed, "|") {
			continue
		}
		// skip header and separator rows
		if strings.Contains(trimmed, "---") || strings.Contains(trimmed, "Name") {
			continue
		}
		cols := strings.Split(trimmed, "|")
		if len(cols) < 4 {
			continue
		}
		name := strings.Trim(strings.TrimSpace(cols[1]), "_*` ")
		purpose := strings.Trim(strings.TrimSpace(cols[3]), "_*` ")
		if name == "" || strings.HasPrefix(cols[1], " _") {
			continue
		}
		repos = append(repos, repoEntry{name: name, purpose: purpose})
	}
	return repos
}

// renderRepoList renders repos as "name — purpose" lines.
func renderRepoList(repos []repoEntry, maxW int) []string {
	if len(repos) == 0 {
		return []string{styleDim.Render("No repos configured. Run SETUP.md to populate ROUTER.md.")}
	}
	var lines []string
	for _, r := range repos {
		name := styleSubtext.Render(r.name)
		sep := styleDivider.Render("  —  ")
		purpose := styleDim.Render(r.purpose)
		line := name + sep + purpose
		if lipgloss.Width(line) > maxW {
			purpose = styleDim.Render(truncate(r.purpose, maxW-lipgloss.Width(name+sep)))
			line = name + sep + purpose
		}
		lines = append(lines, line)
	}
	return lines
}

// renderRouteChain renders the workflow pipeline with colored arrows.
func renderRouteChain(chain []routeStep, maxW int) []string {
	if len(chain) == 0 {
		return nil
	}
	sep := styleDivider.Render("  ")
	sepW := lipgloss.Width(sep)

	var rows []string
	row := ""
	rowW := 0
	for i, step := range chain {
		chip := styleSubtext.Render(step.name)
		chipW := lipgloss.Width(chip)

		var arrow string
		var arrowW int
		if i < len(chain)-1 {
			if chain[i].advance == "manual" {
				arrow = sep + styleStatusWaiting.Render("→") + sep
			} else {
				arrow = sep + styleHealthOK.Render("→") + sep
			}
			arrowW = sepW*2 + 1
		}

		needed := chipW + arrowW
		if rowW > 0 && rowW+needed > maxW {
			rows = append(rows, row)
			row = ""
			rowW = 0
		}
		row += chip
		rowW += chipW
		if arrow != "" {
			row += arrow
			rowW += arrowW
		}
	}
	if row != "" {
		rows = append(rows, row)
	}
	return rows
}

func (m Model) renderTable(rows []*featureRow, w int) string {
	const (
		wTicket   = 16
		wStatus   = 18
		wWorkflow = 20
		wWorker   = 20
	)

	header := " " +
		padRight(styleTableHeader.Render("Ticket"), wTicket) + "  " +
		padRight(styleTableHeader.Render("Status"), wStatus) + "  " +
		padRight(styleTableHeader.Render("Workflow"), wWorkflow) + "  " +
		padRight(styleTableHeader.Render("Worker"), wWorker) + "  " +
		styleTableHeader.Render("Tmux")

	div := " " + styleDivider.Render(strings.Repeat("─", w-1))

	var lines []string
	lines = append(lines, header, div)

	for i, row := range rows {
		s := row.s
		selected := i == m.cursor

		icon := statusIcon(s.Status)
		statusCell := statusStyle(s.Status).Render(icon + " " + s.Status)

		var tmuxCell string
		if s.Runtime.Tmux != nil {
			if row.tmuxLive {
				tmuxCell = styleTmuxLive.Render("✓")
			} else {
				tmuxCell = styleTmuxDead.Render("✗")
			}
		} else {
			tmuxCell = styleTmuxNone.Render("-")
		}

		worker := row.workerName
		if worker == "" {
			worker = styleDim.Render("—")
		}

		line := " " +
			padRight(truncate(s.Ticket, wTicket), wTicket) + "  " +
			padRight(statusCell, wStatus) + "  " +
			padRight(truncate(s.Stage.Workflow, wWorkflow), wWorkflow) + "  " +
			padRight(truncate(worker, wWorker), wWorker) + "  " +
			tmuxCell

		if selected {
			line = styleRowSelected.Width(w).Render(line)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// ── Detail view ───────────────────────────────────────────────────

func (m Model) viewDetail() string {
	if m.detail == nil {
		return ""
	}
	s := m.detail.s
	outerW := m.width - 2
	innerW := outerW - 2
	var b strings.Builder

	b.WriteString("\n" + drawBox(styleDetailTitle.Render(" "+s.Slug+" "), nil, outerW) + "\n")

	// State fields
	var stateLines []string
	fields := []struct{ label, value string }{
		{" Ticket  ", s.Ticket},
		{" Status  ", statusStyle(s.Status).Render(statusIcon(s.Status) + " " + s.Status)},
		{" Workflow", s.Stage.Workflow},
		{" Owner   ", s.Stage.Owner},
	}
	for _, f := range fields {
		stateLines = append(stateLines, fmt.Sprintf("%s  %s",
			styleDetailLabel.Render(f.label), f.value))
	}
	if s.Runtime.Tmux != nil {
		if m.detail.tmuxLive {
			hint := styleTmuxLive.Render("tmux attach -t " + s.Runtime.Tmux.Session + ":" + s.Stage.Workflow)
			stateLines = append(stateLines, fmt.Sprintf("%s  %s", styleDetailLabel.Render(" Session "), hint))
		} else {
			stateLines = append(stateLines, fmt.Sprintf("%s  %s",
				styleDetailLabel.Render(" Session "),
				styleTmuxDead.Render("not running — run orc next "+s.Ticket+" to restart")))
		}
	}
	if s.NextAction.Prompt != "" {
		prompt := s.NextAction.Prompt
		if len(prompt) > innerW-20 {
			prompt = prompt[:innerW-23] + "…"
		}
		stateLines = append(stateLines, fmt.Sprintf("%s  %s",
			styleDetailLabel.Render(" Next    "), styleSubtext.Render(prompt)))
	}
	b.WriteString(drawBox(styleSection.Render(" State "), stateLines, outerW) + "\n")

	// History
	if len(s.History) > 0 {
		var histLines []string
		for _, h := range s.History {
			ts := h.At
			if len(ts) > 10 {
				ts = ts[:10]
			}
			histLines = append(histLines, fmt.Sprintf(" %s  %-20s  %-18s  %s",
				styleDim.Render(ts),
				styleSubtext.Render(truncate(h.Workflow, 20)),
				styleDim.Render(truncate(h.Owner, 18)),
				styleSubtext.Render(truncate(h.Result, innerW-72)),
			))
		}
		b.WriteString(drawBox(styleSection.Render(" History "), histLines, outerW) + "\n")
	}

	// Files
	if len(m.detailFiles) > 0 {
		chips := " "
		for i, f := range m.detailFiles {
			exists := fileExists(f.path)
			var chip string
			if i == m.fileIdx {
				chip = styleFileSelected.Render(f.label)
			} else if exists {
				chip = styleFileOK.Render(f.label)
			} else {
				chip = styleFileMissing.Render(f.label)
			}
			chips += chip + " "
		}
		fileLines := []string{chips}
		if m.fileIdx < len(m.detailFiles) {
			f := m.detailFiles[m.fileIdx]
			if fileExists(f.path) {
				fileLines = append(fileLines, " "+styleDim.Render("enter to view "+f.label))
			} else {
				fileLines = append(fileLines, " "+styleDim.Render(f.label+" does not exist yet"))
			}
		}
		b.WriteString(drawBox(styleSection.Render(" Files "), fileLines, outerW) + "\n")
	}

	help := strings.Join([]string{
		helpItem("tab/←→", "cycle files"),
		helpItem("enter", "view file"),
		helpItem("t", "attach"),
		helpItem("esc", "back"),
		helpItem("q", "quit"),
	}, "  ")
	b.WriteString(styleHelp.Render(" " + help))

	return b.String()
}

// ── File viewer ───────────────────────────────────────────────────

func (m Model) viewFile() string {
	outerW := m.width - 2
	var b strings.Builder
	title := styleDetailTitle.Render(" "+m.detail.s.Ticket) +
		styleDim.Render(" · ") +
		styleSubtext.Render(m.viewerTitle+" ")
	b.WriteString("\n" + drawBox(title, nil, outerW) + "\n")
	b.WriteString(m.viewport.View())
	help := strings.Join([]string{
		helpItem("↑↓/pgup/pgdn", "scroll"),
		helpItem("esc", "back"),
		helpItem("q", "quit"),
	}, "  ")
	b.WriteString("\n" + styleHelp.Render("  "+help))
	return b.String()
}

// ── Commands ──────────────────────────────────────────────────────

func tickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func loadData(root string) tea.Cmd {
	return func() tea.Msg {
		features := collectFeatures(root)
		report := health.Run(root)

		wfDir := filepath.Join(root, "workflows")

		// workflow names (all dirs)
		var wfNames []string
		if entries, err := os.ReadDir(wfDir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					wfNames = append(wfNames, e.Name())
				}
			}
		}

		// route chain — follow next_workflow links from intake
		var chain []routeStep
		seen := map[string]bool{}
		cur := "intake"
		for cur != "" && !seen[cur] {
			seen[cur] = true
			cfg, _ := workflow.Load(wfDir, cur)
			advance := ""
			if cfg != nil {
				advance = cfg.Advance
			}
			chain = append(chain, routeStep{name: cur, advance: advance})
			if cfg == nil {
				break
			}
			cur = cfg.NextWorkflow
		}

		// worker names
		allWorkers, _ := workers.Load(filepath.Join(root, "workers"))
		var workerNames []string
		for _, w := range allWorkers {
			name := w.Name
			if name == "" {
				name = w.ID
			}
			workerNames = append(workerNames, name)
		}

		repos := parseRouterRepos(filepath.Join(root, "ROUTER.md"))

		return dataMsg{
			features:      features,
			healthItems:   report.Results,
			workflowNames: wfNames,
			workerNames:   workerNames,
			routeChain:    chain,
			repos:         repos,
		}
	}
}

func attachTmux(session, window string) tea.Cmd {
	return tea.ExecProcess(
		newTmuxCmd(session, window),
		func(err error) tea.Msg { return nil },
	)
}

func newTmuxCmd(session, window string) *exec.Cmd {
	target := session + ":" + window
	if os.Getenv("TMUX") != "" {
		return exec.Command("tmux", "switch-client", "-t", target)
	}
	return exec.Command("tmux", "attach-session", "-t", target)
}

// ── Helpers ───────────────────────────────────────────────────────

func (m Model) visibleFeatures() []*featureRow {
	var out []*featureRow
	for _, f := range m.features {
		if f.s.Status == "archived" && !m.showArchived {
			continue
		}
		out = append(out, f)
	}
	return out
}

func collectFeatures(root string) []*featureRow {
	featuresDir := filepath.Join(root, "features")
	allWorkers, _ := workers.Load(filepath.Join(root, "workers"))
	activeSessions := make(map[string]bool)
	if tmux.Available() {
		for _, name := range tmux.ListSessions() {
			activeSessions[name] = true
		}
	}

	var rows []*featureRow

	collect := func(dir string) {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if !e.IsDir() || e.Name() == "_template" {
				continue
			}
			featureDir := filepath.Join(dir, e.Name())
			s, err := state.Load(featureDir)
			if err != nil {
				continue
			}
			workerName := ""
			wfCfg, _ := workflow.Load(filepath.Join(root, "workflows"), s.Stage.Workflow)
			workerID := s.Stage.Owner
			if workerID == "" && wfCfg != nil {
				workerID = wfCfg.Worker
			}
			if workerID != "" {
				if w := workers.FindByID(allWorkers, workerID); w != nil {
					workerName = w.Name
				} else {
					workerName = workerID
				}
			}
			live := s.Runtime.Tmux != nil && activeSessions[s.Runtime.Tmux.Session]
			rows = append(rows, &featureRow{
				s:          s,
				featureDir: featureDir,
				workerName: workerName,
				tmuxLive:   live,
			})
		}
	}

	collect(featuresDir)
	collect(filepath.Join(featuresDir, "_archive"))
	return rows
}

func buildFileList(featureDir string, s *state.State) []detailFile {
	candidates := []detailFile{
		{"TICKET.md", filepath.Join(featureDir, "TICKET.md")},
		{"SPEC.md", filepath.Join(featureDir, "SPEC.md")},
		{"PLAN.md", filepath.Join(featureDir, "PLAN.md")},
		{"WORKLOG.md", filepath.Join(featureDir, "WORKLOG.md")},
		{"DECISIONS.md", filepath.Join(featureDir, "DECISIONS.md")},
		{"impl/QA_HANDOFF.md", filepath.Join(featureDir, "impl", "QA_HANDOFF.md")},
		{"impl/REVIEW.md", filepath.Join(featureDir, "impl", "REVIEW.md")},
		{"impl/PR.md", filepath.Join(featureDir, "impl", "PR.md")},
		{"qa/QA_PLAN.md", filepath.Join(featureDir, "qa", "QA_PLAN.md")},
		{"qa/QA_RESULT.md", filepath.Join(featureDir, "qa", "QA_RESULT.md")},
	}
	core := map[string]bool{"TICKET.md": true, "SPEC.md": true, "PLAN.md": true}
	var out []detailFile
	for _, f := range candidates {
		if fileExists(f.path) || core[f.label] {
			out = append(out, f)
		}
	}
	return out
}

func renderFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		return string(data), nil
	}
	out, err := r.Render(string(data))
	if err != nil {
		return string(data), nil
	}
	return out, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
