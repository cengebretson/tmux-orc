package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 6
		// viewport-backed views hold content pre-rendered at the old width;
		// rebuild it (the dashboard and detail views render from m.width live)
		switch m.view {
		case viewFile:
			m.reRenderViewerFile()
		case viewWorkflowDetail:
			m.reRenderWorkflowDetail()
		case viewCharacterSheet:
			return m, tea.ClearScreen
		}
		return m, nil

	case tickMsg:
		interval := m.refreshInterval
		if interval == 0 {
			interval = defaultRefreshInterval
		}
		return m, tea.Batch(loadData(m.root), tickEvery(interval))

	case dataMsg:
		m.features = msg.features
		m.healthItems = msg.healthItems
		m.workflowNames = msg.workflowNames
		m.workerNames = msg.workerNames
		m.allWorkers = msg.allWorkers
		m.workflows = msg.workflows
		m.repos = msg.repos
		m.sectionItems = msg.sectionItems
		m.refreshInterval = msg.refreshInterval
		if m.quote == "" {
			m.quote = pickQuote(msg.quotes)
		}
		m.lastRefresh = time.Now()
		if rows := m.visibleFeatures(); m.cursor >= len(rows) && len(rows) > 0 {
			m.cursor = len(rows) - 1
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case rainbowTickMsg:
		if m.rainbowStep > 0 {
			m.rainbowStep--
			if m.rainbowStep > 0 {
				return m, rainbowTick()
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.view {
	case viewDashboard:
		return m.handleDashboardKey(msg)
	case viewDetail:
		return m.handleDetailKey(msg)
	case viewWorkflowDetail:
		return m.handleWorkflowDetailKey(msg)
	case viewFile:
		return m.handleFileKey(msg)
	case viewCharacterSheet:
		return m.handleCharacterSheetKey(msg)
	}
	return m, nil
}

func (m Model) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	// track last 3 keys for "orc" easter egg
	m.keyBuffer[0] = m.keyBuffer[1]
	m.keyBuffer[1] = m.keyBuffer[2]
	m.keyBuffer[2] = msg.String()
	if m.keyBuffer == [3]string{"o", "r", "c"} && m.rainbowStep == 0 {
		m.rainbowStep = rainbowSteps
		return m, rainbowTick()
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "r":
		return m, loadData(m.root)
	case "/":
		if m.focusedPane == "features" {
			m.searching = true
			m.search.Focus()
			return m, textinput.Blink
		}

	case "tab", "shift+tab":
		m.cycleSectionFocus(msg.String() == "tab")

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
		m.toggleSection("health")
	case "2":
		m.toggleSection("workflows")
	case "3":
		m.toggleSection("workers")
	case "4":
		m.toggleSection("routes")

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
					return m, attachTmux(row.s.Runtime.Tmux.Session, row.s.Stage.Name)
				}
			}
		}

	case "!":
		if m.focusedPane == "section" && m.sectionFocus == "workers" {
			items := m.sectionItems["workers"]
			if m.sectionCursor < len(items) {
				m.charSheetWorker = workerForPath(items[m.sectionCursor].path, m.allWorkers)
				m.charSheetReturn = viewDashboard
				m.view = viewCharacterSheet
				return m, tea.ClearScreen
			}
		}

	case "enter":
		if m.focusedPane == "section" {
			m.openSectionItem()
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

	return m, nil
}

// cycleSectionFocus moves focus through the navigable sections with tab /
// shift+tab, wrapping back to the features pane at either end.
func (m *Model) cycleSectionFocus(forward bool) {
	navigable := m.navigableSections()
	if len(navigable) == 0 {
		return
	}
	if m.focusedPane != "section" {
		if forward {
			m.focusSection(navigable[0])
		} else {
			m.focusSection(navigable[len(navigable)-1])
		}
		return
	}
	idx := -1
	for i, k := range navigable {
		if k == m.sectionFocus {
			idx = i
			break
		}
	}
	next := idx - 1
	if forward {
		next = idx + 1
	}
	if next < 0 || next >= len(navigable) {
		m.focusedPane = "features"
		m.sectionFocus = ""
	} else {
		m.focusSection(navigable[next])
	}
}

