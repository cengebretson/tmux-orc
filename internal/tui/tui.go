package tui

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cengebretson/orc/internal/config"
	"github.com/cengebretson/orc/internal/health"
	"github.com/cengebretson/orc/internal/state"
	"github.com/cengebretson/orc/internal/tmux"
	"github.com/cengebretson/orc/internal/workers"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

//go:embed quotes.json
var quotesJSON []byte

func pickQuote() string {
	var quotes []string
	if err := json.Unmarshal(quotesJSON, &quotes); err != nil || len(quotes) == 0 {
		return ""
	}
	return quotes[rand.Intn(len(quotes))]
}

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
	workflows     []workflowChain
	repos         []config.Repo
	sectionItems  map[string][]sectionItem
}

type routeStep struct {
	name    string
	advance string // "auto" or "manual"
}

type repairLoop struct {
	name   string
	target string // stage in main chain it loops back to
}

type workflowChain struct {
	name  string
	steps []routeStep
	loops []repairLoop
}

type sectionItem struct {
	label string
	path  string
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
	workflows     []workflowChain
	repos         []config.Repo
	expanded      map[string]bool
	cursor        int
	showArchived  bool
	lastRefresh   time.Time
	width         int
	height        int

	// section pane navigation
	focusedPane   string // "features" or "section"
	sectionFocus  string // "workflows" | "workers" | "routes"
	sectionCursor int
	sectionItems  map[string][]sectionItem

	// detail
	detail      *featureRow
	detailFiles []detailFile
	fileIdx     int

	// file viewer
	viewport      viewport.Model
	viewerTitle   string
	viewerContext string // label shown in file viewer title bar
	viewerReturn  viewState

	// search
	search    textinput.Model
	searching bool

	quote string
}

type detailFile struct {
	label string
	path  string
}

