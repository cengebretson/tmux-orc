package tui

import (
	"embed"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
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

	// ── first pass: tally quest counts for derived stats ────────────
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
	maxHP := stats[4] * 5 // VIT × 5 → 15–90
	currentHP := maxHP - pausedQ*8
	if currentHP < 1 {
		currentHP = 1
	}
	ac := max(0, 10-(stats[1]-10)/2) // DEX-derived, lower = better

	// HP colour: green=full, yellow=≥50%, red=<50%
	hpStyle := green
	switch {
	case currentHP < maxHP/2:
		hpStyle = red
	case currentHP < maxHP:
		hpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(p.Yellow))
	}

	dot := styleDim.Render("  ·  ")
	rpgLine := yellow.Render(fmt.Sprintf("LVL %d", level)) +
		dot +
		hpStyle.Render(fmt.Sprintf("HP %d/%d", currentHP, maxHP)) +
		dot +
		peach.Render(fmt.Sprintf("XP %d", xp)) +
		dot +
		styleSubtext.Render(fmt.Sprintf("AC %d", ac))

	// ── right column: class, name, description, RPG line, attributes ─
	desc := workerDescription(w.FilePath)
	const barW = 14
	rightLines := []string{
		yellow.Render(class),
		mauve.Render(displayName),
	}
	if desc != "" {
		rightLines = append(rightLines, styleDim.Render(truncate(desc, 42)))
	}
	rightLines = append(rightLines, rpgLine, "")
	for i, v := range stats {
		filled := v * barW / 18
		bar := green.Render(strings.Repeat("█", filled)) + styleDim.Render(strings.Repeat("░", barW-filled))
		rightLines = append(rightLines, fmt.Sprintf("%s  %s  %2d", peach.Render(statNames[i]), bar, v))
	}

	// ── portrait + right column side by side ─────────────────────────
	var bodyLines []string
	rows := max(len(art), len(rightLines))
	for i := range rows {
		left := ""
		if i < len(art) {
			left = art[i]
		}
		right := ""
		if i < len(rightLines) {
			right = rightLines[i]
		}
		bodyLines = append(bodyLines, styleDim.Render(fmt.Sprintf("%-10s", left))+"   "+right)
	}

	// ── quest log ────────────────────────────────────────────────────
	bodyLines = append(bodyLines, "")
	bodyLines = append(bodyLines, styleDetailLabel.Render("QUEST LOG"))
	if len(quests) == 0 {
		bodyLines = append(bodyLines, styleDim.Render("  (no quests)"))
	}
	for _, q := range quests {
		f := q.f
		st := statusStyle(f.s.Status)
		slug := strings.TrimPrefix(f.s.Slug, f.s.Ticket+"-")
		bodyLines = append(bodyLines, fmt.Sprintf(
			"  %s  %-12s  %s  %s",
			st.Render(statusIcon(f.s.Status)),
			f.s.Ticket,
			styleDim.Render(truncate(slug, 24)),
			st.Render(f.s.Status),
		))
	}

	bodyLines = append(bodyLines, "")
	bodyLines = append(bodyLines, styleDim.Render("[!] or [esc] to dismiss"))

	// ── assemble box ─────────────────────────────────────────────────
	title := yellow.Render("★") + "  " + yellow.Render("BARD'S TALE") + "  " + yellow.Render("★")
	inner := title + "\n\n" + strings.Join(bodyLines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(p.Yellow)).
		Padding(1, 3).
		Render(inner)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
