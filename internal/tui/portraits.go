package tui

import (
	"embed"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cengebretson/orc/internal/workers"
	"github.com/charmbracelet/lipgloss"
)

//go:embed assets/portraits/*.txt
var portraitsFS embed.FS

// bardClass maps a worker to a Bard's Tale character class based on its id/name.
func bardClass(w *workers.Worker) string {
	combined := strings.ToLower(w.ID + " " + w.Name)
	switch {
	case containsAny(combined, "review", "ninja", "rogue", "shadow", "spy", "audit"):
		return "ROGUE"
	case containsAny(combined, "qa", "test", "quality", "ranger", "hunter"):
		return "RANGER"
	case containsAny(combined, "doc", "write", "spec", "bard", "scribe", "analyst"):
		return "BARD"
	case containsAny(combined, "dev", "implement", "build", "engineer", "warrior", "code", "fix"):
		return "WARRIOR"
	default:
		return "ADVENTURER"
	}
}

func containsAny(s string, terms ...string) bool {
	for _, t := range terms {
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}

// workerStats generates deterministic D&D-style stats from the worker ID.
// Returns [STR, DEX, INT, WIS, VIT] each in the range 3–18.
func workerStats(id string) [5]int {
	h := fnv.New64a()
	h.Write([]byte(id))
	v := h.Sum64()
	var out [5]int
	for i := range out {
		out[i] = int((v>>(uint(i)*11))&0xF) + 3
	}
	return out
}

// portraits holds retro ASCII art for each class, loaded from assets/portraits/.
var portraits = func() map[string][]string {
	classes := []string{"WARRIOR", "RANGER", "BARD", "ROGUE", "ADVENTURER"}
	m := make(map[string][]string, len(classes))
	for _, class := range classes {
		data, err := portraitsFS.ReadFile("assets/portraits/" + strings.ToLower(class) + ".txt")
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		m[class] = lines
	}
	return m
}()

// engineWeapon maps an engine name to a Bard's Tale-style weapon name.
func engineWeapon(engine string) string {
	switch strings.ToLower(engine) {
	case "claude":
		return "Arcane Tome"
	case "codex":
		return "Codex Blade"
	case "cursor":
		return "Cursor Staff"
	case "gemini":
		return "Gemini Orb"
	default:
		return "Unknown Relic"
	}
}

// workerDescription extracts the first non-heading paragraph from a worker .md file.
func workerDescription(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	body := string(data)
	// strip frontmatter
	if strings.HasPrefix(strings.TrimSpace(body), "---") {
		content := strings.TrimSpace(body)[3:]
		if end := strings.Index(content, "\n---"); end != -1 {
			body = strings.TrimSpace(content[end+4:])
		}
	}
	var para []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			if len(para) > 0 {
				break
			}
			continue
		}
		if line == "" {
			if len(para) > 0 {
				break
			}
			continue
		}
		para = append(para, line)
	}
	return strings.Join(para, " ")
}

// workerForPath resolves the *workers.Worker for a sectionItem path by matching
// the file path stored on each worker at load time. Falls back to a stub.
func workerForPath(path string, allWorkers []*workers.Worker) *workers.Worker {
	for _, w := range allWorkers {
		if w.FilePath == path {
			return w
		}
	}
	id := strings.TrimSuffix(filepath.Base(path), ".md")
	return &workers.Worker{ID: id, Name: id}
}