func New(root string) Model {
	ti := textinput.New()
	ti.Placeholder = "filter tickets..."
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(mauve))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(text))
	ti.Prompt = "/ "
	ti.CharLimit = 64

	return Model{
		root:         root,
		lastRefresh:  time.Now(),
		focusedPane:  "features",
		sectionItems: map[string][]sectionItem{},
		expanded: map[string]bool{
			"health":    false,
			"workflows": true,
			"workers":   false,
			"routes":    true,
		},
		search: ti,
		quote:  pickQuote(),
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
		m.workflows = msg.workflows
		m.repos = msg.repos
		m.sectionItems = msg.sectionItems
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
		// ── Search mode: route keys to textinput ─────────────────────
		if m.searching {
			switch msg.String() {
			case "esc":
				m.searching = false
				m.search.Blur()
				m.search.SetValue("")
				m.cursor = 0
				return m, nil
			case "enter":
				m.searching = false
				m.search.Blur()
				m.cursor = 0
				return m, nil
			default:
				var cmd tea.Cmd
				m.search, cmd = m.search.Update(msg)
				m.cursor = 0
				return m, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "/":
			if m.focusedPane == "features" {
				m.searching = true
				m.search.Focus()
				return m, textinput.Blink
			}

		case "tab", "shift+tab":
			navigable := m.navigableSections()
			if len(navigable) == 0 {
				break
			}
			forward := msg.String() == "tab"
			if m.focusedPane == "features" {
				if forward {
					m.focusedPane = "section"
					m.sectionFocus = navigable[0]
					m.sectionCursor = 0
					m.expanded[navigable[0]] = true
				} else {
					m.focusedPane = "section"
					last := navigable[len(navigable)-1]
					m.sectionFocus = last
					m.sectionCursor = 0
					m.expanded[last] = true
				}
			} else {
				idx := -1
				for i, k := range navigable {
					if k == m.sectionFocus {
						idx = i
						break
					}
				}
				var next int
				if forward {
					next = idx + 1
				} else {
					next = idx - 1
				}
				if next < 0 || next >= len(navigable) {
					m.focusedPane = "features"
					m.sectionFocus = ""
				} else {
					m.sectionFocus = navigable[next]
					m.sectionCursor = 0
					m.expanded[navigable[next]] = true
				}
			}

		case "esc", "b":
			if m.search.Value() != "" {
				m.search.SetValue("")
				m.cursor = 0
			} else if m.focusedPane == "section" {
				m.focusedPane = "features"
				m.sectionFocus = ""
				m.sectionCursor = 0
			}

		case "1":
			wasExpanded := m.expanded["health"]
			m.expanded["health"] = !wasExpanded
			if wasExpanded && m.sectionFocus == "health" {
				m.focusedPane = "features"
				m.sectionFocus = ""
				m.sectionCursor = 0
			}
		case "2":
			wasExpanded := m.expanded["workflows"]
			m.expanded["workflows"] = !wasExpanded
			if wasExpanded && m.sectionFocus == "workflows" {
				m.focusedPane = "features"
				m.sectionFocus = ""
				m.sectionCursor = 0
			}
		case "3":
			wasExpanded := m.expanded["workers"]
			m.expanded["workers"] = !wasExpanded
			if wasExpanded && m.sectionFocus == "workers" {
				m.focusedPane = "features"
				m.sectionFocus = ""
				m.sectionCursor = 0
			}
		case "4":
			wasExpanded := m.expanded["routes"]
			m.expanded["routes"] = !wasExpanded
			if wasExpanded && m.sectionFocus == "routes" {
				m.focusedPane = "features"
				m.sectionFocus = ""
				m.sectionCursor = 0
			}

		case "up", "k":
			if m.focusedPane == "section" {
				if m.sectionCursor > 0 {
					m.sectionCursor--
				}
			} else {
				if m.cursor > 0 {
					m.cursor--
				}
			}

		case "down", "j":
			if m.focusedPane == "section" {
				items := m.sectionItems[m.sectionFocus]
				if m.sectionCursor < len(items)-1 {
					m.sectionCursor++
				}
			} else {
				rows := m.visibleFeatures()
				if m.cursor < len(rows)-1 {
					m.cursor++
				}
			}

		case "a":
			if m.focusedPane == "features" {
				m.showArchived = !m.showArchived
				m.cursor = 0
			}

		case "t":
			if m.focusedPane == "features" {
				rows := m.visibleFeatures()
				if m.cursor < len(rows) {
					row := rows[m.cursor]
					if row.s.Runtime.Tmux != nil && row.tmuxLive {
						return m, attachTmux(row.s.Slug, row.s.Stage.Name)
					}
				}
			}

		case "enter":
			if m.focusedPane == "section" {
				items := m.sectionItems[m.sectionFocus]
				if m.sectionCursor < len(items) {
					f := items[m.sectionCursor]
					var content string
					var err error
					switch m.sectionFocus {
					case "workers":
						content, err = renderWorkerFile(f.path, m.width-4)
					case "workflows":
						content, err = renderWorkflowFile(f.path, m.width-4)
					default:
						content, err = renderFile(f.path)
					}
					if err != nil {
						content = styleHealthErr.Render("could not read: " + err.Error())
					}
					m.viewport = viewport.New(m.width-4, m.height-6)
					m.viewport.SetContent(content)
					m.viewerTitle = f.label
					m.viewerContext = sectionLabel(m.sectionFocus)
					m.viewerReturn = viewDashboard
					m.view = viewFile
				}
			} else {
				rows := m.visibleFeatures()
				if m.cursor < len(rows) {
					m.detail = rows[m.cursor]
					m.detailFiles = buildFileList(m.detail.featureDir, m.detail.s)
					m.fileIdx = 0
					m.view = viewDetail
				}
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
				return m, attachTmux(m.detail.s.Slug, m.detail.s.Stage.Name)
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
				m.viewerContext = m.detail.s.Ticket
				m.viewerReturn = viewDetail
				m.view = viewFile
			}
		}

	case viewFile:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "b":
			m.view = m.viewerReturn
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) navigableSections() []string {
	out := []string{"health"}
	for _, key := range []string{"workflows", "workers", "routes"} {
		if len(m.sectionItems[key]) > 0 {
			out = append(out, key)
		}
	}
	return out
}

func sectionLabel(key string) string {
	labels := map[string]string{
		"workflows": "Workflows",
		"workers":   "Workers",
		"routes":    "Routes",
	}
	if l, ok := labels[key]; ok {
		return l
	}
	return key
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

	// ── Column widths ────────────────────────────────────────────────
	const logoW = 30
	const rightBoxOuter = logoW + 4 // border (2) + 1-space padding each side (2)
	const logoGap = 1
	useLogo := m.width > rightBoxOuter+logoGap+44

	leftW := outerW
	if useLogo {
		leftW = outerW - rightBoxOuter - logoGap
	}
	leftInnerW := leftW - 2

	// ── Header stats ─────────────────────────────────────────────────
	since := time.Since(m.lastRefresh)
	ago := since.Round(time.Second)
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
		stalenessStyle(since).Render(fmt.Sprintf("  ·  ↺ %s ago", ago))

	// ── Left column: header + sections ───────────────────────────────
	var left strings.Builder
	left.WriteString(drawBoxLabeled(headerTitle, []string{statsLine}, leftW) + "\n")

	healthFocused := m.focusedPane == "section" && m.sectionFocus == "health"
	left.WriteString(m.sectionBox("health", "1", "Health",
		fmt.Sprintf("%d checks", len(m.healthItems)),
		m.renderHealthLines(leftInnerW-4), leftW, healthFocused) + "\n")

	wfFocused := m.focusedPane == "section" && m.sectionFocus == "workflows"
	var wfContent []string
	if wfFocused {
		wfContent = renderNavigableList(m.sectionItems["workflows"], m.sectionCursor)
	} else {
		for _, pc := range m.workflows {
			lines := renderRouteChain(pc.steps, pc.loops, leftInnerW-4)
			if pc.name != "" && pc.name != "default" {
				label := styleDim.Render(pc.name + ":")
				wfContent = append(wfContent, label)
			}
			wfContent = append(wfContent, lines...)
			wfContent = append(wfContent, "")
		}
		// trim trailing blank
		for len(wfContent) > 0 && wfContent[len(wfContent)-1] == "" {
			wfContent = wfContent[:len(wfContent)-1]
		}
	}
	left.WriteString(m.sectionBox("workflows", "2", "Workflows",
		fmt.Sprintf("%d", len(m.workflowNames)),
		wfContent, leftW, wfFocused) + "\n")

	wkFocused := m.focusedPane == "section" && m.sectionFocus == "workers"
	var wkContent []string
	if wkFocused {
		wkContent = renderNavigableList(m.sectionItems["workers"], m.sectionCursor)
	} else {
		wkContent = renderNameList(leftInnerW-4, m.workerNames)
	}
	left.WriteString(m.sectionBox("workers", "3", "Workers",
		fmt.Sprintf("%d", len(m.workerNames)),
		wkContent, leftW, wkFocused) + "\n")

	rtFocused := m.focusedPane == "section" && m.sectionFocus == "routes"
	var rtContent []string
	if rtFocused {
		rtContent = renderNavigableList(m.sectionItems["routes"], m.sectionCursor)
	} else {
		rtContent = renderRepoList(m.repos, leftInnerW-4)
	}
	left.WriteString(m.sectionBox("routes", "4", "Routes",
		fmt.Sprintf("%d repos", len(m.repos)),
		rtContent, leftW, rtFocused))

	// ── Top block: left column + right box (logo + quote) ───────────
	var topBlock string
	if useLogo {
		leftStr := left.String()
		leftHeight := lipgloss.Height(leftStr)

		logoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(surface1))
		quoteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(overlay0)).Italic(true)

		var rightLines []string
		for _, l := range strings.Split(logo, "\n") {
			rightLines = append(rightLines, " "+logoStyle.Render(l))
		}
		rightLines = append(rightLines, "")
		if m.quote != "" {
			for _, l := range strings.Split(wrapText(m.quote, logoW), "\n") {
				centered := lipgloss.PlaceHorizontal(logoW, lipgloss.Center, quoteStyle.Render(l))
				rightLines = append(rightLines, " "+centered)
			}
		}
		targetLines := leftHeight - 2
		if minContent := len(rightLines); targetLines < minContent {
			targetLines = minContent
		}
		for len(rightLines) < targetLines {
			rightLines = append(rightLines, "")
		}
		rightLines = rightLines[:targetLines]

		rightBox := drawBoxLabeledWith("", rightLines, rightBoxOuter, surface1)
		topBlock = "\n" + lipgloss.JoinHorizontal(lipgloss.Top, leftStr, strings.Repeat(" ", logoGap), rightBox) + "\n"
	} else {
		topBlock = "\n" + left.String() + "\n"
	}

	// ── Features box (full width, height-capped with scrolling) ──────
	// Remaining height = terminal - topBlock lines - help bar(1) - box borders(2) - table header(2)
	topLines := strings.Count(topBlock, "\n")
	maxDataRows := m.height - topLines - 1 - 4
	if maxDataRows < 1 {
		maxDataRows = 1
	}

	archiveToggle := styleDim.Render("  [a] show archived")
	if m.showArchived {
		archiveToggle = styleDim.Render("  [a] hide archived")
	}

	allRows := m.visibleFeatures()
	total := len(allRows)

	// Scroll window: keep cursor in view.
	offset := 0
	if m.cursor >= maxDataRows {
		offset = m.cursor - maxDataRows + 1
	}
	if offset+maxDataRows > total {
		offset = total - maxDataRows
	}
	if offset < 0 {
		offset = 0
	}
	end := offset + maxDataRows
	if end > total {
		end = total
	}
	visibleRows := allRows[offset:end]

	var featuresTitle string
	if m.searching {
		query := m.search.Value()
		matchCount := len(m.visibleFeatures())
		var matchHint string
		if query == "" {
			matchHint = styleDim.Render("  type to filter  esc cancel")
		} else {
			noun := "matches"
			if matchCount == 1 {
				noun = "match"
			}
			matchHint = styleDim.Render(fmt.Sprintf("  %d %s  esc cancel", matchCount, noun))
		}
		featuresTitle = styleSection.Render("Features") + "  " + m.search.View() + matchHint
	} else if m.search.Value() != "" {
		noun := "matches"
		if total == 1 {
			noun = "match"
		}
		featuresTitle = styleSection.Render("Features") +
			styleDim.Render("  /") + " " + styleStatusWaiting.Render(m.search.Value()) +
			styleDim.Render(fmt.Sprintf("  %d %s  esc clear", total, noun))
	} else {
		featuresTitle = styleSection.Render("Features") + archiveToggle + styleDim.Render("  [/] search")
		if total > maxDataRows {
			featuresTitle += styleDim.Render(fmt.Sprintf("  %d–%d of %d", offset+1, end, total))
		}
	}

	var tableLines []string
	if total == 0 {
		tableLines = []string{"  " + styleDim.Render("No features found. Run orc work <ticket> to start one.")}
	} else {
		tableLines = strings.Split(m.renderTable(visibleRows, outerW-2, m.cursor-offset), "\n")
	}
	featuresBorderColor := surface1
	if m.focusedPane == "features" {
		featuresBorderColor = mauve
	}

	var b strings.Builder
	b.WriteString(topBlock)
	b.WriteString(drawBoxLabeledWith(featuresTitle, tableLines, outerW, featuresBorderColor) + "\n")

	// ── Help bar ─────────────────────────────────────────────────────
	if !m.searching {
		var helpItems []string
		helpItems = append(helpItems,
			helpItem("↑↓", "navigate"),
			helpItem("enter", "open"),
			helpItem("tab", "focus sections"),
			helpItem("t", "attach"),
			helpItem("1-4", "expand/collapse"),

			helpItem("q", "quit"),
		)
		b.WriteString(styleHelp.Render(" " + strings.Join(helpItems, "  ")))
	}

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

