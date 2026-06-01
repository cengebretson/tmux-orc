package tui

import (
	"fmt"
	"hash/fnv"
	"path/filepath"
	"strings"

	"github.com/cengebretson/orc/internal/workers"
	"github.com/charmbracelet/lipgloss"
)

// bardClass returns the Bard's Tale character class for a worker.
// If bards_tale_class is set in the worker's frontmatter, that wins.
// Otherwise it falls back to a heuristic based on the worker's id/name.
func bardClass(w *workers.Worker) string {
	if c := strings.ToUpper(strings.TrimSpace(w.BardsTale.Class)); c != "" {
		return c
	}
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

// portraits holds retro ASCII art for each class. Each string is exactly 10 chars wide.
var portraits = map[string][]string{
	"WARRIOR": {
		`  _____  `,
		` /o   o\ `,
		`| |___| |`,
		` \_____/ `,
		`  /|||\ `,
		` / ||| \ `,
		`  /   \  `,
		` [=====] `,
		`  |   |  `,
		`  |___|  `,
	},
	"RANGER": {
		`   /\    `,
		`  /  \   `,
		` | oo |  `,
		`  \__/   `,
		`  (||)   `,
		` / || \  `,
		`  /  \   `,
		` | /\ |  `,
		` |/  \|  `,
		`         `,
	},
	"BARD": {
		`  _~_~_  `,
		` /o . o\ `,
		`|  \_/  |`,
		` \  |  / `,
		`  | | |  `,
		` /|_|_|\ `,
		`  |   |  `,
		` /|   |\ `,
		`  ~   ~  `,
		`  ♪   ♫  `,
	},
	"ROGUE": {
		`  _____  `,
		` / ### \ `,
		`| ^   ^ |`,
		` \  -  / `,
		`  \___/  `,
		`  /   \  `,
		` /  |  \ `,
		`  / | \  `,
		` /  |  \ `,
		`   / \   `,
	},
	"ADVENTURER": {
		`    O    `,
		`   /|\   `,
		`  / | \  `,
		`    |    `,
		`   / \   `,
		`  /   \  `,
		`         `,
		`         `,
		`         `,
		`         `,
	},
}

// workerForPath resolves the *workers.Worker for a sectionItem path.
// Falls back to a stub with ID derived from the filename.
func workerForPath(path string, allWorkers []*workers.Worker) *workers.Worker {
	id := strings.TrimSuffix(filepath.Base(path), ".md")
	for _, w := range allWorkers {
		if w.ID == id {
			return w
		}
	}
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

	displayName := w.Name
	if displayName == "" {
		displayName = w.ID
	}

	// ── right column: class name, stats ─────────────────────────────
	const barW = 14
	rightLines := []string{
		yellow.Render(class),
		mauve.Render(displayName),
		"",
	}
	for i, v := range stats {
		filled := v * barW / 18
		bar := green.Render(strings.Repeat("█", filled)) + styleDim.Render(strings.Repeat("░", barW-filled))
		rightLines = append(rightLines, fmt.Sprintf("%s  %s  %2d", peach.Render(statNames[i]), bar, v))
	}

	// ── portrait + stats side by side ───────────────────────────────
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

	// ── active quests ────────────────────────────────────────────────
	bodyLines = append(bodyLines, "")
	bodyLines = append(bodyLines, styleDetailLabel.Render("ACTIVE QUESTS"))

	questCount := 0
	for _, f := range m.features {
		if f.workerName != displayName && f.workerName != w.ID {
			continue
		}
		questCount++
		st := statusStyle(f.s.Status)
		desc := strings.TrimPrefix(f.s.Slug, f.s.Ticket+"-")
		bodyLines = append(bodyLines, fmt.Sprintf(
			"  %s  %-12s  %s  %s",
			st.Render(statusIcon(f.s.Status)),
			f.s.Ticket,
			styleDim.Render(truncate(desc, 24)),
			st.Render(f.s.Status),
		))
	}
	if questCount == 0 {
		bodyLines = append(bodyLines, styleDim.Render("  (no active quests)"))
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