// renderCharacterSheet renders the Bard's Tale easter egg full-screen for a worker.
func renderCharacterSheet(m Model, w *workers.Worker) string {
	p := activeTheme.Palette
	class := bardClass(w)
	stats := workerStats(w.ID)
	statNames := [5]string{"STR", "DEX", "INT", "WIS", "VIT"}

	art := portraits[class]
	if art == nil {
		art = portraits["ADVENTURER"]
	}

	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color(p.Yellow)).Bold(true)
	mauve := lipgloss.NewStyle().Foreground(lipgloss.Color(p.Mauve)).Bold(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color(p.Green))
	peach := lipgloss.NewStyle().Foreground(lipgloss.Color(p.Peach))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color(p.Red))

	displayName := w.Name
	if displayName == "" {
		displayName = w.ID
	}

	// ── quest tally for derived stats ────────────────────────────────
	var activeQ, pausedQ, doneQ int
	type questEntry struct{ f *featureRow }
	var quests []questEntry
	for _, f := range m.features {
		if f.workerName != displayName && f.workerName != w.ID {
			continue
		}
		quests = append(quests, questEntry{f})
		switch f.s.Status {
		case "active":
			activeQ++
		case "paused":
			pausedQ++
		case "done", "archived":
			doneQ++
		}
	}

	// ── derived RPG stats ────────────────────────────────────────────
	xp := doneQ*500 + activeQ*100
	level := xp/300 + 1
	if level > 20 {
		level = 20
	}
	maxHP := stats[4] * 5
	currentHP := maxHP - pausedQ*8
	if currentHP < 1 {
		currentHP = 1
	}
	ac := max(0, 10-(stats[1]-10)/2)

	hpStyle := green
	switch {
	case currentHP < maxHP/2:
		hpStyle = red
	case currentHP < maxHP:
		hpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(p.Yellow))
	}

	// ── panel style shared across all three panels ───────────────────
	// outer box: Padding(1,3)=6 + border=2 → 8 overhead
	// panel:     Padding(0,2)=4 + border=2 → 6 overhead
	panelW := m.width - 14 // content width inside each panel
	if panelW < 40 {
		panelW = 40
	}
	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(p.Surface2)).
		Padding(0, 2).
		Width(panelW)

	// ── panel 1: portrait + stats ────────────────────────────────────
	dot := styleDim.Render("  ·  ")
	rpgLine := yellow.Render(fmt.Sprintf("LVL %2d", level)) +
		dot + hpStyle.Render(fmt.Sprintf("HP %d/%d", currentHP, maxHP)) +
		dot + peach.Render(fmt.Sprintf("XP %d", xp)) +
		dot + styleSubtext.Render(fmt.Sprintf("AC %d", ac))

	desc := workerDescription(w.FilePath)
	const barW = 14
	rightLines := []string{
		yellow.Render(class),
		mauve.Render(displayName),
	}
	if desc != "" {
		rightLines = append(rightLines, styleDim.Render(truncate(desc, panelW-16)))
	}
	rightLines = append(rightLines, rpgLine, "")
	for i, v := range stats {
		filled := v * barW / 18
		bar := green.Render(strings.Repeat("█", filled)) + styleDim.Render(strings.Repeat("░", barW-filled))
		rightLines = append(rightLines, fmt.Sprintf("%s  %s  %2d", peach.Render(statNames[i]), bar, v))
	}

	var statsRows []string
	for i := range max(len(art), len(rightLines)) {
		left, right := "", ""
		if i < len(art) {
			left = art[i]
		}
		if i < len(rightLines) {
			right = rightLines[i]
		}
		statsRows = append(statsRows, styleDim.Render(fmt.Sprintf("%-10s", left))+"   "+right)
	}
	statsPanel := panel.Render(strings.Join(statsRows, "\n"))

	// ── panel 2: equipment ───────────────────────────────────────────
	weapon := engineWeapon(w.Engine)
	equipLines := []string{
		styleSection.Render("Equipment"),
		"",
		fmt.Sprintf("  %s  %-20s  %s",
			peach.Render("⚔"),
			styleDetailValue.Render(weapon),
			styleDim.Render(w.Model),
		),
	}
	if len(w.Args) > 0 {
		keys := make([]string, 0, len(w.Args))
		for k := range w.Args {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			equipLines = append(equipLines, fmt.Sprintf("  %s  %s  %s",
				styleDim.Render("✦"),
				styleDetailLabel.Render(k),
				styleSubtext.Render(w.Args[k]),
			))
		}
	}
	equipPanel := panel.Render(strings.Join(equipLines, "\n"))

	// ── panel 3: quest log ───────────────────────────────────────────
	questLines := []string{styleSection.Render("Quest Log"), ""}
	if len(quests) == 0 {
		questLines = append(questLines, styleDim.Render("  (no quests)"))
	}
	for _, q := range quests {
		f := q.f
		st := statusStyle(f.s.Status)
		slug := strings.TrimPrefix(f.s.Slug, f.s.Ticket+"-")
		questLines = append(questLines, fmt.Sprintf(
			"  %s  %-12s  %s  %s",
			st.Render(statusIcon(f.s.Status)),
			f.s.Ticket,
			styleDim.Render(truncate(slug, 24)),
			st.Render(f.s.Status),
		))
	}
	questPanel := panel.Render(strings.Join(questLines, "\n"))

	// ── outer box ────────────────────────────────────────────────────
	title := yellow.Render("★") + "  " + yellow.Render("BARD'S TALE") + "  " + yellow.Render("★")
	dismiss := styleDim.Render("[!] or [esc] to dismiss")
	body := strings.Join([]string{statsPanel, equipPanel, questPanel}, "\n\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(p.Yellow)).
		Padding(1, 3).
		Render(title + "\n\n" + body + "\n\n" + dismiss)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
