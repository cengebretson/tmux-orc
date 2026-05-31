package tui

import "github.com/charmbracelet/lipgloss"

const logo = `⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⣿⣿⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣤⣶⣧⣄⣉⣉⣠⣼⣶⣤⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⢰⣿⣿⣿⣿⡿⣿⣿⣿⣿⢿⣿⣿⣿⣿⡆⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⣼⣤⣤⣈⠙⠳⢄⣉⣋⡡⠞⠋⣁⣤⣤⣧⠀⠀⠀⠀⠀⠀⠀
⠀⢲⣶⣤⣄⡀⢀⣿⣄⠙⠿⣿⣦⣤⡿⢿⣤⣴⣿⠿⠋⣠⣿⠀⢀⣠⣤⣶⡖⠀
⠀⠀⠙⣿⠛⠇⢸⣿⣿⡟⠀⡄⢉⠉⢀⡀⠉⡉⢠⠀⢻⣿⣿⡇⠸⠛⣿⠋⠀⠀
⠀⠀⠀⠘⣷⠀⢸⡏⠻⣿⣤⣤⠂⣠⣿⣿⣄⠑⣤⣤⣿⠟⢹⡇⠀⣾⠃⠀⠀⠀
⠀⠀⠀⠀⠘⠀⢸⣿⡀⢀⠙⠻⢦⣌⣉⣉⣡⡴⠟⠋⡀⢀⣿⡇⠀⠃⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⢸⣿⣧⠈⠛⠂⠀⠉⠛⠛⠉⠀⠐⠛⠁⣼⣿⡇⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠸⣏⠀⣤⡶⠖⠛⠋⠉⠉⠙⠛⠲⢶⣤⠀⣹⠇⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⢹⣿⣶⣿⣿⣿⣿⣿⣿⣶⣿⡏⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠉⠉⠛⠛⠛⠛⠉⠉⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀`

// Catppuccin Mocha palette
const (
	crust   = "#11111b"
	mantle  = "#181825"
	base    = "#1e1e2e"
	surface0 = "#313244"
	surface1 = "#45475a"
	surface2 = "#585b70"
	overlay0 = "#6c7086"
	overlay1 = "#7f849c"
	subtext0 = "#a6adc8"
	subtext1 = "#bac2de"
	text     = "#cdd6f4"
	lavender = "#b4befe"
	blue     = "#89b4fa"
	sapphire = "#74c7ec"
	sky      = "#89dceb"
	teal     = "#94e2d5"
	green    = "#a6e3a1"
	yellow   = "#f9e2af"
	peach    = "#fab387"
	maroon   = "#eba0ac"
	red      = "#f38ba8"
	mauve    = "#cba6f7"
	pink     = "#f5c2e7"
	flamingo = "#f2cdcd"
)

var (
	// Base styles
	styleBase = lipgloss.NewStyle().
			Background(lipgloss.Color(base)).
			Foreground(lipgloss.Color(text))

	styleDim = lipgloss.NewStyle().
			Foreground(lipgloss.Color(overlay0))

	styleSubtext = lipgloss.NewStyle().
			Foreground(lipgloss.Color(subtext0))

	// Header
	styleHeader = lipgloss.NewStyle().
			Foreground(lipgloss.Color(mauve)).
			Bold(true)

	// Section titles
	styleSection = lipgloss.NewStyle().
			Foreground(lipgloss.Color(lavender)).
			Bold(true).
			MarginTop(1)

	// Table
	styleTableHeader = lipgloss.NewStyle().
				Foreground(lipgloss.Color(subtext0)).
				Bold(true)

	styleRowSelected = lipgloss.NewStyle().
				Background(lipgloss.Color(surface0)).
				Foreground(lipgloss.Color(text))

	styleRowNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color(text))

	// Status colors
	styleStatusReady = lipgloss.NewStyle().
				Foreground(lipgloss.Color(blue))

	styleStatusInProgress = lipgloss.NewStyle().
				Foreground(lipgloss.Color(mauve))

	styleStatusWaiting = lipgloss.NewStyle().
				Foreground(lipgloss.Color(yellow))

	styleStatusBlocked = lipgloss.NewStyle().
				Foreground(lipgloss.Color(red))

	styleStatusArchived = lipgloss.NewStyle().
				Foreground(lipgloss.Color(overlay0))

	styleStatusPending = lipgloss.NewStyle().
				Foreground(lipgloss.Color(peach))

	// Health indicators
	styleHealthOK = lipgloss.NewStyle().
			Foreground(lipgloss.Color(green))

	styleHealthWarn = lipgloss.NewStyle().
			Foreground(lipgloss.Color(yellow))

	styleHealthErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color(red))

	// Detail view
	styleDetailBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(surface1)).
			Padding(0, 1)

	styleDetailLabel = lipgloss.NewStyle().
				Foreground(lipgloss.Color(subtext0))

	styleDetailValue = lipgloss.NewStyle().
				Foreground(lipgloss.Color(text))

	styleDetailTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(mauve)).
				Bold(true)

	// File chips
	styleFileOK = lipgloss.NewStyle().
			Foreground(lipgloss.Color(green)).
			Background(lipgloss.Color(surface0)).
			Padding(0, 1)

	styleFileMissing = lipgloss.NewStyle().
				Foreground(lipgloss.Color(overlay0)).
				Background(lipgloss.Color(surface0)).
				Padding(0, 1)

	styleFileSelected = lipgloss.NewStyle().
				Foreground(lipgloss.Color(base)).
				Background(lipgloss.Color(mauve)).
				Padding(0, 1)

	// Help bar
	styleHelp = lipgloss.NewStyle().
			Foreground(lipgloss.Color(overlay0)).
			MarginTop(1)

	styleHelpKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color(subtext1))

	// Tmux status
	styleTmuxLive    = lipgloss.NewStyle().Foreground(lipgloss.Color(green))
	styleTmuxDead    = lipgloss.NewStyle().Foreground(lipgloss.Color(red))
	styleTmuxNone    = lipgloss.NewStyle().Foreground(lipgloss.Color(overlay0))

	// Border/divider
	styleDivider = lipgloss.NewStyle().
			Foreground(lipgloss.Color(surface1))

	// Section box — inner content area (no border; border added by styleSectionBox)
	styleBox = lipgloss.NewStyle().
			Padding(0, 1)
)

func statusStyle(status string) lipgloss.Style {
	switch status {
	case "ready":
		return styleStatusReady
	case "in_progress":
		return styleStatusInProgress
	case "waiting_for_human":
		return styleStatusWaiting
	case "blocked":
		return styleStatusBlocked
	case "archived":
		return styleStatusArchived
	case "pending":
		return styleStatusPending
	default:
		return styleSubtext
	}
}

func statusIcon(status string) string {
	switch status {
	case "ready":
		return "●"
	case "in_progress":
		return "▶"
	case "waiting_for_human":
		return "◐"
	case "blocked":
		return "✗"
	case "archived":
		return "✓"
	case "pending":
		return "○"
	default:
		return "·"
	}
}

func helpItem(key, desc string) string {
	return styleHelpKey.Render(key) + styleDim.Render(" "+desc)
}