func (m *Model) focusSection(name string) {
	m.focusedPane = "section"
	m.sectionFocus = name
	m.sectionCursor = 0
	m.expanded[name] = true
}

// toggleSection expands or collapses a dashboard section; collapsing the
// focused section returns focus to the features pane.
func (m *Model) toggleSection(name string) {
	wasExpanded := m.expanded[name]
	m.expanded[name] = !wasExpanded
	if wasExpanded && m.sectionFocus == name {
		m.focusedPane = "features"
		m.sectionFocus = ""
		m.sectionCursor = 0
	}
}

// openSectionItem opens the focused section's selected item: a worker file,
// a workflow detail page, or a plain file view.
func (m *Model) openSectionItem() {
	items := m.sectionItems[m.sectionFocus]
	if m.sectionCursor >= len(items) {
		return
	}
	f := items[m.sectionCursor]
	switch m.sectionFocus {
	case "workers":
		content, err := renderWorkerFile(f.path, m.features, m.width-4)
		if err != nil {
			content = styleHealthErr.Render("could not read: " + err.Error())
		}
		m.charSheetWorker = workerForPath(f.path, m.allWorkers)
		m.openViewer(content, f.label, sectionLabel(m.sectionFocus), viewDashboard, f.path, true)
	case "workflows":
		m.wfDetailName = f.label
		m.wfDetailCursor = 0
		content := renderWorkflowDetail(f.label, m.workflows, m.allWorkers, filepath.Join(m.root, "stages"), m.features, 0, m.width-4)
		m.viewport = viewport.New(m.width-4, m.height-6)
		m.viewport.SetContent(content)
		m.view = viewWorkflowDetail
	default:
		content, err := renderFile(f.path, m.width-4)
		if err != nil {
			content = styleHealthErr.Render("could not read: " + err.Error())
		}
		m.openViewer(content, f.label, sectionLabel(m.sectionFocus), viewDashboard, f.path, false)
	}
}

// openViewer switches to the file viewer with pre-rendered content.
func (m *Model) openViewer(content, title, context string, returnView viewState, path string, isWorker bool) {
	m.viewport = viewport.New(m.width-4, m.height-6)
	m.viewport.SetContent(content)
	m.viewerTitle = title
	m.viewerContext = context
	m.viewerReturn = returnView
	m.viewerPath = path
	m.viewerIsWorker = isWorker
	m.view = viewFile
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			return m, attachTmux(m.detail.s.Runtime.Tmux.Session, m.detail.s.Stage.Name)
		}
	case "enter":
		if m.fileIdx < len(m.detailFiles) {
			f := m.detailFiles[m.fileIdx]
			content, err := renderFile(f.path, m.width-4)
			if err != nil {
				content = styleHealthErr.Render("could not read file: " + err.Error())
			}
			m.openViewer(content, f.label, m.detail.s.Ticket, viewDetail, f.path, false)
		}
	}
	return m, nil
}

