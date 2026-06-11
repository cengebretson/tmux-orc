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
		if m.view == viewCharacterSheet {
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
				items := m.sectionItems[m.sectionFocus]
				if m.sectionCursor < len(items) {
					f := items[m.sectionCursor]
					var content string
					var err error
					switch m.sectionFocus {
					case "workers":
						content, err = renderWorkerFile(f.path, m.features, m.width-4)
						if err != nil {
							content = styleHealthErr.Render("could not read: " + err.Error())
						}
						m.charSheetWorker = workerForPath(f.path, m.allWorkers)
						m.viewport = viewport.New(m.width-4, m.height-6)
						m.viewport.SetContent(content)
						m.viewerTitle = f.label
						m.viewerContext = sectionLabel(m.sectionFocus)
						m.viewerReturn = viewDashboard
						m.view = viewFile
					case "workflows":
						m.wfDetailName = f.label
						m.wfDetailCursor = 0
						content = renderWorkflowDetail(f.label, m.workflows, m.allWorkers, filepath.Join(m.root, "stages"), m.features, 0, m.width-4)
						m.viewport = viewport.New(m.width-4, m.height-6)
						m.viewport.SetContent(content)
						m.view = viewWorkflowDetail
					default:
						content, err = renderFile(f.path, m.width-4)
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
				return m, attachTmux(m.detail.s.Runtime.Tmux.Session, m.detail.s.Stage.Name)
			}
		case "enter":
			if m.fileIdx < len(m.detailFiles) {
				f := m.detailFiles[m.fileIdx]
				content, err := renderFile(f.path, m.width-4)
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

	case viewWorkflowDetail:
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
				stagesDir := filepath.Join(m.root, "stages")
				content, err := renderFile(filepath.Join(stagesDir, stageName+".md"), m.width-4)
				if err != nil {
					content = styleHealthErr.Render("could not read: " + err.Error())
				}
				wfCount := stageWorkflowCount(m.workflows, stageName)
				wfWord := "workflows"
				if wfCount == 1 {
					wfWord = "workflow"
				}
				m.viewport = viewport.New(m.width-4, m.height-6)
				m.viewport.SetContent(content)
				m.viewerTitle = fmt.Sprintf("%s · step %d of %d · %s · %d %s", stageName, stepNum, total, advance, wfCount, wfWord)
				m.viewerContext = m.wfDetailName
				m.viewerReturn = viewWorkflowDetail
				m.view = viewFile
			}
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case viewFile:
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

	case viewCharacterSheet:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "!", "esc", "b":
			m.view = m.charSheetReturn
			return m, tea.ClearScreen
		}
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
