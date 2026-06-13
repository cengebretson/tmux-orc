package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderBrokenFeature builds the file-viewer content for a feature whose
// STATE.yaml could not be parsed: the parse error followed by the raw file so
// the user can spot the problem (bad indentation, a stray field) without
// leaving the TUI.
func renderBrokenFeature(row *featureRow) string {
	var b strings.Builder
	b.WriteString(styleHealthErr.Render("⚠ STATE.yaml could not be parsed") + "\n\n")
	if row.loadErr != nil {
		b.WriteString(styleDim.Render("error: ") + row.loadErr.Error() + "\n")
	}
	b.WriteString(styleDim.Render("path:  ") + filepath.Join(row.featureDir, "STATE.yaml") + "\n\n")

	raw, err := os.ReadFile(filepath.Join(row.featureDir, "STATE.yaml"))
	if err != nil {
		b.WriteString(styleHealthErr.Render("could not read file: " + err.Error()))
		return b.String()
	}
	b.WriteString(styleDivider.Render(strings.Repeat("─", 40)) + "\n")
	b.WriteString(string(raw))
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
		BorderForeground(lipgloss.Color(activeTheme.Palette.Surface1)).
		Width(innerW).
		Render(strings.Join(all, "\n"))
}

// drawBoxLabeled renders a rounded box with the title embedded in the top border.
func drawBoxLabeled(title string, contentLines []string, outerW int) string {
	return drawBoxLabeledWith(title, contentLines, outerW, activeTheme.Palette.Surface1)
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