func (m Model) handleWorkflowDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "b":
		m.view = viewDashboard
	case "up", "k":
		if m.wfDetailCursor > 0 {
			m.wfDetailCursor--
			m.reRenderWorkflowDetailAndScroll()
		}
	case "down", "j":
		if m.wfDetailCursor < wfDetailTotal(m.wfDetailName, m.workflows)-1 {
			m.wfDetailCursor++
			m.reRenderWorkflowDetailAndScroll()
		}
	case "enter":
		stageName, advance, stepNum, total := wfDetailSelectedStage(m.wfDetailName, m.wfDetailCursor, m.workflows)
		if stageName != "" {
			stagePath := filepath.Join(m.root, "stages", stageName+".md")
			content, err := renderFile(stagePath, m.width-4)
			if err != nil {
				content = styleHealthErr.Render("could not read: " + err.Error())
			}
			wfCount := stageWorkflowCount(m.workflows, stageName)
			wfWord := "workflows"
			if wfCount == 1 {
				wfWord = "workflow"
			}
			title := fmt.Sprintf("%s · step %d of %d · %s · %d %s", stageName, stepNum, total, advance, wfCount, wfWord)
			m.openViewer(content, title, m.wfDetailName, viewWorkflowDetail, stagePath, false)
		}
	default:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleFileKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "b":
		if m.viewerReturn == viewWorkflowDetail {
			m.reRenderWorkflowDetail()
		}
		m.view = m.viewerReturn
	case "left", "h":
		if m.viewerReturn == viewDetail && m.fileIdx > 0 {
			m.fileIdx--
			m.loadViewerFile()
		}
	case "right", "l":
		if m.viewerReturn == viewDetail && m.fileIdx < len(m.detailFiles)-1 {
			m.fileIdx++
			m.loadViewerFile()
		}
	case "!":
		if m.charSheetWorker != nil {
			m.charSheetReturn = viewFile
			m.view = viewCharacterSheet
			return m, tea.ClearScreen
		}
	default:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleCharacterSheetKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "!", "esc", "b":
		m.view = m.charSheetReturn
		return m, tea.ClearScreen
	}
	return m, nil
}

func (m *Model) reRenderWorkflowDetail() {
	yOff := m.viewport.YOffset
	content := renderWorkflowDetail(m.wfDetailName, m.workflows, m.allWorkers, filepath.Join(m.root, "stages"), m.features, m.wfDetailCursor, m.width-4)
	m.viewport.SetContent(content)
	m.viewport.SetYOffset(yOff)
}

// reRenderWorkflowDetailAndScroll re-renders the workflow detail and scrolls the
// viewport so the selected cursor row stays visible.
func (m *Model) reRenderWorkflowDetailAndScroll() {
	content := renderWorkflowDetail(m.wfDetailName, m.workflows, m.allWorkers, filepath.Join(m.root, "stages"), m.features, m.wfDetailCursor, m.width-4)
	m.viewport.SetContent(content)
	// Stage rows begin after the route box (~8 lines) and the table header+divider (2 lines).
	const headerLines = 10
	targetLine := headerLines + m.wfDetailCursor
	viewH := m.viewport.Height
	curY := m.viewport.YOffset
	if targetLine < curY {
		m.viewport.SetYOffset(targetLine)
	} else if targetLine >= curY+viewH {
		m.viewport.SetYOffset(targetLine - viewH + 1)
	}
}

// loadViewerFile loads m.detailFiles[m.fileIdx] into the viewport for viewFile.
func (m *Model) loadViewerFile() {
	f := m.detailFiles[m.fileIdx]
	content, err := renderFile(f.path, m.viewport.Width)
	if err != nil {
		content = styleHealthErr.Render("could not read file: " + err.Error())
	}
	m.viewport.SetContent(content)
	m.viewport.SetYOffset(0)
	m.viewerTitle = f.label
	m.viewerPath = f.path
}

// reRenderViewerFile rebuilds the file viewer content at the current viewport
// width, preserving the scroll position. Called on window resize.
func (m *Model) reRenderViewerFile() {
	if m.viewerPath == "" {
		return
	}
	var content string
	var err error
	if m.viewerIsWorker {
		content, err = renderWorkerFile(m.viewerPath, m.features, m.viewport.Width)
	} else {
		content, err = renderFile(m.viewerPath, m.viewport.Width)
	}
	if err != nil {
		content = styleHealthErr.Render("could not read: " + err.Error())
	}
	yOff := m.viewport.YOffset
	m.viewport.SetContent(content)
	m.viewport.SetYOffset(yOff)
}

// ── Commands ──────────────────────────────────────────────────────

func tickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
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