// drawBoxLabeled renders a rounded box with the title embedded in the top border.
func drawBoxLabeled(title string, contentLines []string, outerW int) string {
	return drawBoxLabeledWith(title, contentLines, outerW, surface1)
}

// drawBoxLabeledWith is drawBoxLabeled with a configurable border color.
func drawBoxLabeledWith(title string, contentLines []string, outerW int, borderColor string) string {
	innerW := outerW - 2
	bd := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor))

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
func (m Model) sectionBox(key, keyStr, name, summary string, content []string, outerW int, focused bool) string {
	innerW := outerW - 2
	borderColor := surface1
	if focused {
		borderColor = mauve
	}
	bd := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor))
	title := styleDim.Render(keyStr) + " " + styleSection.Render(name)

	if !m.expanded[key] {
		label := " " + title
		if summary != "" {
			label += styleDim.Render("  " + summary)
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
	return drawBoxLabeledWith(title, indented, outerW, borderColor)
}

// renderNavigableList renders a list of section items with a cursor indicator.
func renderNavigableList(items []sectionItem, cursor int) []string {
	var lines []string
	for i, item := range items {
		if i == cursor {
			lines = append(lines, styleHealthOK.Render("▶")+"  "+styleSubtext.Render(item.label)+
				styleDim.Render("  enter to view"))
		} else {
			lines = append(lines, "   "+styleDim.Render(item.label))
		}
	}
	return lines
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

// renderRepoList renders repos as "name — purpose" lines.
func renderRepoList(repos []config.Repo, maxW int) []string {
	if len(repos) == 0 {
		return []string{styleDim.Render("No repos configured. Edit orc.yaml to add repos.")}
	}
	var lines []string
	for _, r := range repos {
		name := styleSubtext.Render(r.Name)
		sep := styleDivider.Render("  —  ")
		purpose := styleDim.Render(r.Purpose)
		line := name + sep + purpose
		if lipgloss.Width(line) > maxW {
			purpose = styleDim.Render(truncate(r.Purpose, maxW-lipgloss.Width(name+sep)))
			line = name + sep + purpose
		}
		lines = append(lines, line)
	}
	return lines
}

// renderRouteChain renders the workflow stage sequence with colored arrows and repair loop annotations.
func renderRouteChain(chain []routeStep, loops []repairLoop, maxW int) []string {
	if len(chain) == 0 {
		return nil
	}
	sep := styleDivider.Render("  ")
	sepW := lipgloss.Width(sep)

	// build index: workflow name → x-offset in rendered row
	chipOffsets := map[string]int{}

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
		chipOffsets[step.name] = rowW
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

	// repair loop annotations: ↺ name positioned under target chip
	if len(loops) > 0 {
		// group loops by target for layout
		type loopAnnotation struct {
			offset int
			label  string
		}
		var annotations []loopAnnotation
		for _, lp := range loops {
			offset, ok := chipOffsets[lp.target]
			if !ok {
				continue
			}
			label := styleStatusWaiting.Render("↺ ") + styleSubtext.Render(lp.name)
			annotations = append(annotations, loopAnnotation{offset: offset, label: label})
		}
		// sort by offset so we build the line left-to-right
		sort.Slice(annotations, func(i, j int) bool {
			return annotations[i].offset < annotations[j].offset
		})
		if len(annotations) > 0 {
			loopLine := ""
			loopW := 0
			for _, ann := range annotations {
				if ann.offset > loopW {
					loopLine += strings.Repeat(" ", ann.offset-loopW)
					loopW = ann.offset
				}
				connector := styleDivider.Render("└╴")
				full := connector + ann.label
				w := lipgloss.Width(full)
				loopLine += full
				loopW += w
			}
			rows = append(rows, loopLine)
		}
	}

	return rows
}

func (m Model) renderTable(rows []*featureRow, w int, selectedIdx int) string {
	const (
		wTicket   = 12
		wName     = 22
		wStatus   = 18
		wWorkflow = 16
		wWorker   = 18
	)

	header := " " +
		padRight(styleTableHeader.Render("Ticket"), wTicket) + "  " +
		padRight(styleTableHeader.Render("Name"), wName) + "  " +
		padRight(styleTableHeader.Render("Status"), wStatus) + "  " +
		padRight(styleTableHeader.Render("Workflow"), wWorkflow) + "  " +
		padRight(styleTableHeader.Render("Worker"), wWorker) + "  " +
		styleTableHeader.Render("Tmux")

	div := " " + styleDivider.Render(strings.Repeat("─", w-1))

	var lines []string
	lines = append(lines, header, div)

	for i, row := range rows {
		s := row.s
		selected := i == selectedIdx

		icon := statusIcon(s.Status)
		name := strings.TrimPrefix(s.Slug, s.Ticket+"-")

		plainWorker := row.workerName
		if plainWorker == "" {
			plainWorker = "—"
		}
		plainTmux := "-"
		if s.Runtime.Tmux != nil {
			if row.tmuxLive {
				plainTmux = "✓"
			} else {
				plainTmux = "✗"
			}
		}

		if selected {
			// Plain unstyled text so styleRowSelected background covers the full row
			line := " " +
				padRight(truncate(s.Ticket, wTicket), wTicket) + "  " +
				padRight(truncate(name, wName), wName) + "  " +
				padRight(truncate(icon+" "+s.Status, wStatus), wStatus) + "  " +
				padRight(truncate(s.Stage.Name, wWorkflow), wWorkflow) + "  " +
				padRight(truncate(plainWorker, wWorker), wWorker) + "  " +
				plainTmux
			lines = append(lines, styleRowSelected.Width(w).Render(line))
		} else {
			statusCell := statusStyle(s.Status).Render(icon + " " + s.Status)
			nameCell := styleDim.Render(truncate(name, wName))
			workerCell := styleDim.Render(truncate(plainWorker, wWorker))
			var tmuxCell string
			if s.Runtime.Tmux != nil {
				if row.tmuxLive {
					tmuxCell = styleTmuxLive.Render(plainTmux)
				} else {
					tmuxCell = styleTmuxDead.Render(plainTmux)
				}
			} else {
				tmuxCell = styleTmuxNone.Render(plainTmux)
			}
			line := " " +
				padRight(truncate(s.Ticket, wTicket), wTicket) + "  " +
				padRight(nameCell, wName) + "  " +
				padRight(statusCell, wStatus) + "  " +
				padRight(truncate(s.Stage.Name, wWorkflow), wWorkflow) + "  " +
				padRight(workerCell, wWorker) + "  " +
				tmuxCell
			lines = append(lines, line)
		}
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
		{" Workflow", s.Stage.Name},
		{" Owner   ", s.Stage.Owner},
	}
	for _, f := range fields {
		stateLines = append(stateLines, fmt.Sprintf("%s  %s",
			styleDetailLabel.Render(f.label), f.value))
	}
	if s.Runtime.Tmux != nil {
		if m.detail.tmuxLive {
			hint := styleTmuxLive.Render("tmux attach -t " + s.Runtime.Tmux.Session + ":" + s.Stage.Name)
			stateLines = append(stateLines, fmt.Sprintf("%s  %s", styleDetailLabel.Render(" Session "), hint))
		} else {
			stateLines = append(stateLines, fmt.Sprintf("%s  %s",
				styleDetailLabel.Render(" Session "),
				styleTmuxDead.Render("not running — run orc next "+s.Ticket+" to restart")))
		}
	}
	if nextName, nextAdvance := nextStageFor(m.workflows, s.Workflow, s.Stage.Name); nextName != "" {
		var nextVal string
		if nextAdvance == "auto" {
			nextVal = styleHealthOK.Render("→ "+nextName) + styleDim.Render("  auto")
		} else {
			nextVal = styleStatusWaiting.Render("→ "+nextName) + styleDim.Render("  awaiting approval")
		}
		stateLines = append(stateLines, fmt.Sprintf("%s  %s",
			styleDetailLabel.Render(" Next    "), nextVal))
	} else if s.Stage.Name != "" {
		stateLines = append(stateLines, fmt.Sprintf("%s  %s",
			styleDetailLabel.Render(" Next    "), styleDim.Render("last stage")))
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
				styleSubtext.Render(truncate(h.Stage, 20)),
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
	title := styleDetailTitle.Render(" "+m.viewerContext) +
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

// nextStageFor returns the name and advance mode of the stage after stageName
// in the named workflow chain, or empty strings if it is the last stage.
func nextStageFor(chains []workflowChain, workflowName, stageName string) (name, advance string) {
	for _, c := range chains {
		if c.name != workflowName && workflowName != "" && workflowName != "default" {
			continue
		}
		for i, step := range c.steps {
			if step.name == stageName && i+1 < len(c.steps) {
				next := c.steps[i+1]
				return next.name, next.advance
			}
		}
	}
	return "", ""
}

func loadData(root string) tea.Cmd {
	return func() tea.Msg {
		features := collectFeatures(root)
		report := health.Run(root)

		// build workflow chains from workflows.yaml
		workflowCfg, _ := config.Load(root)
		var chains []workflowChain
		allStages := map[string]bool{}
		for _, pname := range workflowCfg.Names() {
			stages := workflowCfg.StageNames(pname)
			var steps []routeStep
			inThisChain := map[string]bool{}
			for _, stageName := range stages {
				sc, _ := workflowCfg.StageConfig(pname, stageName)
				steps = append(steps, routeStep{name: stageName, advance: sc.Advance})
				inThisChain[stageName] = true
				allStages[stageName] = true
			}
			// repair loops from repair_stages section
			var loops []repairLoop
			for rname, rdef := range workflowCfg.RepairStages {
				if inThisChain[rdef.Repairs] {
					loops = append(loops, repairLoop{name: rname, target: rdef.Repairs})
				}
			}
			chains = append(chains, workflowChain{name: pname, steps: steps, loops: loops})
		}
		// fallback: flat list of all stage files
		if len(chains) == 0 {
			stagesDir := filepath.Join(root, "stages")
			entries, _ := os.ReadDir(stagesDir)
			var steps []routeStep
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
					steps = append(steps, routeStep{name: strings.TrimSuffix(e.Name(), ".md")})
				}
			}
			chains = []workflowChain{{name: "", steps: steps}}
		}
		// first chain used for single-chain references
		var chain []routeStep
		if len(chains) > 0 {
			chain = chains[0].steps
		}

		// collect all stage names for display
		var wfNames []string
		for stageName := range allStages {
			wfNames = append(wfNames, stageName)
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

		var repos []config.Repo
		if cfg, err := config.Load(root); err == nil {
			repos = cfg.Repos
		}

		// section items for navigable file viewer
		si := map[string][]sectionItem{}

		// stages: in workflow order first
		stagesDir := filepath.Join(root, "stages")
		seenStages := map[string]bool{}
		for _, step := range chain {
			p := filepath.Join(stagesDir, step.name+".md")
			if _, err := os.Stat(p); err == nil {
				si["workflows"] = append(si["workflows"], sectionItem{label: step.name, path: p})
				seenStages[step.name] = true
			}
		}
		// any remaining stage files not in the chain
		if entries, err := os.ReadDir(stagesDir); err == nil {
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
					continue
				}
				name := strings.TrimSuffix(e.Name(), ".md")
				if seenStages[name] {
					continue
				}
				si["workflows"] = append(si["workflows"], sectionItem{label: name, path: filepath.Join(stagesDir, e.Name())})
			}
		}

		// workers: actual .md files in workers/
		workersDir := filepath.Join(root, "workers")
		if entries, err := filepath.Glob(filepath.Join(workersDir, "*.md")); err == nil {
			for _, p := range entries {
				base := filepath.Base(p)
				if base == "_template.md" {
					continue
				}
				id := strings.TrimSuffix(base, ".md")
				label := id
				for _, w := range allWorkers {
					if w.ID == id && w.Name != "" {
						label = w.Name
						break
					}
				}
				si["workers"] = append(si["workers"], sectionItem{label: label, path: p})
			}
		}

		// routes: ROUTER.md as a single item
		routerPath := filepath.Join(root, "ROUTER.md")
		if _, err := os.Stat(routerPath); err == nil {
			si["routes"] = []sectionItem{{label: "ROUTER.md", path: routerPath}}
		}

		return dataMsg{
			features:      features,
			healthItems:   report.Results,
			workflowNames: wfNames,
			workerNames:   workerNames,
			workflows:     chains,
			repos:         repos,
			sectionItems:  si,
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
	query := strings.ToLower(strings.TrimSpace(m.search.Value()))
	var out []*featureRow
	for _, f := range m.features {
		if f.s.Status == "archived" && !m.showArchived {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(f.s.Ticket + " " + f.s.Slug)
			if !strings.Contains(haystack, query) {
				continue
			}
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
			workerID := s.Stage.Owner
			if workerID == "" {
				if wfCfg, _ := config.Load(root); wfCfg != nil {
					if sc, ok := wfCfg.StageConfig(s.Workflow, s.Stage.Name); ok {
						workerID = sc.Worker
					}
				}
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

// catppuccinMochaStyle is a glamour style JSON matching the Catppuccin Mocha palette.
const catppuccinMochaStyle = `{
  "document": { "block_prefix": "\n", "block_suffix": "\n", "color": "#cdd6f4", "margin": 2 },
  "block_quote": { "indent": 1, "indent_token": "│ ", "color": "#a6adc8" },
  "paragraph": {},
  "list": { "level_indent": 2 },
  "heading": { "block_suffix": "\n", "color": "#cba6f7", "bold": true },
  "h1": { "prefix": " ", "suffix": " ", "color": "#1e1e2e", "background_color": "#cba6f7", "bold": true },
  "h2": { "prefix": "## ", "color": "#cba6f7", "bold": true },
  "h3": { "prefix": "### ", "color": "#b4befe", "bold": true },
  "h4": { "prefix": "#### ", "color": "#89b4fa" },
  "h5": { "prefix": "##### ", "color": "#74c7ec" },
  "h6": { "prefix": "###### ", "color": "#6c7086" },
  "text": {},
  "strikethrough": { "crossed_out": true },
  "emph": { "italic": true },
  "strong": { "bold": true, "color": "#f5c2e7" },
  "hr": { "color": "#45475a", "format": "\n────────\n" },
  "item": { "block_prefix": "• " },
  "enumeration": { "block_prefix": ". " },
  "task": { "ticked": "[✓] ", "unticked": "[ ] " },
  "link": { "color": "#89b4fa", "underline": true },
  "link_text": { "color": "#74c7ec", "bold": true },
  "image": { "color": "#f5c2e7", "underline": true },
  "image_text": { "color": "#6c7086", "format": "Image: {{.text}} →" },
  "code": { "prefix": " ", "suffix": " ", "color": "#f38ba8", "background_color": "#313244" },
  "code_block": {
    "color": "#cdd6f4", "margin": 2,
    "chroma": {
      "text": { "color": "#cdd6f4" },
      "error": { "color": "#f38ba8", "background_color": "#313244" },
      "comment": { "color": "#6c7086" },
      "comment_preproc": { "color": "#fab387" },
      "keyword": { "color": "#cba6f7" },
      "keyword_reserved": { "color": "#cba6f7" },
      "keyword_namespace": { "color": "#f38ba8" },
      "keyword_type": { "color": "#f9e2af" },
      "operator": { "color": "#89dceb" },
      "punctuation": { "color": "#cdd6f4" },
      "name": { "color": "#cdd6f4" },
      "name_builtin": { "color": "#fab387" },
      "name_tag": { "color": "#cba6f7" },
      "name_attribute": { "color": "#89b4fa" },
      "name_class": { "color": "#f9e2af", "bold": true },
      "name_constant": { "color": "#fab387" },
      "name_decorator": { "color": "#f9e2af" },
      "name_function": { "color": "#89b4fa" },
      "literal_number": { "color": "#fab387" },
      "literal_string": { "color": "#a6e3a1" },
      "literal_string_escape": { "color": "#94e2d5" },
      "generic_deleted": { "color": "#f38ba8" },
      "generic_emph": { "italic": true },
      "generic_inserted": { "color": "#a6e3a1" },
      "generic_strong": { "bold": true },
      "generic_subheading": { "color": "#6c7086" },
      "background": { "background_color": "#1e1e2e" }
    }
  },
  "table": {},
  "definition_list": {},
  "definition_term": {},
  "definition_description": { "block_prefix": "\n🠶 " },
  "html_block": {},
  "html_span": {}
}`

func renderFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes([]byte(catppuccinMochaStyle)),
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

// renderWorkerFile renders a worker .md file as a frontmatter info box followed
// by the markdown body. width is the available viewport width.
func renderWorkerFile(path string, width int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	raw := string(data)
	body := raw

	// Split frontmatter from body.
	var w workers.Worker
	hasFM := false
	if strings.HasPrefix(strings.TrimSpace(raw), "---") {
		content := strings.TrimSpace(raw)[3:]
		if end := strings.Index(content, "\n---"); end != -1 {
			fm := strings.TrimSpace(content[:end])
			if e := yaml.Unmarshal([]byte(fm), &w); e == nil {
				hasFM = true
				rest := content[end+4:]
				body = strings.TrimSpace(rest)
			}
		}
	}

	var sb strings.Builder

	if hasFM {
		// Build info rows: label → value pairs.
		type row struct{ label, value string }
		var rows []row
		add := func(label, value string) {
			if value != "" {
				rows = append(rows, row{label, value})
			}
		}
		add("id", w.ID)
		add("product", w.Product)
		add("model", w.Model)
		add("cost tier", w.CostTier)
		add("reasoning", w.ReasoningEffort)
		add("launch mode", w.LaunchMode)
		// Measure label column width.
		labelW := 0
		for _, r := range rows {
			if len(r.label) > labelW {
				labelW = len(r.label)
			}
		}

		// Render rows as styled lines.
		innerW := width - 4
		var lines []string
		for _, r := range rows {
			pad := strings.Repeat(" ", labelW-len(r.label))
			label := styleDetailLabel.Render(r.label+pad) + styleDim.Render("  ")
			val := styleDetailValue.Render(r.value)
			lines = append(lines, "  "+label+val)
		}

		workerName := w.Name
		if workerName == "" {
			workerName = w.ID
		}
		sb.WriteString(drawBoxLabeledWith(
			styleHeader.Render(workerName),
			lines,
			innerW,
			mauve,
		))
		sb.WriteString("\n")
	}

	// Render markdown body.
	if body != "" {
		r, err := glamour.NewTermRenderer(
			glamour.WithStylesFromJSONBytes([]byte(catppuccinMochaStyle)),
			glamour.WithWordWrap(width),
		)
		if err == nil {
			if out, err := r.Render(body); err == nil {
				sb.WriteString(out)
			} else {
				sb.WriteString(body)
			}
		} else {
			sb.WriteString(body)
		}
	}

	return sb.String(), nil
}

// renderWorkflowFile renders a stage markdown file. width is the available viewport width.
func renderWorkflowFile(path string, width int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	body := strings.TrimSpace(string(data))
	if body == "" {
		return "", nil
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes([]byte(catppuccinMochaStyle)),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return body, nil
	}
	out, err := r.Render(body)
	if err != nil {
		return body, nil
	}
	return out, nil
}

// wrapText wraps s to fit within maxW columns, breaking on word boundaries.
func wrapText(s string, maxW int) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	line := words[0]
	for _, w := range words[1:] {
		if len(line)+1+len(w) <= maxW {
			line += " " + w
		} else {
			lines = append(lines, line)
			line = w
		}
	}
	lines = append(lines, line)
	return strings.Join(lines, "\n")
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
